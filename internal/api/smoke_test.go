package api

import (
	"net/http"
	"reflect"
	"testing"
)

func TestLaunchSmoke(t *testing.T) {
	t.Run("Health endpoint responds and reports ok", func(t *testing.T) {
		router := newTestRouter()

		resp := doJSONRequest(t, router, http.MethodGet, "/health", nil, nil)
		if resp.Code != http.StatusOK {
			t.Fatalf("expected /health 200, got %d %s", resp.Code, resp.Body.String())
		}

		payload := decodeJSONMap(t, resp.Body.Bytes())
		if payload["status"] != "ok" {
			t.Fatalf("expected health status ok, got %v payload=%v", payload["status"], payload)
		}
	})

	t.Run("Profile lifecycle supports unique handles and metadata add change clear", func(t *testing.T) {
		router := newTestRouter()

		setAlice := doJSONRequest(t, router, http.MethodPatch, "/v1/me", map[string]any{
			"handle": "alice",
		}, humanHeaders("alice", "alice@a.test"))
		if setAlice.Code != http.StatusOK {
			t.Fatalf("expected alice handle set 200, got %d %s", setAlice.Code, setAlice.Body.String())
		}
		setAlicePayload := decodeJSONMap(t, setAlice.Body.Bytes())
		aliceHuman, ok := setAlicePayload["human"].(map[string]any)
		if !ok {
			t.Fatalf("missing human payload: %v", setAlicePayload)
		}
		if aliceHuman["handle"] != "alice" {
			t.Fatalf("expected alice handle, got %v", aliceHuman["handle"])
		}

		bobConflict := doJSONRequest(t, router, http.MethodPatch, "/v1/me", map[string]any{
			"handle": "alice",
		}, humanHeaders("bob", "bob@b.test"))
		if bobConflict.Code != http.StatusConflict {
			t.Fatalf("expected bob duplicate handle 409, got %d %s", bobConflict.Code, bobConflict.Body.String())
		}
		requireErrorCode(t, decodeJSONMap(t, bobConflict.Body.Bytes()), "human_handle_exists")

		addMetadata := doJSONRequest(t, router, http.MethodPatch, "/v1/me/metadata", map[string]any{
			"metadata": map[string]any{
				"public": true,
				"bio":    "Alice launch smoke profile",
			},
		}, humanHeaders("alice", "alice@a.test"))
		if addMetadata.Code != http.StatusOK {
			t.Fatalf("expected profile metadata add 200, got %d %s", addMetadata.Code, addMetadata.Body.String())
		}
		requireEntityMetadata(t, decodeJSONMap(t, addMetadata.Body.Bytes()), "human", map[string]any{
			"public": true,
			"bio":    "Alice launch smoke profile",
		})

		changeMetadata := doJSONRequest(t, router, http.MethodPatch, "/v1/me/metadata", map[string]any{
			"metadata": map[string]any{
				"public": true,
				"bio":    "Alice launch smoke profile updated",
				"stage":  "launch",
			},
		}, humanHeaders("alice", "alice@a.test"))
		if changeMetadata.Code != http.StatusOK {
			t.Fatalf("expected profile metadata change 200, got %d %s", changeMetadata.Code, changeMetadata.Body.String())
		}
		requireEntityMetadata(t, decodeJSONMap(t, changeMetadata.Body.Bytes()), "human", map[string]any{
			"public": true,
			"bio":    "Alice launch smoke profile updated",
			"stage":  "launch",
		})

		clearMetadata := doJSONRequest(t, router, http.MethodPatch, "/v1/me/metadata", map[string]any{
			"metadata": map[string]any{},
		}, humanHeaders("alice", "alice@a.test"))
		if clearMetadata.Code != http.StatusOK {
			t.Fatalf("expected profile metadata clear 200, got %d %s", clearMetadata.Code, clearMetadata.Body.String())
		}
		requireEntityMetadata(t, decodeJSONMap(t, clearMetadata.Body.Bytes()), "human", map[string]any{})
	})

	t.Run("Organization lifecycle supports unique handles metadata add change clear and delete", func(t *testing.T) {
		router := newTestRouter()

		orgID := createOrg(t, router, "alice", "alice@a.test", "Launch Alpha")
		ensureHandleConfirmed(t, router, "bob", "bob@b.test")

		bobConflict := doJSONRequest(t, router, http.MethodPost, "/v1/orgs", map[string]any{
			"handle":       "launch-alpha",
			"display_name": "Launch Alpha Duplicate",
		}, humanHeaders("bob", "bob@b.test"))
		if bobConflict.Code != http.StatusConflict {
			t.Fatalf("expected bob duplicate org handle 409, got %d %s", bobConflict.Code, bobConflict.Body.String())
		}
		requireErrorCode(t, decodeJSONMap(t, bobConflict.Body.Bytes()), "org_handle_exists")

		addMetadata := doJSONRequest(t, router, http.MethodPatch, "/v1/orgs/"+orgID+"/metadata", map[string]any{
			"metadata": map[string]any{
				"public":      true,
				"description": "Launch Alpha Organization",
			},
		}, humanHeaders("alice", "alice@a.test"))
		if addMetadata.Code != http.StatusOK {
			t.Fatalf("expected org metadata add 200, got %d %s", addMetadata.Code, addMetadata.Body.String())
		}
		requireEntityMetadata(t, decodeJSONMap(t, addMetadata.Body.Bytes()), "organization", map[string]any{
			"public":      true,
			"description": "Launch Alpha Organization",
		})

		changeMetadata := doJSONRequest(t, router, http.MethodPatch, "/v1/orgs/"+orgID+"/metadata", map[string]any{
			"metadata": map[string]any{
				"public":      true,
				"description": "Launch Alpha Organization Updated",
				"stage":       "launch",
			},
		}, humanHeaders("alice", "alice@a.test"))
		if changeMetadata.Code != http.StatusOK {
			t.Fatalf("expected org metadata change 200, got %d %s", changeMetadata.Code, changeMetadata.Body.String())
		}
		requireEntityMetadata(t, decodeJSONMap(t, changeMetadata.Body.Bytes()), "organization", map[string]any{
			"public":      true,
			"description": "Launch Alpha Organization Updated",
			"stage":       "launch",
		})

		clearMetadata := doJSONRequest(t, router, http.MethodPatch, "/v1/orgs/"+orgID+"/metadata", map[string]any{
			"metadata": map[string]any{},
		}, humanHeaders("alice", "alice@a.test"))
		if clearMetadata.Code != http.StatusOK {
			t.Fatalf("expected org metadata clear 200, got %d %s", clearMetadata.Code, clearMetadata.Body.String())
		}
		requireEntityMetadata(t, decodeJSONMap(t, clearMetadata.Body.Bytes()), "organization", map[string]any{})

		deleteResp := doJSONRequest(t, router, http.MethodDelete, "/v1/orgs/"+orgID, nil, humanHeaders("alice", "alice@a.test"))
		if deleteResp.Code != http.StatusOK {
			t.Fatalf("expected org delete 200, got %d %s", deleteResp.Code, deleteResp.Body.String())
		}
		deletePayload := decodeJSONMap(t, deleteResp.Body.Bytes())
		if deletePayload["result"] != "deleted" {
			t.Fatalf("expected org delete result deleted, got %v payload=%v", deletePayload["result"], deletePayload)
		}

		myOrgs := doJSONRequest(t, router, http.MethodGet, "/v1/me/orgs", nil, humanHeaders("alice", "alice@a.test"))
		if myOrgs.Code != http.StatusOK {
			t.Fatalf("expected /v1/me/orgs 200 after delete, got %d %s", myOrgs.Code, myOrgs.Body.String())
		}
		if membershipHasOrg(t, decodeJSONMap(t, myOrgs.Body.Bytes()), orgID) {
			t.Fatalf("deleted org %q still present in /v1/me/orgs", orgID)
		}
	})

	t.Run("Two agent lifecycle supports bind handle finalize metadata list and revoke", func(t *testing.T) {
		router := newTestRouter()

		orgID := createOrg(t, router, "alice", "alice@a.test", "Launch Agents")
		aliceHumanID := currentHumanID(t, router, "alice", "alice@a.test")

		tokenA, agentUUIDA := registerAgentWithUUID(t, router, "alice", "alice@a.test", orgID, "launch-agent-a", aliceHumanID)

		addMetadata := doJSONRequest(t, router, http.MethodPatch, "/v1/agents/me/metadata", map[string]any{
			"metadata": map[string]any{
				"public": true,
				"role":   "primary",
			},
		}, map[string]string{
			"Authorization": "Bearer " + tokenA,
		})
		if addMetadata.Code != http.StatusOK {
			t.Fatalf("expected agent metadata add 200, got %d %s", addMetadata.Code, addMetadata.Body.String())
		}
		requireEntityMetadata(t, decodeJSONMap(t, addMetadata.Body.Bytes()), "agent", map[string]any{
			"public": true,
			"role":   "primary",
		})

		changeMetadata := doJSONRequest(t, router, http.MethodPatch, "/v1/agents/me/metadata", map[string]any{
			"metadata": map[string]any{
				"public": true,
				"role":   "primary-updated",
				"stage":  "launch",
			},
		}, map[string]string{
			"Authorization": "Bearer " + tokenA,
		})
		if changeMetadata.Code != http.StatusOK {
			t.Fatalf("expected agent metadata change 200, got %d %s", changeMetadata.Code, changeMetadata.Body.String())
		}
		requireEntityMetadata(t, decodeJSONMap(t, changeMetadata.Body.Bytes()), "agent", map[string]any{
			"public": true,
			"role":   "primary-updated",
			"stage":  "launch",
		})

		clearMetadata := doJSONRequest(t, router, http.MethodPatch, "/v1/agents/me/metadata", map[string]any{
			"metadata": map[string]any{},
		}, map[string]string{
			"Authorization": "Bearer " + tokenA,
		})
		if clearMetadata.Code != http.StatusOK {
			t.Fatalf("expected agent metadata clear 200, got %d %s", clearMetadata.Code, clearMetadata.Body.String())
		}
		requireEntityMetadata(t, decodeJSONMap(t, clearMetadata.Body.Bytes()), "agent", map[string]any{})

		createBindToken := doJSONRequest(t, router, http.MethodPost, "/v1/me/agents/bind-tokens", map[string]any{
			"org_id": orgID,
		}, humanHeaders("alice", "alice@a.test"))
		if createBindToken.Code != http.StatusCreated {
			t.Fatalf("expected second bind token 201, got %d %s", createBindToken.Code, createBindToken.Body.String())
		}
		createBindTokenPayload := decodeJSONMap(t, createBindToken.Body.Bytes())
		bindTokenB, _ := createBindTokenPayload["bind_token"].(string)
		if bindTokenB == "" {
			t.Fatalf("expected second bind_token in response")
		}

		redeemB := doJSONRequest(t, router, http.MethodPost, "/v1/agents/bind", map[string]any{
			"bind_token": bindTokenB,
			"agent_id":   "temporary-second-agent-name",
		}, nil)
		if redeemB.Code != http.StatusCreated {
			t.Fatalf("expected second bind redeem 201, got %d %s", redeemB.Code, redeemB.Body.String())
		}
		redeemBPayload := decodeJSONMap(t, redeemB.Body.Bytes())
		tokenB, _ := redeemBPayload["token"].(string)
		if tokenB == "" {
			t.Fatalf("expected second bind redeem token")
		}

		duplicateHandle := doJSONRequest(t, router, http.MethodPatch, "/v1/agents/me", map[string]any{
			"handle": "launch-agent-a",
		}, map[string]string{
			"Authorization": "Bearer " + tokenB,
		})
		if duplicateHandle.Code != http.StatusConflict {
			t.Fatalf("expected duplicate agent handle 409, got %d %s", duplicateHandle.Code, duplicateHandle.Body.String())
		}
		requireErrorCode(t, decodeJSONMap(t, duplicateHandle.Body.Bytes()), "agent_exists")

		finalizeB := doJSONRequest(t, router, http.MethodPatch, "/v1/agents/me", map[string]any{
			"handle": "launch-agent-b",
		}, map[string]string{
			"Authorization": "Bearer " + tokenB,
		})
		if finalizeB.Code != http.StatusOK {
			t.Fatalf("expected second agent finalize 200, got %d %s", finalizeB.Code, finalizeB.Body.String())
		}
		finalizeBPayload := decodeJSONMap(t, finalizeB.Body.Bytes())
		agentB := requireEntity(t, finalizeBPayload, "agent")
		agentUUIDB, _ := agentB["agent_uuid"].(string)
		if agentB["handle"] != "launch-agent-b" || agentUUIDB == "" {
			t.Fatalf("unexpected second agent payload: %v", agentB)
		}

		myAgents := doJSONRequest(t, router, http.MethodGet, "/v1/me/agents", nil, humanHeaders("alice", "alice@a.test"))
		if myAgents.Code != http.StatusOK {
			t.Fatalf("expected /v1/me/agents 200, got %d %s", myAgents.Code, myAgents.Body.String())
		}
		agents := requireAgentList(t, decodeJSONMap(t, myAgents.Body.Bytes()))
		requireAgentStatus(t, agents, agentUUIDA, "active")
		requireAgentStatus(t, agents, agentUUIDB, "active")

		revokeA := doJSONRequest(t, router, http.MethodDelete, "/v1/agents/"+agentUUIDA, nil, humanHeaders("alice", "alice@a.test"))
		if revokeA.Code != http.StatusOK {
			t.Fatalf("expected first agent revoke 200, got %d %s", revokeA.Code, revokeA.Body.String())
		}

		revokedAAuth := doJSONRequest(t, router, http.MethodGet, "/v1/agents/me", nil, map[string]string{
			"Authorization": "Bearer " + tokenA,
		})
		if revokedAAuth.Code != http.StatusUnauthorized {
			t.Fatalf("expected revoked first agent token to fail with 401, got %d %s", revokedAAuth.Code, revokedAAuth.Body.String())
		}

		revokeB := doJSONRequest(t, router, http.MethodDelete, "/v1/agents/"+agentUUIDB, nil, humanHeaders("alice", "alice@a.test"))
		if revokeB.Code != http.StatusOK {
			t.Fatalf("expected second agent revoke 200, got %d %s", revokeB.Code, revokeB.Body.String())
		}

		revokedBAuth := doJSONRequest(t, router, http.MethodGet, "/v1/agents/me", nil, map[string]string{
			"Authorization": "Bearer " + tokenB,
		})
		if revokedBAuth.Code != http.StatusUnauthorized {
			t.Fatalf("expected revoked second agent token to fail with 401, got %d %s", revokedBAuth.Code, revokedBAuth.Body.String())
		}

		myAgentsAfterRevoke := doJSONRequest(t, router, http.MethodGet, "/v1/me/agents", nil, humanHeaders("alice", "alice@a.test"))
		if myAgentsAfterRevoke.Code != http.StatusOK {
			t.Fatalf("expected /v1/me/agents 200 after revoke, got %d %s", myAgentsAfterRevoke.Code, myAgentsAfterRevoke.Body.String())
		}
		agentsAfterRevoke := requireAgentList(t, decodeJSONMap(t, myAgentsAfterRevoke.Body.Bytes()))
		requireAgentStatus(t, agentsAfterRevoke, agentUUIDA, "revoked")
		requireAgentStatus(t, agentsAfterRevoke, agentUUIDB, "revoked")
	})
}

func requireErrorCode(t *testing.T, payload map[string]any, want string) {
	t.Helper()
	if payload["error"] != want {
		t.Fatalf("expected error %q, got %v payload=%v", want, payload["error"], payload)
	}
}

func requireEntityMetadata(t *testing.T, payload map[string]any, entityKey string, want map[string]any) {
	t.Helper()
	entity := requireEntity(t, payload, entityKey)
	if len(want) == 0 {
		got, exists := entity["metadata"]
		if !exists || got == nil {
			return
		}
		gotMap, ok := got.(map[string]any)
		if ok && len(gotMap) == 0 {
			return
		}
		t.Fatalf("expected %s.metadata to be empty or omitted, got %v payload=%v", entityKey, got, payload)
	}
	got, ok := entity["metadata"].(map[string]any)
	if !ok {
		t.Fatalf("expected %s.metadata object, got %T payload=%v", entityKey, entity["metadata"], payload)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %s.metadata=%v, got %v", entityKey, want, got)
	}
}

func requireEntity(t *testing.T, payload map[string]any, entityKey string) map[string]any {
	t.Helper()
	entity, ok := payload[entityKey].(map[string]any)
	if !ok {
		t.Fatalf("expected %s object, got %T payload=%v", entityKey, payload[entityKey], payload)
	}
	return entity
}

func membershipHasOrg(t *testing.T, payload map[string]any, orgID string) bool {
	t.Helper()
	if _, exists := payload["memberships"]; !exists {
		return false
	}
	memberships, ok := payload["memberships"].([]any)
	if !ok {
		t.Fatalf("expected memberships array, got %T payload=%v", payload["memberships"], payload)
	}
	for _, raw := range memberships {
		row, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		org, ok := row["org"].(map[string]any)
		if !ok {
			continue
		}
		if org["org_id"] == orgID {
			return true
		}
	}
	return false
}

func requireAgentList(t *testing.T, payload map[string]any) []map[string]any {
	t.Helper()
	rawAgents, ok := payload["agents"].([]any)
	if !ok {
		t.Fatalf("expected agents array, got %T payload=%v", payload["agents"], payload)
	}
	out := make([]map[string]any, 0, len(rawAgents))
	for _, raw := range rawAgents {
		agent, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("expected agent row object, got %T payload=%v", raw, payload)
		}
		out = append(out, agent)
	}
	return out
}

func requireAgentStatus(t *testing.T, agents []map[string]any, agentUUID, wantStatus string) {
	t.Helper()
	for _, agent := range agents {
		if agent["agent_uuid"] != agentUUID {
			continue
		}
		if agent["status"] != wantStatus {
			t.Fatalf("expected agent %q status %q, got %v payload=%v", agentUUID, wantStatus, agent["status"], agent)
		}
		return
	}
	t.Fatalf("agent %q not found in list %v", agentUUID, agents)
}
