package store

import (
	"fmt"
	"os"
	"strings"
)

const (
	defaultStateBackend = "memory"
	defaultQueueBackend = "memory"
)

// NewStoresFromEnv wires backend implementations from env configuration.
func NewStoresFromEnv() (ControlPlaneStore, MessageQueueStore, error) {
	stateBackend := strings.ToLower(strings.TrimSpace(os.Getenv("STATOCYST_STATE_BACKEND")))
	queueBackend := strings.ToLower(strings.TrimSpace(os.Getenv("STATOCYST_QUEUE_BACKEND")))
	if stateBackend == "" {
		stateBackend = defaultStateBackend
	}
	if queueBackend == "" {
		queueBackend = defaultQueueBackend
	}

	switch {
	case stateBackend == "memory" && queueBackend == "memory":
		mem := NewMemoryStore()
		return mem, mem, nil
	case stateBackend == "memory" && queueBackend == "s3":
		mem := NewMemoryStore()
		queue, err := NewS3QueueStoreFromEnv()
		if err != nil {
			return nil, nil, err
		}
		return mem, queue, nil
	case stateBackend == "s3" && queueBackend == "memory":
		state, err := NewS3StateStoreFromEnv()
		if err != nil {
			return nil, nil, err
		}
		return state, state, nil
	case stateBackend == "s3" && queueBackend == "s3":
		state, err := NewS3StateStoreFromEnv()
		if err != nil {
			return nil, nil, err
		}
		queue, err := NewS3QueueStoreFromEnv()
		if err != nil {
			return nil, nil, err
		}
		return state, queue, nil
	case stateBackend != "memory" && stateBackend != "s3":
		return nil, nil, fmt.Errorf("unsupported state backend %q", stateBackend)
	default:
		return nil, nil, fmt.Errorf("unsupported queue backend %q", queueBackend)
	}
}
