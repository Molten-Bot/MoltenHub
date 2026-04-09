package main

import (
	"net/http/httptest"
	"testing"
	"time"

	"moltenhub/internal/api"
	"moltenhub/internal/auth"
	"moltenhub/internal/longpoll"
	"moltenhub/internal/store"
)

func newSmokeTestServer() *httptest.Server {
	mem := store.NewMemoryStore()
	waiters := longpoll.NewWaiters()
	handler := api.NewHandler(
		mem,
		mem,
		waiters,
		auth.NewDevHumanAuthProvider(),
		"https://hub.example.com",
		"",
		"",
		"",
		"",
		"example.com",
		true,
		15*time.Minute,
		false,
	)
	return httptest.NewServer(api.NewRouter(handler))
}

func TestRunnerLaunchSmokeFlow(t *testing.T) {
	server := newSmokeTestServer()
	defer server.Close()

	r := &runner{
		baseURL: server.URL,
		client:  server.Client(),
	}
	r.client.Timeout = 15 * time.Second

	steps := []struct {
		name string
		run  func(*runner) error
	}{
		{name: "Health endpoint responds and reports ok", run: (*runner).stepHealth},
		{name: "Alice creates handle", run: (*runner).stepAliceCreatesHandle},
		{name: "Bob tries to add the same handle and gets an error", run: (*runner).stepBobCannotTakeAliceHandle},
		{name: "Alice adds metadata to her profile", run: (*runner).stepAliceAddsProfileMetadata},
		{name: "Alice changes metadata from her profile", run: (*runner).stepAliceChangesProfileMetadata},
		{name: "Alice clears metadata from her profile", run: (*runner).stepAliceClearsProfileMetadata},
		{name: "Alice creates an organization", run: (*runner).stepAliceCreatesOrganization},
		{name: "Bob tries to add an organization with the same handle and gets an error", run: (*runner).stepBobCannotTakeOrgHandle},
		{name: "Alice adds metadata to an organization", run: (*runner).stepAliceAddsOrgMetadata},
		{name: "Alice changes metadata to an organization", run: (*runner).stepAliceChangesOrgMetadata},
		{name: "Alice clears metadata from an organization", run: (*runner).stepAliceClearsOrgMetadata},
		{name: "Alice creates an organization and deletes it", run: (*runner).stepAliceDeletesOrganization},
		{name: "Alice creates a bind token and an agent binds successfully", run: (*runner).stepAgentBinds},
		{name: "Alice creates a bind token and the agent updates its profile handle", run: (*runner).stepAgentFinalizesHandle},
		{name: "Alice creates a bind token and the agent tries to add an existing handle and gets an error", run: (*runner).stepAgentDuplicateHandleRejected},
		{name: "Alice creates a bind token and the agent adds profile metadata", run: (*runner).stepAgentAddsMetadata},
		{name: "Alice creates a bind token and the agent changes profile metadata", run: (*runner).stepAgentChangesMetadata},
		{name: "Alice creates a bind token and the agent clears profile metadata", run: (*runner).stepAgentClearsMetadata},
		{name: "Alice invites two agents by bind token, binds both agents, and sees both in her list", run: (*runner).stepAliceSeesBothAgents},
		{name: "Alice creates trust between both bound agents", run: (*runner).stepAliceCreatesAgentTrust},
		{name: "OpenClaw plugin registration succeeds for both agents", run: (*runner).stepOpenClawRegisterPlugin},
		{name: "OpenClaw HTTP publish/pull/ack succeeds between bound agents", run: (*runner).stepOpenClawHTTPDelivery},
		{name: "OpenClaw websocket delivery and ack succeeds", run: (*runner).stepOpenClawWebSocketDelivery},
		{name: "Alice binds an agent and revokes it", run: (*runner).stepAliceRevokesFirstAgent},
		{name: "Alice binds two agents and revokes both agents", run: (*runner).stepAliceRevokesBothAgents},
	}

	for _, tc := range steps {
		if err := tc.run(r); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
	}
}
