package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type apiContractSuite struct {
	Cases []apiContractCase `json:"cases"`
}

type apiContractCase struct {
	Name               string            `json:"name"`
	Setup              string            `json:"setup,omitempty"`
	Method             string            `json:"method"`
	Path               string            `json:"path"`
	Headers            map[string]string `json:"headers,omitempty"`
	Body               json.RawMessage   `json:"body,omitempty"`
	ExpectedStatus     int               `json:"expected_status"`
	ExpectedError      string            `json:"expected_error,omitempty"`
	RequiredJSONPaths  []string          `json:"required_json_paths,omitempty"`
	ForbiddenJSONPaths []string          `json:"forbidden_json_paths,omitempty"`
	ExpectedJSONValues map[string]any    `json:"expected_json_values,omitempty"`
}

func TestAPIContractFixtures(t *testing.T) {
	files, err := filepath.Glob("testdata/api_contract/*.json")
	if err != nil {
		t.Fatalf("glob contract fixtures: %v", err)
	}
	if len(files) == 0 {
		t.Fatalf("no contract fixtures found in testdata/api_contract")
	}

	for _, file := range files {
		var suite apiContractSuite
		raw, err := os.ReadFile(filepath.Clean(file))
		if err != nil {
			t.Fatalf("read fixture %s: %v", file, err)
		}
		if err := json.Unmarshal(raw, &suite); err != nil {
			t.Fatalf("decode fixture %s: %v", file, err)
		}

		for _, tc := range suite.Cases {
			tc := tc
			t.Run(filepath.Base(file)+"/"+tc.Name, func(t *testing.T) {
				router := newTestRouter()
				vars := map[string]string{}
				applyAPIContractSetup(t, router, tc.Setup, vars)

				path := substituteFixtureVars(tc.Path, vars)
				headers := substituteFixtureVarsMap(tc.Headers, vars)
				body := parseFixtureBody(t, tc.Body, vars)

				resp := doJSONRequest(t, router, tc.Method, path, body, headers)
				if resp.Code != tc.ExpectedStatus {
					t.Fatalf("expected status %d, got %d body=%s", tc.ExpectedStatus, resp.Code, resp.Body.String())
				}

				needsJSONChecks := tc.ExpectedError != "" || len(tc.RequiredJSONPaths) > 0 || len(tc.ForbiddenJSONPaths) > 0 || len(tc.ExpectedJSONValues) > 0
				if !needsJSONChecks {
					return
				}

				payload := decodeJSONMap(t, resp.Body.Bytes())

				if tc.ExpectedError != "" {
					gotError, _ := payload["error"].(string)
					if gotError != tc.ExpectedError {
						t.Fatalf("expected error %q, got %q payload=%v", tc.ExpectedError, gotError, payload)
					}
				}

				for _, path := range tc.RequiredJSONPaths {
					got, ok := lookupJSONPath(payload, path)
					if !ok || got == nil {
						t.Fatalf("required json path %q missing in payload=%v", path, payload)
					}
				}

				for _, path := range tc.ForbiddenJSONPaths {
					if _, ok := lookupJSONPath(payload, path); ok {
						t.Fatalf("forbidden json path %q present in payload=%v", path, payload)
					}
				}

				for path, want := range tc.ExpectedJSONValues {
					got, ok := lookupJSONPath(payload, path)
					if !ok {
						t.Fatalf("expected json path %q missing in payload=%v", path, payload)
					}
					if !reflect.DeepEqual(got, want) {
						t.Fatalf("path %q mismatch: expected %v (%T), got %v (%T)", path, want, want, got, got)
					}
				}
			})
		}
	}
}

func applyAPIContractSetup(t *testing.T, router http.Handler, setup string, vars map[string]string) {
	t.Helper()

	switch strings.TrimSpace(setup) {
	case "", "none":
		return
	case "alice_handle_confirmed":
		ensureHandleConfirmed(t, router, "alice", "alice@a.test")
	case "trusted_agents":
		orgA, orgB, tokenA, tokenB, orgTrustID, agentTrustID := setupTrustedAgents(t, router)
		vars["ORG_A"] = orgA
		vars["ORG_B"] = orgB
		vars["TOKEN_A"] = tokenA
		vars["TOKEN_B"] = tokenB
		vars["ORG_TRUST_ID"] = orgTrustID
		vars["AGENT_TRUST_ID"] = agentTrustID
	case "agents_no_trust":
		aliceHumanID := currentHumanID(t, router, "alice", "alice@a.test")
		bobHumanID := currentHumanID(t, router, "bob", "bob@b.test")
		orgA := createOrg(t, router, "alice", "alice@a.test", "No Trust Org A")
		orgB := createOrg(t, router, "bob", "bob@b.test", "No Trust Org B")
		tokenA := registerAgent(t, router, "alice", "alice@a.test", orgA, "agent-a", aliceHumanID)
		tokenB := registerAgent(t, router, "bob", "bob@b.test", orgB, "agent-b", bobHumanID)
		vars["ORG_A"] = orgA
		vars["ORG_B"] = orgB
		vars["TOKEN_A"] = tokenA
		vars["TOKEN_B"] = tokenB
	default:
		t.Fatalf("unsupported contract fixture setup %q", setup)
	}
}

func substituteFixtureVarsMap(in map[string]string, vars map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = substituteFixtureVars(value, vars)
	}
	return out
}

func parseFixtureBody(t *testing.T, raw json.RawMessage, vars map[string]string) any {
	t.Helper()
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var out any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode fixture body: %v", err)
	}
	return substituteFixtureAny(out, vars)
}

func substituteFixtureAny(in any, vars map[string]string) any {
	switch value := in.(type) {
	case string:
		return substituteFixtureVars(value, vars)
	case []any:
		out := make([]any, len(value))
		for i := range value {
			out[i] = substituteFixtureAny(value[i], vars)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(value))
		for key, entry := range value {
			out[key] = substituteFixtureAny(entry, vars)
		}
		return out
	default:
		return in
	}
}

func substituteFixtureVars(in string, vars map[string]string) string {
	out := in
	for key, value := range vars {
		out = strings.ReplaceAll(out, fmt.Sprintf("__%s__", key), value)
	}
	return out
}

func lookupJSONPath(root map[string]any, path string) (any, bool) {
	if strings.TrimSpace(path) == "" {
		return nil, false
	}
	var current any = root
	for _, segment := range strings.Split(path, ".") {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		next, ok := m[segment]
		if !ok {
			return nil, false
		}
		current = next
	}
	return current, true
}
