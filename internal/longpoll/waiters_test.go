package longpoll

import (
	"testing"
	"time"
)

func TestWaitersRegisterNotifyAndCancel(t *testing.T) {
	w := NewWaiters()
	chA, cancelA := w.Register("agent-a")
	chB, cancelB := w.Register("agent-a")

	w.Notify("agent-a")

	select {
	case <-chA:
	default:
		t.Fatal("expected signal for first waiter")
	}
	select {
	case <-chB:
	default:
		t.Fatal("expected signal for second waiter")
	}

	cancelA()
	if got := len(w.byAgent["agent-a"]); got != 1 {
		t.Fatalf("expected one waiter left after cancel, got %d", got)
	}

	cancelB()
	if _, ok := w.byAgent["agent-a"]; ok {
		t.Fatal("expected all waiters removed after final cancel")
	}

	// Idempotent cancel should not panic.
	cancelB()
}

func TestWaitersNotifyDoesNotBlockWhenChannelIsFull(t *testing.T) {
	w := NewWaiters()
	ch, _ := w.Register("agent-a")

	w.Notify("agent-a")

	done := make(chan struct{})
	go func() {
		w.Notify("agent-a")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("notify blocked while waiter channel was full")
	}

	select {
	case <-ch:
	default:
		t.Fatal("expected buffered waiter signal")
	}
}

func TestWaitersNotifyUnknownAgentIsNoop(t *testing.T) {
	w := NewWaiters()
	w.Notify("missing-agent")
}
