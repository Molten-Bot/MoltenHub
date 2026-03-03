package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"statocyst/internal/auth"
	"statocyst/internal/longpoll"
	"statocyst/internal/model"
	"statocyst/internal/store"
)

const (
	maxPullTimeoutMS     = 30000
	defaultPullTimeoutMS = 5000
)

var agentIDRegex = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

type Handler struct {
	store           *store.MemoryStore
	waiters         *longpoll.Waiters
	humanAuth       auth.HumanAuthProvider
	now             func() time.Time
	idFactory       func() (string, error)
	supabaseURL     string
	supabaseAnonKey string
}

func NewHandler(st *store.MemoryStore, waiters *longpoll.Waiters, humanAuth auth.HumanAuthProvider, supabaseURL, supabaseAnonKey string) *Handler {
	return &Handler{
		store:           st,
		waiters:         waiters,
		humanAuth:       humanAuth,
		now:             time.Now,
		idFactory:       newUUIDv7,
		supabaseURL:     strings.TrimSpace(supabaseURL),
		supabaseAnonKey: strings.TrimSpace(supabaseAnonKey),
	}
}

func NewRouter(handler *Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handler.handleHealthz)
	mux.HandleFunc("/healthz", handler.handleHealthz)
	mux.HandleFunc("/openapi.yaml", handler.handleOpenAPIYAML)
	mux.HandleFunc("/v1/ui/config", handler.handleUIConfig)
	mux.HandleFunc("/v1/me", handler.handleMe)
	mux.HandleFunc("/v1/me/orgs", handler.handleMyOrgs)
	mux.HandleFunc("/v1/orgs", handler.handleOrgs)
	mux.HandleFunc("/v1/orgs/", handler.handleOrgSubroutes)
	mux.HandleFunc("/v1/org-invites/", handler.handleOrgInvites)
	mux.HandleFunc("/v1/agents/register", handler.handleRegisterAgent)
	mux.HandleFunc("/v1/agents/", handler.handleAgentsSubroutes)
	mux.HandleFunc("/v1/org-trusts", handler.handleOrgTrusts)
	mux.HandleFunc("/v1/org-trusts/", handler.handleOrgTrustByID)
	mux.HandleFunc("/v1/agent-trusts", handler.handleAgentTrusts)
	mux.HandleFunc("/v1/agent-trusts/", handler.handleAgentTrustByID)
	mux.HandleFunc("/v1/messages/publish", handler.handlePublish)
	mux.HandleFunc("/v1/messages/pull", handler.handlePull)
	mux.HandleFunc("/", handler.handleUI)
	return mux
}

func (h *Handler) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeMethodNotAllowed(w)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, map[string]string{
		"error":   code,
		"message": message,
	})
}

func writeMethodNotAllowed(w http.ResponseWriter) {
	writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
}

func decodeJSON(r *http.Request, out any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(out)
}

func validateAgentID(agentID string) bool {
	return agentIDRegex.MatchString(agentID)
}

func parsePullTimeout(r *http.Request) (time.Duration, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("timeout_ms"))
	if raw == "" {
		return time.Duration(defaultPullTimeoutMS) * time.Millisecond, nil
	}
	ms, err := strconv.Atoi(raw)
	if err != nil {
		return 0, errors.New("timeout_ms must be an integer")
	}
	if ms < 0 || ms > maxPullTimeoutMS {
		return 0, errors.New("timeout_ms must be in range 0..30000")
	}
	return time.Duration(ms) * time.Millisecond, nil
}

func (h *Handler) authenticateHuman(r *http.Request) (model.Human, error) {
	identity, err := h.humanAuth.Authenticate(r)
	if err != nil {
		return model.Human{}, err
	}
	return h.store.UpsertHuman(identity.Provider, identity.Subject, identity.Email, h.now().UTC(), h.idFactory)
}

func (h *Handler) authenticateAgent(r *http.Request) (string, error) {
	token, err := auth.ExtractBearerToken(r.Header.Get("Authorization"))
	if err != nil {
		return "", err
	}
	tokenHash := auth.HashToken(token)
	return h.store.AgentIDForTokenHash(tokenHash)
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}
