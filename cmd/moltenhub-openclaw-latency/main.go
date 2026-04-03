package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type apiResponse struct {
	Code int
	Raw  []byte
	JSON map[string]any
}

type sample struct {
	SampleIndex int
	PublishMS   float64
	DeliveryMS  float64
	EndToEndMS  float64
}

type metricStats struct {
	Count int
	Min   float64
	P50   float64
	P95   float64
	P99   float64
	Avg   float64
	Max   float64
}

type scenario struct {
	Name       string
	SenderMode string
	RecvMode   string
}

type scenarioResult struct {
	Name      string
	Sender    string
	Receiver  string
	Publish   metricStats
	Delivery  metricStats
	EndToEnd  metricStats
	Samples   []sample
	Failures  []string
	Succeeded int
}

type runner struct {
	baseURL       string
	httpClient    *http.Client
	iterations    int
	sampleRetries int
	pullTimeoutMS int
	wsReadTimeout time.Duration
	verbose       bool
}

type wsClient struct {
	conn  *websocket.Conn
	token string
	base  string
}

type agentIdentity struct {
	AgentID   string
	AgentUUID string
}

type reportContext struct {
	BaseURL      string
	GeneratedAt  time.Time
	AgentA       agentIdentity
	AgentB       agentIdentity
	HealthStatus string
	BootStatus   string
	StartupMode  string
	StateHealthy string
	QueueHealthy string
	Iterations   int
}

func main() {
	var cfg struct {
		baseURL        string
		agentAToken    string
		agentBToken    string
		iterations     int
		sampleRetries  int
		pullTimeoutMS  int
		httpTimeoutSec int
		wsTimeoutSec   int
		maxP95MS       int
		reportPath     string
		verbose        bool
	}

	flag.StringVar(&cfg.baseURL, "base-url", "https://na.hub.molten-qa.site", "MoltenHub base URL")
	flag.StringVar(&cfg.agentAToken, "agent-a-token", "", "Agent A bearer token")
	flag.StringVar(&cfg.agentBToken, "agent-b-token", "", "Agent B bearer token")
	flag.IntVar(&cfg.iterations, "iterations", 6, "Number of samples per scenario")
	flag.IntVar(&cfg.sampleRetries, "sample-retries", 1, "Retries per failed sample")
	flag.IntVar(&cfg.pullTimeoutMS, "pull-timeout-ms", 5000, "Pull timeout in milliseconds")
	flag.IntVar(&cfg.httpTimeoutSec, "http-timeout-sec", 45, "HTTP timeout in seconds")
	flag.IntVar(&cfg.wsTimeoutSec, "ws-timeout-sec", 10, "Websocket read timeout in seconds")
	flag.IntVar(&cfg.maxP95MS, "max-p95-ms", 0, "Fail when delivery p95 exceeds threshold (0 disables)")
	flag.StringVar(&cfg.reportPath, "report-path", "", "Optional markdown report output path")
	flag.BoolVar(&cfg.verbose, "verbose", false, "Print per-sample timings")
	flag.Parse()

	if strings.TrimSpace(cfg.agentAToken) == "" || strings.TrimSpace(cfg.agentBToken) == "" {
		fmt.Fprintln(os.Stderr, "missing required args: -agent-a-token and -agent-b-token")
		os.Exit(2)
	}
	if cfg.iterations <= 0 {
		fmt.Fprintln(os.Stderr, "-iterations must be > 0")
		os.Exit(2)
	}
	if cfg.sampleRetries < 0 {
		fmt.Fprintln(os.Stderr, "-sample-retries must be >= 0")
		os.Exit(2)
	}
	if cfg.pullTimeoutMS <= 0 {
		fmt.Fprintln(os.Stderr, "-pull-timeout-ms must be > 0")
		os.Exit(2)
	}
	if cfg.httpTimeoutSec <= 0 {
		fmt.Fprintln(os.Stderr, "-http-timeout-sec must be > 0")
		os.Exit(2)
	}
	if cfg.wsTimeoutSec <= 0 {
		fmt.Fprintln(os.Stderr, "-ws-timeout-sec must be > 0")
		os.Exit(2)
	}

	r := &runner{
		baseURL: strings.TrimRight(strings.TrimSpace(cfg.baseURL), "/"),
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.httpTimeoutSec) * time.Second,
		},
		iterations:    cfg.iterations,
		sampleRetries: cfg.sampleRetries,
		pullTimeoutMS: cfg.pullTimeoutMS,
		wsReadTimeout: time.Duration(cfg.wsTimeoutSec) * time.Second,
		verbose:       cfg.verbose,
	}

	agentA, err := r.whoAmI(cfg.agentAToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve agent A identity: %v\n", err)
		os.Exit(1)
	}
	agentB, err := r.whoAmI(cfg.agentBToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "resolve agent B identity: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("agent_a id=%s uuid=%s\n", agentA.AgentID, agentA.AgentUUID)
	fmt.Printf("agent_b id=%s uuid=%s\n", agentB.AgentID, agentB.AgentUUID)

	if err := r.registerPlugin(cfg.agentAToken, "a"); err != nil {
		fmt.Fprintf(os.Stderr, "register plugin for agent A failed: %v\n", err)
		os.Exit(1)
	}
	if err := r.registerPlugin(cfg.agentBToken, "b"); err != nil {
		fmt.Fprintf(os.Stderr, "register plugin for agent B failed: %v\n", err)
		os.Exit(1)
	}

	health, _ := r.healthSnapshot()

	scenarios := []scenario{
		{Name: "http->http", SenderMode: "http", RecvMode: "http"},
		{Name: "http->ws", SenderMode: "http", RecvMode: "ws"},
		{Name: "ws->http", SenderMode: "ws", RecvMode: "http"},
		{Name: "ws->ws", SenderMode: "ws", RecvMode: "ws"},
	}

	results := make([]scenarioResult, 0, len(scenarios))
	for _, sc := range scenarios {
		fmt.Printf("== scenario %s ==\n", sc.Name)
		result, runErr := r.runScenario(sc, cfg.agentAToken, cfg.agentBToken, agentB.AgentUUID)
		if runErr != nil {
			fmt.Fprintf(os.Stderr, "scenario %s failures: %v\n", sc.Name, runErr)
		}
		results = append(results, result)

		fmt.Printf(
			"summary %s success=%d failures=%d delivery_ms(min=%.3f p50=%.3f p95=%.3f p99=%.3f avg=%.3f max=%.3f)\n",
			result.Name,
			result.Succeeded,
			len(result.Failures),
			result.Delivery.Min,
			result.Delivery.P50,
			result.Delivery.P95,
			result.Delivery.P99,
			result.Delivery.Avg,
			result.Delivery.Max,
		)
	}

	ctx := reportContext{
		BaseURL:      r.baseURL,
		GeneratedAt:  time.Now().UTC(),
		AgentA:       agentA,
		AgentB:       agentB,
		Iterations:   cfg.iterations,
		HealthStatus: sp(health, "status"),
		BootStatus:   sp(health, "boot_status"),
		StartupMode:  sp(health, "storage", "startup_mode"),
		StateHealthy: sp(health, "storage", "state", "healthy"),
		QueueHealthy: sp(health, "storage", "queue", "healthy"),
	}

	if strings.TrimSpace(cfg.reportPath) != "" {
		if err := writeMarkdownReport(cfg.reportPath, ctx, results); err != nil {
			fmt.Fprintf(os.Stderr, "write markdown report: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("wrote markdown report: %s\n", cfg.reportPath)
	}

	exitCode := 0
	for _, result := range results {
		if result.Succeeded == 0 || len(result.Failures) > 0 {
			exitCode = 1
		}
		if cfg.maxP95MS > 0 && result.Delivery.Count > 0 && result.Delivery.P95 > float64(cfg.maxP95MS) {
			fmt.Fprintf(
				os.Stderr,
				"scenario %s failed p95 threshold: p95=%.3f threshold=%d\n",
				result.Name,
				result.Delivery.P95,
				cfg.maxP95MS,
			)
			exitCode = 1
		}
	}

	if exitCode != 0 {
		fmt.Fprintln(os.Stderr, "OpenClaw transport synthetic check failed")
		os.Exit(exitCode)
	}

	fmt.Println("OpenClaw transport synthetic check passed")
}

func (r *runner) runScenario(sc scenario, senderToken, receiverToken, receiverUUID string) (scenarioResult, error) {
	result := scenarioResult{Name: sc.Name, Sender: sc.SenderMode, Receiver: sc.RecvMode}

	if err := r.drainQueue(senderToken); err != nil {
		return result, err
	}
	if err := r.drainQueue(receiverToken); err != nil {
		return result, err
	}

	var senderWS *wsClient
	var receiverWS *wsClient
	var err error

	if sc.SenderMode == "ws" {
		senderWS, err = r.openWS(senderToken, "sender-"+slug(sc.Name))
		if err != nil {
			return result, err
		}
	}
	if sc.RecvMode == "ws" {
		receiverWS, err = r.openWS(receiverToken, "receiver-"+slug(sc.Name))
		if err != nil {
			if senderWS != nil {
				senderWS.close()
			}
			return result, err
		}
	}

	defer func() {
		if senderWS != nil {
			senderWS.close()
		}
		if receiverWS != nil {
			receiverWS.close()
		}
	}()

	stamp := time.Now().UTC().Format("20060102T150405Z")
	for i := 1; i <= r.iterations; i++ {
		var sampleErr error
		for attempt := 0; attempt <= r.sampleRetries; attempt++ {
			payload := fmt.Sprintf("latency-%s-%s-%02d-%d", slug(sc.Name), stamp, i, time.Now().UnixNano())
			s, err := r.runSample(sc, senderToken, receiverToken, receiverUUID, payload, senderWS, receiverWS, i)
			if err == nil {
				result.Samples = append(result.Samples, s)
				result.Succeeded++
				sampleErr = nil
				break
			}
			sampleErr = err
		}
		if sampleErr != nil {
			result.Failures = append(result.Failures, fmt.Sprintf("sample %02d: %v", i, sampleErr))
		}
	}

	publishValues := make([]float64, 0, len(result.Samples))
	deliveryValues := make([]float64, 0, len(result.Samples))
	e2eValues := make([]float64, 0, len(result.Samples))
	for _, s := range result.Samples {
		publishValues = append(publishValues, s.PublishMS)
		deliveryValues = append(deliveryValues, s.DeliveryMS)
		e2eValues = append(e2eValues, s.EndToEndMS)
	}
	result.Publish = computeStats(publishValues)
	result.Delivery = computeStats(deliveryValues)
	result.EndToEnd = computeStats(e2eValues)

	if len(result.Failures) > 0 {
		return result, errors.New(strings.Join(result.Failures, "; "))
	}
	return result, nil
}

func (r *runner) runSample(
	sc scenario,
	senderToken, receiverToken, receiverUUID, payload string,
	senderWS, receiverWS *wsClient,
	sampleIndex int,
) (sample, error) {
	start := time.Now()

	var (
		messageID string
		publishAt time.Time
		publishMS float64
		err       error
	)

	if sc.SenderMode == "http" {
		messageID, publishAt, publishMS, err = r.publishHTTP(senderToken, receiverUUID, payload)
	} else {
		if senderWS == nil {
			return sample{}, fmt.Errorf("sender websocket not initialized")
		}
		messageID, publishAt, publishMS, err = r.publishWS(senderWS, receiverUUID, payload)
	}
	if err != nil {
		return sample{}, err
	}

	var receivedAt time.Time
	if sc.RecvMode == "http" {
		receivedAt, err = r.receiveHTTP(receiverToken, messageID)
	} else {
		if receiverWS == nil {
			return sample{}, fmt.Errorf("receiver websocket not initialized")
		}
		receivedAt, err = r.receiveWS(receiverWS, messageID)
	}
	if err != nil {
		return sample{}, err
	}

	s := sample{
		SampleIndex: sampleIndex,
		PublishMS:   publishMS,
		DeliveryMS:  float64(receivedAt.Sub(publishAt).Microseconds()) / 1000.0,
		EndToEndMS:  float64(receivedAt.Sub(start).Microseconds()) / 1000.0,
	}

	if r.verbose {
		fmt.Printf(
			"sample=%02d mode=%s->%s publish_ms=%.3f delivery_ms=%.3f e2e_ms=%.3f\n",
			sampleIndex,
			sc.SenderMode,
			sc.RecvMode,
			s.PublishMS,
			s.DeliveryMS,
			s.EndToEndMS,
		)
	}

	return s, nil
}

func (r *runner) whoAmI(token string) (agentIdentity, error) {
	resp, err := r.callJSON(http.MethodGet, r.baseURL+"/v1/agents/me", token, nil)
	if err != nil {
		return agentIdentity{}, err
	}
	if resp.Code != http.StatusOK {
		return agentIdentity{}, fmt.Errorf("/v1/agents/me status=%d body=%s", resp.Code, strings.TrimSpace(string(resp.Raw)))
	}

	agentID := sp(resp.JSON, "result", "agent", "agent_id")
	if agentID == "" {
		agentID = sp(resp.JSON, "agent", "agent_id")
	}
	agentUUID := sp(resp.JSON, "result", "agent", "agent_uuid")
	if agentUUID == "" {
		agentUUID = sp(resp.JSON, "agent", "agent_uuid")
	}
	if agentID == "" || agentUUID == "" {
		return agentIdentity{}, fmt.Errorf("missing agent_id/agent_uuid in /v1/agents/me body=%s", strings.TrimSpace(string(resp.Raw)))
	}

	return agentIdentity{AgentID: agentID, AgentUUID: agentUUID}, nil
}

func (r *runner) registerPlugin(token, suffix string) error {
	payload := map[string]any{
		"plugin_id":    fmt.Sprintf("synthetic-transport-%s", suffix),
		"package":      "@moltenbot/openclaw-plugin-moltenhub",
		"transport":    "websocket",
		"session_mode": "dedicated",
		"session_key":  "synthetic-main",
	}
	resp, err := r.callJSON(http.MethodPost, r.baseURL+"/v1/openclaw/messages/register-plugin", token, payload)
	if err != nil {
		return err
	}
	if resp.Code != http.StatusOK {
		return fmt.Errorf("register-plugin status=%d body=%s", resp.Code, strings.TrimSpace(string(resp.Raw)))
	}

	result := runtimeResult(resp.JSON)
	if got := sp(result, "plugin", "transport"); got != "websocket" {
		return fmt.Errorf("register-plugin did not persist websocket transport, got=%q", got)
	}
	if got := sp(result, "agent", "metadata", "agent_type"); got != "openclaw" {
		return fmt.Errorf("register-plugin did not set metadata.agent_type=openclaw, got=%q", got)
	}
	return nil
}

func (r *runner) publishHTTP(token, toAgentUUID, text string) (string, time.Time, float64, error) {
	payload := map[string]any{
		"to_agent_uuid": toAgentUUID,
		"message": map[string]any{
			"kind": "text_message",
			"text": text,
		},
	}
	start := time.Now()
	resp, err := r.callJSON(http.MethodPost, r.baseURL+"/v1/openclaw/messages/publish", token, payload)
	finished := time.Now()
	if err != nil {
		return "", time.Time{}, 0, err
	}
	if resp.Code != http.StatusAccepted {
		return "", time.Time{}, 0, fmt.Errorf("publish http status=%d body=%s", resp.Code, strings.TrimSpace(string(resp.Raw)))
	}

	result := runtimeResult(resp.JSON)
	messageID := sp(result, "message_id")
	if messageID == "" {
		messageID = sp(result, "message", "message_id")
	}
	if messageID == "" {
		return "", time.Time{}, 0, fmt.Errorf("publish http missing message_id body=%s", strings.TrimSpace(string(resp.Raw)))
	}
	return messageID, finished, float64(finished.Sub(start).Microseconds()) / 1000.0, nil
}

func (r *runner) publishWS(sender *wsClient, toAgentUUID, text string) (string, time.Time, float64, error) {
	requestID := fmt.Sprintf("ws-publish-%d", time.Now().UnixNano())
	frame := map[string]any{
		"type":          "publish",
		"request_id":    requestID,
		"to_agent_uuid": toAgentUUID,
		"message": map[string]any{
			"kind": "text_message",
			"text": text,
		},
	}

	start := time.Now()
	if err := sender.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return "", time.Time{}, 0, fmt.Errorf("set ws write deadline: %w", err)
	}
	if err := sender.conn.WriteJSON(frame); err != nil {
		return "", time.Time{}, 0, fmt.Errorf("write ws publish frame: %w", err)
	}

	deadline := time.Now().Add(r.wsReadTimeout)
	for time.Now().Before(deadline) {
		evt, err := readWSJSON(sender.conn, time.Until(deadline))
		if err != nil {
			return "", time.Time{}, 0, err
		}
		if readStringPath(evt, "type") == "delivery" {
			_, deliveryID, ok := extractDelivery(evt)
			if ok && deliveryID != "" {
				_ = r.ackBestEffort(sender.token, deliveryID)
			}
			continue
		}
		if readStringPath(evt, "type") != "response" || readStringPath(evt, "request_id") != requestID {
			continue
		}
		if readStringPath(evt, "ok") != "true" || readStringPath(evt, "status") != "202" {
			return "", time.Time{}, 0, fmt.Errorf("ws publish rejected payload=%v", evt)
		}
		msgID := sp(runtimeResult(evt), "message_id")
		if msgID == "" {
			msgID = sp(runtimeResult(evt), "message", "message_id")
		}
		if msgID == "" {
			return "", time.Time{}, 0, fmt.Errorf("ws publish missing message_id payload=%v", evt)
		}
		finished := time.Now()
		return msgID, finished, float64(finished.Sub(start).Microseconds()) / 1000.0, nil
	}
	return "", time.Time{}, 0, fmt.Errorf("timed out waiting for ws publish response")
}

func (r *runner) receiveHTTP(token, expectedMessageID string) (time.Time, error) {
	for attempts := 0; attempts < 16; attempts++ {
		path := fmt.Sprintf("%s/v1/openclaw/messages/pull?timeout_ms=%d", r.baseURL, r.pullTimeoutMS)
		resp, err := r.callJSON(http.MethodGet, path, token, nil)
		if err != nil {
			continue
		}
		if resp.Code == http.StatusNoContent {
			continue
		}
		if resp.Code != http.StatusOK {
			return time.Time{}, fmt.Errorf("pull http status=%d body=%s", resp.Code, strings.TrimSpace(string(resp.Raw)))
		}

		msgID := sp(runtimeResult(resp.JSON), "message", "message_id")
		if msgID == "" {
			msgID = sp(runtimeResult(resp.JSON), "message_id")
		}
		deliveryID := sp(runtimeResult(resp.JSON), "delivery", "delivery_id")
		if deliveryID != "" {
			_ = r.ackBestEffort(token, deliveryID)
		}
		if msgID == expectedMessageID {
			return time.Now(), nil
		}
	}
	return time.Time{}, fmt.Errorf("did not receive expected message_id=%s via HTTP", expectedMessageID)
}

func (r *runner) receiveWS(receiver *wsClient, expectedMessageID string) (time.Time, error) {
	deadline := time.Now().Add(time.Duration(r.pullTimeoutMS) * time.Millisecond * 3)
	for time.Now().Before(deadline) {
		evt, err := readWSJSON(receiver.conn, time.Until(deadline))
		if err != nil {
			return time.Time{}, err
		}

		msgID, deliveryID, ok := extractDelivery(evt)
		if !ok {
			continue
		}
		if deliveryID != "" {
			_ = r.ackBestEffort(receiver.token, deliveryID)
		}
		if msgID == expectedMessageID {
			return time.Now(), nil
		}
	}
	return time.Time{}, fmt.Errorf("did not receive expected message_id=%s via websocket", expectedMessageID)
}

func (r *runner) openWS(token, sessionKey string) (*wsClient, error) {
	base, err := url.Parse(r.baseURL)
	if err != nil {
		return nil, err
	}
	switch strings.ToLower(strings.TrimSpace(base.Scheme)) {
	case "https":
		base.Scheme = "wss"
	case "http":
		base.Scheme = "ws"
	default:
		return nil, fmt.Errorf("unsupported base url scheme %q", base.Scheme)
	}
	base.Path = "/v1/openclaw/messages/ws"
	q := base.Query()
	q.Set("session_key", sessionKey)
	base.RawQuery = q.Encode()

	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+token)
	conn, resp, err := websocket.DefaultDialer.Dial(base.String(), headers)
	if err != nil {
		statusCode := 0
		if resp != nil {
			statusCode = resp.StatusCode
		}
		return nil, fmt.Errorf("websocket dial failed status=%d err=%w", statusCode, err)
	}

	first, err := readWSJSON(conn, r.wsReadTimeout)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if readStringPath(first, "type") != "session_ready" {
		_ = conn.Close()
		return nil, fmt.Errorf("expected session_ready frame got=%v", first)
	}

	return &wsClient{conn: conn, token: token, base: r.baseURL}, nil
}

func (w *wsClient) close() {
	if w == nil || w.conn == nil {
		return
	}
	_ = w.conn.Close()
	w.conn = nil
}

func (r *runner) drainQueue(token string) error {
	for i := 0; i < 128; i++ {
		resp, err := r.callJSON(http.MethodGet, r.baseURL+"/v1/openclaw/messages/pull?timeout_ms=0", token, nil)
		if err != nil {
			return err
		}
		if resp.Code == http.StatusNoContent {
			return nil
		}
		if resp.Code != http.StatusOK {
			return fmt.Errorf("drain queue pull status=%d body=%s", resp.Code, strings.TrimSpace(string(resp.Raw)))
		}
		deliveryID := sp(runtimeResult(resp.JSON), "delivery", "delivery_id")
		if deliveryID != "" {
			_ = r.ackBestEffort(token, deliveryID)
		}
	}
	return fmt.Errorf("drain queue exceeded maximum attempts")
}

func (r *runner) ackBestEffort(token, deliveryID string) error {
	resp, err := r.callJSON(http.MethodPost, r.baseURL+"/v1/openclaw/messages/ack", token, map[string]any{"delivery_id": deliveryID})
	if err != nil {
		return err
	}
	if resp.Code == http.StatusOK {
		return nil
	}
	if resp.Code == http.StatusNotFound {
		errCode := sp(resp.JSON, "error")
		if errCode == "unknown_delivery" {
			return nil
		}
	}
	return fmt.Errorf("ack status=%d body=%s", resp.Code, strings.TrimSpace(string(resp.Raw)))
}

func (r *runner) healthSnapshot() (map[string]any, error) {
	resp, err := r.callJSON(http.MethodGet, r.baseURL+"/health", "", nil)
	if err != nil {
		return nil, err
	}
	if resp.Code != http.StatusOK {
		return map[string]any{}, nil
	}
	return resp.JSON, nil
}

func (r *runner) callJSON(method, endpoint, token string, payload any) (*apiResponse, error) {
	var body io.Reader
	if payload != nil {
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		body = bytes.NewReader(raw)
	}

	req, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	out := &apiResponse{Code: resp.StatusCode, Raw: raw, JSON: map[string]any{}}
	_ = json.Unmarshal(raw, &out.JSON)
	return out, nil
}

func runtimeResult(payload map[string]any) map[string]any {
	if payload == nil {
		return map[string]any{}
	}
	result, ok := payload["result"].(map[string]any)
	if ok {
		return result
	}
	return payload
}

func extractDelivery(evt map[string]any) (messageID string, deliveryID string, ok bool) {
	if evt == nil {
		return "", "", false
	}
	if sp(evt, "type") == "delivery" {
		result := runtimeResult(evt)
		messageID = sp(result, "message", "message_id")
		if messageID == "" {
			messageID = sp(result, "message_id")
		}
		deliveryID = sp(result, "delivery", "delivery_id")
		return messageID, deliveryID, messageID != ""
	}
	if sp(evt, "type") == "response" && sp(evt, "ok") == "true" && sp(evt, "status") == "200" {
		result := runtimeResult(evt)
		messageID = sp(result, "message", "message_id")
		if messageID == "" {
			messageID = sp(result, "message_id")
		}
		deliveryID = sp(result, "delivery", "delivery_id")
		if messageID != "" || deliveryID != "" {
			return messageID, deliveryID, true
		}
	}
	return "", "", false
}

func readWSJSON(conn *websocket.Conn, timeout time.Duration) (map[string]any, error) {
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	if err := conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := conn.ReadJSON(&payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func computeStats(values []float64) metricStats {
	if len(values) == 0 {
		return metricStats{}
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)

	sum := 0.0
	for _, value := range sorted {
		sum += value
	}

	return metricStats{
		Count: len(sorted),
		Min:   sorted[0],
		P50:   percentile(sorted, 50),
		P95:   percentile(sorted, 95),
		P99:   percentile(sorted, 99),
		Avg:   sum / float64(len(sorted)),
		Max:   sorted[len(sorted)-1],
	}
}

func percentile(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if p <= 0 {
		return sorted[0]
	}
	if p >= 100 {
		return sorted[len(sorted)-1]
	}
	rank := int(math.Ceil((float64(p) / 100.0) * float64(len(sorted))))
	if rank < 1 {
		rank = 1
	}
	if rank > len(sorted) {
		rank = len(sorted)
	}
	return sorted[rank-1]
}

func writeMarkdownReport(path string, ctx reportContext, results []scenarioResult) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}

	var b strings.Builder
	b.WriteString("# OpenClaw Transport Latency Report\n\n")
	b.WriteString(fmt.Sprintf("- Generated (UTC): `%s`\n", ctx.GeneratedAt.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("- Base URL: `%s`\n", ctx.BaseURL))
	b.WriteString(fmt.Sprintf("- Iterations per scenario: `%d`\n", ctx.Iterations))
	b.WriteString(fmt.Sprintf("- Agent A: `%s`\n", ctx.AgentA.AgentID))
	b.WriteString(fmt.Sprintf("- Agent B: `%s`\n", ctx.AgentB.AgentID))
	b.WriteString(fmt.Sprintf("- Health: status=`%s` boot_status=`%s` startup_mode=`%s` state_healthy=`%s` queue_healthy=`%s`\n\n", ctx.HealthStatus, ctx.BootStatus, ctx.StartupMode, ctx.StateHealthy, ctx.QueueHealthy))

	b.WriteString("## Scenario Summary\n\n")
	b.WriteString("| Scenario | Successful Samples | Failed Samples | Delivery p50 (ms) | Delivery p95 (ms) | Delivery max (ms) | End-to-end p95 (ms) |\n")
	b.WriteString("|---|---:|---:|---:|---:|---:|---:|\n")
	for _, result := range results {
		b.WriteString(fmt.Sprintf(
			"| `%s` | %d | %d | %.3f | %.3f | %.3f | %.3f |\n",
			result.Name,
			result.Succeeded,
			len(result.Failures),
			result.Delivery.P50,
			result.Delivery.P95,
			result.Delivery.Max,
			result.EndToEnd.P95,
		))
	}

	b.WriteString("\n## Metric Details\n\n")
	for _, result := range results {
		b.WriteString(fmt.Sprintf("### %s\n\n", result.Name))
		b.WriteString("| Metric | count | min | p50 | p95 | p99 | avg | max |\n")
		b.WriteString("|---|---:|---:|---:|---:|---:|---:|---:|\n")
		b.WriteString(fmt.Sprintf(
			"| publish_rtt_ms | %d | %.3f | %.3f | %.3f | %.3f | %.3f | %.3f |\n",
			result.Publish.Count,
			result.Publish.Min,
			result.Publish.P50,
			result.Publish.P95,
			result.Publish.P99,
			result.Publish.Avg,
			result.Publish.Max,
		))
		b.WriteString(fmt.Sprintf(
			"| delivery_ms | %d | %.3f | %.3f | %.3f | %.3f | %.3f | %.3f |\n",
			result.Delivery.Count,
			result.Delivery.Min,
			result.Delivery.P50,
			result.Delivery.P95,
			result.Delivery.P99,
			result.Delivery.Avg,
			result.Delivery.Max,
		))
		b.WriteString(fmt.Sprintf(
			"| end_to_end_ms | %d | %.3f | %.3f | %.3f | %.3f | %.3f | %.3f |\n",
			result.EndToEnd.Count,
			result.EndToEnd.Min,
			result.EndToEnd.P50,
			result.EndToEnd.P95,
			result.EndToEnd.P99,
			result.EndToEnd.Avg,
			result.EndToEnd.Max,
		))
		if len(result.Failures) > 0 {
			b.WriteString("\nFailed samples:\n")
			for _, failure := range result.Failures {
				b.WriteString(fmt.Sprintf("- %s\n", failure))
			}
		}
		b.WriteString("\n")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func readStringPath(root map[string]any, path ...string) string {
	current := any(root)
	for _, part := range path {
		object, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		next, ok := object[part]
		if !ok {
			return ""
		}
		current = next
	}
	return normalizeString(current)
}

func sp(root map[string]any, path ...string) string {
	return readStringPath(root, path...)
}

func normalizeString(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case float64:
		if math.Mod(typed, 1) == 0 {
			return fmt.Sprintf("%.0f", typed)
		}
		return fmt.Sprintf("%f", typed)
	default:
		return ""
	}
}

func slug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "->", "-to-")
	if value == "" {
		return "x"
	}
	return value
}
