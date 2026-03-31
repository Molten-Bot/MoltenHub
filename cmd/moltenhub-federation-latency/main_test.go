package main

import "testing"

func TestComputeLatencyStats(t *testing.T) {
	stats := computeLatencyStats([]int64{500, 100, 300, 400, 200})
	if stats.Count != 5 {
		t.Fatalf("expected count 5, got %d", stats.Count)
	}
	if stats.Min != 100 || stats.Max != 500 {
		t.Fatalf("expected min/max 100/500, got %d/%d", stats.Min, stats.Max)
	}
	if stats.P50 != 300 {
		t.Fatalf("expected p50 300, got %d", stats.P50)
	}
	if stats.P95 != 500 {
		t.Fatalf("expected p95 500, got %d", stats.P95)
	}
	if stats.P99 != 500 {
		t.Fatalf("expected p99 500, got %d", stats.P99)
	}
	if stats.Avg != 300 {
		t.Fatalf("expected avg 300, got %f", stats.Avg)
	}
}

func TestPercentileNearestRankBounds(t *testing.T) {
	values := []int64{10, 20, 30}
	if got := percentileNearestRank(values, -1); got != 10 {
		t.Fatalf("expected lower bound percentile 10, got %d", got)
	}
	if got := percentileNearestRank(values, 2); got != 30 {
		t.Fatalf("expected upper bound percentile 30, got %d", got)
	}
}

func TestDirectionReportMeetsSLO(t *testing.T) {
	report := directionReport{
		Label: "na_to_eu",
		EndToEnd: latencyStats{
			Count: 10,
			P95:   9800,
		},
	}
	if !report.meetsSLO(10000) {
		t.Fatalf("expected report to meet 10s SLO")
	}
	if report.meetsSLO(9000) {
		t.Fatalf("expected report not to meet 9s SLO")
	}
}
