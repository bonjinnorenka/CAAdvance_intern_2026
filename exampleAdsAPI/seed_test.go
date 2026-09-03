package main

import (
	"testing"
	"time"
)

func TestGenerateMetricsDeterministic(t *testing.T) {
	a := generateMetrics("acc_00101", "2026-07-01")
	b := generateMetrics("acc_00101", "2026-07-01")
	if a != b {
		t.Fatalf("same account and date must match: %+v vs %+v", a, b)
	}
}

func TestGenerateMetricsChangesWithDate(t *testing.T) {
	a := generateMetrics("acc_00101", "2026-07-01")
	b := generateMetrics("acc_00101", "2026-07-02")
	if a == b {
		t.Fatalf("different dates should change metrics: %+v", a)
	}
}

func TestGenerateMetricsChangesWithAccount(t *testing.T) {
	a := generateMetrics("acc_00101", "2026-07-01")
	b := generateMetrics("acc_00102", "2026-07-01")
	if a == b {
		t.Fatalf("different accounts should change metrics: %+v", a)
	}
}

func TestGenerateMetricsConstraints(t *testing.T) {
	for _, accountID := range []string{"acc_00101", "acc_00105", "acc_00110"} {
		for _, date := range []string{"2026-01-01", "2026-07-15", "2026-12-31"} {
			m := generateMetrics(accountID, date)
			if m.Impressions < 0 || m.Clicks < 0 || m.Cost < 0 || m.Conversions < 0 {
				t.Fatalf("negative metric: %+v", m)
			}
			if m.Clicks > m.Impressions {
				t.Fatalf("clicks > impressions: %+v", m)
			}
			if m.Conversions > m.Clicks {
				t.Fatalf("conversions > clicks: %+v", m)
			}
		}
	}
}

func TestLastFinalizedDate(t *testing.T) {
	before := time.Date(2026, 9, 3, 1, 59, 0, 0, jst)
	got := lastFinalizedDate(before)
	want := time.Date(2026, 9, 1, 0, 0, 0, 0, jst)
	if !got.Equal(want) {
		t.Fatalf("before 02:00: got %s want %s", got.Format(dateLayout), want.Format(dateLayout))
	}

	at := time.Date(2026, 9, 3, 2, 0, 0, 0, jst)
	got = lastFinalizedDate(at)
	want = time.Date(2026, 9, 2, 0, 0, 0, 0, jst)
	if !got.Equal(want) {
		t.Fatalf("at 02:00: got %s want %s", got.Format(dateLayout), want.Format(dateLayout))
	}
}

func TestRateLimiter(t *testing.T) {
	l := newRateLimiter(60, time.Minute)
	start := time.Date(2026, 9, 3, 12, 0, 0, 0, jst)
	for i := 0; i < 60; i++ {
		if !l.allow(start.Add(time.Duration(i) * time.Millisecond)) {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	if l.allow(start.Add(59 * time.Second)) {
		t.Fatal("61st request in the window should be rejected")
	}
	if !l.allow(start.Add(time.Minute)) {
		t.Fatal("request after the window should be allowed")
	}
}
