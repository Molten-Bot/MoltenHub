package main

import "testing"

func TestComputeStats(t *testing.T) {
	stats := computeStats([]float64{500, 100, 300, 400, 200})
	if stats.Count != 5 {
		t.Fatalf("expected count 5, got %d", stats.Count)
	}
	if stats.Min != 100 || stats.Max != 500 {
		t.Fatalf("expected min/max 100/500, got %.0f/%.0f", stats.Min, stats.Max)
	}
	if stats.P50 != 300 {
		t.Fatalf("expected p50 300, got %.0f", stats.P50)
	}
	if stats.P95 != 500 {
		t.Fatalf("expected p95 500, got %.0f", stats.P95)
	}
	if stats.P99 != 500 {
		t.Fatalf("expected p99 500, got %.0f", stats.P99)
	}
	if stats.Avg != 300 {
		t.Fatalf("expected avg 300, got %f", stats.Avg)
	}
}

func TestPercentileBounds(t *testing.T) {
	values := []float64{10, 20, 30}
	if got := percentile(values, -5); got != 10 {
		t.Fatalf("expected lower bound percentile 10, got %.0f", got)
	}
	if got := percentile(values, 105); got != 30 {
		t.Fatalf("expected upper bound percentile 30, got %.0f", got)
	}
}

func TestExtractDelivery(t *testing.T) {
	event := map[string]any{
		"type": "delivery",
		"result": map[string]any{
			"message": map[string]any{
				"message_id": "m-1",
			},
			"delivery": map[string]any{
				"delivery_id": "d-1",
			},
		},
	}
	messageID, deliveryID, ok := extractDelivery(event)
	if !ok {
		t.Fatalf("expected extractDelivery ok")
	}
	if messageID != "m-1" || deliveryID != "d-1" {
		t.Fatalf("unexpected extractDelivery values message_id=%q delivery_id=%q", messageID, deliveryID)
	}
}
