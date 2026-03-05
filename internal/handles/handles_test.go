package handles

import "testing"

func TestNormalize(t *testing.T) {
	if got := Normalize("  A!! B__C..D  "); got != "a-b_c.d" {
		t.Fatalf("unexpected normalize result: %q", got)
	}
}

func TestValidateHandle(t *testing.T) {
	for _, tc := range []struct {
		handle string
		ok     bool
	}{
		{handle: "ab", ok: true},
		{handle: "a", ok: false},
		{handle: "-ab", ok: false},
		{handle: "ab_1", ok: true},
		{handle: "fuck", ok: false},
		{handle: "f.u.c.k", ok: false},
	} {
		err := ValidateHandle(tc.handle)
		if tc.ok && err != nil {
			t.Fatalf("expected handle %q valid, got err=%v", tc.handle, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("expected handle %q invalid", tc.handle)
		}
	}
}

func TestValidateAgentRef(t *testing.T) {
	for _, tc := range []struct {
		ref string
		ok  bool
	}{
		{ref: "alpha-1", ok: true},
		{ref: "o/agent", ok: false},
		{ref: "org/agent", ok: true},
		{ref: "org/human/agent", ok: true},
		{ref: "org/human/agent/extra", ok: false},
		{ref: "org/fuck", ok: false},
	} {
		err := ValidateAgentRef(tc.ref)
		if tc.ok && err != nil {
			t.Fatalf("expected ref %q valid, got err=%v", tc.ref, err)
		}
		if !tc.ok && err == nil {
			t.Fatalf("expected ref %q invalid", tc.ref)
		}
	}
}

func TestBuildAgentURI(t *testing.T) {
	if got := BuildAgentURI("Org", nil, "Agent"); got != "org/agent" {
		t.Fatalf("unexpected org-owned URI: %q", got)
	}
	h := "Human"
	if got := BuildAgentURI("Org", &h, "Agent"); got != "org/human/agent" {
		t.Fatalf("unexpected human-owned URI: %q", got)
	}
}
