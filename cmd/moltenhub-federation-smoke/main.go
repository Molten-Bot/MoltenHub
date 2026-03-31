package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	peerID       = "alpha-beta"
	peerSecret   = "local-federation-shared-secret"
	alphaPeerURL = "http://moltenhub-alpha:8080"
	betaPeerURL  = "http://moltenhub-beta:8080"
)

type runner struct {
	alphaBaseURL string
	betaBaseURL  string
	client       *http.Client

	alphaOrgID     string
	alphaOrgHandle string
	alphaToken     string
	alphaAgentUUID string
	alphaAgentURI  string

	betaOrgID     string
	betaOrgHandle string
	betaToken     string
	betaAgentUUID string
	betaAgentURI  string
}

type step struct {
	name string
	run  func(*runner) error
}

func main() {
	alphaBaseURL := flag.String("alpha-base-url", "http://127.0.0.1:18080", "Alpha MoltenHub base URL")
	betaBaseURL := flag.String("beta-base-url", "http://127.0.0.1:18081", "Beta MoltenHub base URL")
	flag.Parse()

	r := &runner{
		alphaBaseURL: strings.TrimRight(*alphaBaseURL, "/"),
		betaBaseURL:  strings.TrimRight(*betaBaseURL, "/"),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	steps := []step{
		{name: "Both health endpoints respond", run: (*runner).stepHealth},
		{name: "Alpha and Beta create orgs and agents", run: (*runner).stepCreateOrgsAndAgents},
		{name: "Alpha and Beta pair as peers", run: (*runner).stepPairPeers},
		{name: "Alpha and Beta trust each other's orgs and agents", run: (*runner).stepCreateRemoteTrusts},
		{name: "Alpha agent sends a message to Beta over the bridge", run: (*runner).stepAlphaToBeta},
		{name: "Beta agent sends a message back to Alpha over the bridge", run: (*runner).stepBetaToAlpha},
	}

	for _, st := range steps {
		fmt.Printf("RUN  %s\n", st.name)
		if err := st.run(r); err != nil {
			fmt.Fprintf(os.Stderr, "FAIL %s: %v\n", st.name, err)
			os.Exit(1)
		}
		fmt.Printf("PASS %s\n", st.name)
	}
}

func (r *runner) stepHealth() error {
	for _, target := range []struct {
		name    string
		baseURL string
	}{
		{name: "alpha", baseURL: r.alphaBaseURL},
		{name: "beta", baseURL: r.betaBaseURL},
	} {
		status, payload, err := r.requestJSON(target.baseURL, http.MethodGet, "/health", nil, nil)
		if err != nil {
			return err
		}
		if status != http.StatusOK {
			return fmt.Errorf("%s health expected 200, got %d payload=%v", target.name, status, payload)
		}
		if payload["status"] != "ok" {
			return fmt.Errorf("%s health expected ok, got %v", target.name, payload["status"])
		}
	}
	return nil
}

func (r *runner) stepCreateOrgsAndAgents() error {
	alphaOrgID, alphaOrgHandle, err := r.createOrg(r.alphaBaseURL, "alice", "alice@a.test", "launch-alpha", "Launch Alpha")
	if err != nil {
		return err
	}
	r.alphaOrgID = alphaOrgID
	r.alphaOrgHandle = alphaOrgHandle
	alphaToken, alphaAgentUUID, alphaAgentURI, err := r.createAgent(r.alphaBaseURL, "alice", "alice@a.test", alphaOrgID, "launch-agent-a")
	if err != nil {
		return err
	}
	r.alphaToken = alphaToken
	r.alphaAgentUUID = alphaAgentUUID
	r.alphaAgentURI = alphaAgentURI

	betaOrgID, betaOrgHandle, err := r.createOrg(r.betaBaseURL, "bob", "bob@b.test", "launch-beta", "Launch Beta")
	if err != nil {
		return err
	}
	r.betaOrgID = betaOrgID
	r.betaOrgHandle = betaOrgHandle
	betaToken, betaAgentUUID, betaAgentURI, err := r.createAgent(r.betaBaseURL, "bob", "bob@b.test", betaOrgID, "launch-agent-b")
	if err != nil {
		return err
	}
	r.betaToken = betaToken
	r.betaAgentUUID = betaAgentUUID
	r.betaAgentURI = betaAgentURI
	return nil
}

func (r *runner) stepPairPeers() error {
	if err := r.createPeer(r.alphaBaseURL, betaPeerURL, betaPeerURL); err != nil {
		return err
	}
	if err := r.createPeer(r.betaBaseURL, alphaPeerURL, alphaPeerURL); err != nil {
		return err
	}
	return nil
}

func (r *runner) stepCreateRemoteTrusts() error {
	if err := r.createRemoteOrgTrust(r.alphaBaseURL, r.alphaOrgID, r.betaOrgHandle); err != nil {
		return err
	}
	if err := r.createRemoteOrgTrust(r.betaBaseURL, r.betaOrgID, r.alphaOrgHandle); err != nil {
		return err
	}
	if err := r.createRemoteAgentTrust(r.alphaBaseURL, r.alphaAgentUUID, r.betaAgentURI); err != nil {
		return err
	}
	if err := r.createRemoteAgentTrust(r.betaBaseURL, r.betaAgentUUID, r.alphaAgentURI); err != nil {
		return err
	}

	capStatus, capPayload, err := r.requestJSON(r.alphaBaseURL, http.MethodGet, "/v1/agents/me/capabilities", map[string]string{
		"Authorization": "Bearer " + r.alphaToken,
	}, nil)
	if err != nil {
		return err
	}
	if capStatus != http.StatusOK {
		return fmt.Errorf("alpha capabilities expected 200, got %d payload=%v", capStatus, capPayload)
	}
	if !containsString(capPayload, "control_plane", "can_talk_to_uris", r.betaAgentURI) {
		return fmt.Errorf("alpha capabilities missing beta agent URI: payload=%v", capPayload)
	}
	return nil
}

func (r *runner) stepAlphaToBeta() error {
	return r.publishPullAck(r.alphaBaseURL, r.alphaToken, r.betaBaseURL, r.betaToken, r.betaAgentURI, r.alphaAgentURI, "federation alpha to beta")
}

func (r *runner) stepBetaToAlpha() error {
	return r.publishPullAck(r.betaBaseURL, r.betaToken, r.alphaBaseURL, r.alphaToken, r.alphaAgentURI, r.betaAgentURI, "federation beta to alpha")
}

func (r *runner) publishPullAck(senderBaseURL, senderToken, receiverBaseURL, receiverToken, toAgentURI, fromAgentURI, wantPayload string) error {
	pubStatus, pubPayload, err := r.requestJSON(senderBaseURL, http.MethodPost, "/v1/messages/publish", map[string]string{
		"Authorization": "Bearer " + senderToken,
	}, map[string]any{
		"to_agent_uri": toAgentURI,
		"content_type": "text/plain",
		"payload":      wantPayload,
	})
	if err != nil {
		return err
	}
	if pubStatus != http.StatusAccepted {
		return fmt.Errorf("publish expected 202, got %d payload=%v", pubStatus, pubPayload)
	}

	pullStatus, pullPayload, err := r.requestJSON(receiverBaseURL, http.MethodGet, "/v1/messages/pull?timeout_ms=1000", map[string]string{
		"Authorization": "Bearer " + receiverToken,
	}, nil)
	if err != nil {
		return err
	}
	if pullStatus != http.StatusOK {
		return fmt.Errorf("pull expected 200, got %d payload=%v", pullStatus, pullPayload)
	}
	message, err := requireObject(pullPayload, "message")
	if err != nil {
		return err
	}
	if asString(message, "payload") != wantPayload {
		return fmt.Errorf("expected payload %q, got %q", wantPayload, asString(message, "payload"))
	}
	if asString(message, "from_agent_uri") != fromAgentURI {
		return fmt.Errorf("expected from_agent_uri %q, got %q", fromAgentURI, asString(message, "from_agent_uri"))
	}
	if asString(message, "to_agent_uri") != toAgentURI {
		return fmt.Errorf("expected to_agent_uri %q, got %q", toAgentURI, asString(message, "to_agent_uri"))
	}

	delivery, err := requireObject(pullPayload, "delivery")
	if err != nil {
		return err
	}
	deliveryID := asString(delivery, "delivery_id")
	if deliveryID == "" {
		return fmt.Errorf("delivery_id missing from pull payload=%v", pullPayload)
	}
	ackStatus, ackPayload, err := r.requestJSON(receiverBaseURL, http.MethodPost, "/v1/messages/ack", map[string]string{
		"Authorization": "Bearer " + receiverToken,
	}, map[string]any{"delivery_id": deliveryID})
	if err != nil {
		return err
	}
	if ackStatus != http.StatusOK {
		return fmt.Errorf("ack expected 200, got %d payload=%v", ackStatus, ackPayload)
	}
	return nil
}

func (r *runner) createOrg(baseURL, humanID, email, handle, displayName string) (string, string, error) {
	if _, _, err := r.requestJSON(baseURL, http.MethodPatch, "/v1/me", humanHeaders(humanID, email), map[string]any{
		"handle": humanID,
	}); err != nil {
		return "", "", err
	}
	status, payload, err := r.requestJSON(baseURL, http.MethodPost, "/v1/orgs", humanHeaders(humanID, email), map[string]any{
		"handle":       handle,
		"display_name": displayName,
	})
	if err != nil {
		return "", "", err
	}
	if status != http.StatusCreated {
		return "", "", fmt.Errorf("create org expected 201, got %d payload=%v", status, payload)
	}
	org, err := requireObject(payload, "organization")
	if err != nil {
		return "", "", err
	}
	return asString(org, "org_id"), asString(org, "handle"), nil
}

func (r *runner) createAgent(baseURL, humanID, email, orgID, handle string) (string, string, string, error) {
	status, payload, err := r.requestJSON(baseURL, http.MethodPost, "/v1/agents/bind-tokens", humanHeaders(humanID, email), map[string]any{
		"org_id": orgID,
	})
	if err != nil {
		return "", "", "", err
	}
	if status != http.StatusCreated {
		return "", "", "", fmt.Errorf("create bind token expected 201, got %d payload=%v", status, payload)
	}
	bindToken := asString(payload, "bind_token")
	if bindToken == "" {
		return "", "", "", fmt.Errorf("bind_token missing from payload=%v", payload)
	}
	status, payload, err = r.requestJSON(baseURL, http.MethodPost, "/v1/agents/bind", nil, map[string]any{
		"bind_token": bindToken,
		"agent_id":   "temporary-" + handle,
	})
	if err != nil {
		return "", "", "", err
	}
	if status != http.StatusCreated {
		return "", "", "", fmt.Errorf("bind expected 201, got %d payload=%v", status, payload)
	}
	token := asString(payload, "token")
	if token == "" {
		return "", "", "", fmt.Errorf("token missing from payload=%v", payload)
	}
	status, payload, err = r.requestJSON(baseURL, http.MethodPatch, "/v1/agents/me", map[string]string{
		"Authorization": "Bearer " + token,
	}, map[string]any{
		"handle": handle,
	})
	if err != nil {
		return "", "", "", err
	}
	if status != http.StatusOK {
		return "", "", "", fmt.Errorf("finalize agent expected 200, got %d payload=%v", status, payload)
	}
	agent, err := requireObject(payload, "agent")
	if err != nil {
		return "", "", "", err
	}
	return token, asString(agent, "agent_uuid"), asString(agent, "uri"), nil
}

func (r *runner) createPeer(baseURL, canonicalBaseURL, deliveryBaseURL string) error {
	status, payload, err := r.requestJSON(baseURL, http.MethodPost, "/v1/admin/peers", adminHeaders(), map[string]any{
		"peer_id":            peerID,
		"canonical_base_url": canonicalBaseURL,
		"delivery_base_url":  deliveryBaseURL,
		"shared_secret":      peerSecret,
	})
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("create peer expected 201, got %d payload=%v", status, payload)
	}
	return nil
}

func (r *runner) createRemoteOrgTrust(baseURL, localOrgID, remoteOrgHandle string) error {
	status, payload, err := r.requestJSON(baseURL, http.MethodPost, "/v1/admin/remote-org-trusts", adminHeaders(), map[string]any{
		"local_org_id":      localOrgID,
		"peer_id":           peerID,
		"remote_org_handle": remoteOrgHandle,
	})
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("create remote org trust expected 201, got %d payload=%v", status, payload)
	}
	return nil
}

func (r *runner) createRemoteAgentTrust(baseURL, localAgentUUID, remoteAgentURI string) error {
	status, payload, err := r.requestJSON(baseURL, http.MethodPost, "/v1/admin/remote-agent-trusts", adminHeaders(), map[string]any{
		"local_agent_uuid": localAgentUUID,
		"peer_id":          peerID,
		"remote_agent_uri": remoteAgentURI,
	})
	if err != nil {
		return err
	}
	if status != http.StatusCreated {
		return fmt.Errorf("create remote agent trust expected 201, got %d payload=%v", status, payload)
	}
	return nil
}

func (r *runner) requestJSON(baseURL, method, path string, headers map[string]string, body any) (int, map[string]any, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		bodyReader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, strings.TrimRight(baseURL, "/")+path, bodyReader)
	if err != nil {
		return 0, nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := r.client.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, nil, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return resp.StatusCode, map[string]any{}, nil
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return 0, nil, fmt.Errorf("decode %s %s response: %w body=%s", method, path, err, string(data))
	}
	return resp.StatusCode, payload, nil
}

func humanHeaders(humanID, email string) map[string]string {
	return map[string]string{
		"X-Human-Id":    humanID,
		"X-Human-Email": email,
	}
}

func adminHeaders() map[string]string {
	return humanHeaders("ops", "ops@molten.bot")
}

func requireObject(payload map[string]any, key string) (map[string]any, error) {
	obj, ok := payload[key].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("expected %s object, got %T payload=%v", key, payload[key], payload)
	}
	return obj, nil
}

func asString(payload map[string]any, key string) string {
	value, _ := payload[key].(string)
	return value
}

func containsString(payload map[string]any, topKey, nestedKey, want string) bool {
	top, ok := payload[topKey].(map[string]any)
	if !ok {
		return false
	}
	items, ok := top[nestedKey].([]any)
	if !ok {
		return false
	}
	for _, item := range items {
		if got, ok := item.(string); ok && got == want {
			return true
		}
	}
	return false
}
