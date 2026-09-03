package main

import (
	"testing"
	"time"
)

func TestLastFinalizedDateBeforeFinalizeHour(t *testing.T) {
	now := time.Date(2026, 9, 3, 1, 59, 0, 0, jst)
	got := lastFinalizedDate(now)
	want := time.Date(2026, 9, 1, 0, 0, 0, 0, jst)
	if !got.Equal(want) {
		t.Fatalf("got %s want %s", formatDate(got), formatDate(want))
	}
}

func TestLastFinalizedDateAfterFinalizeHour(t *testing.T) {
	now := time.Date(2026, 9, 3, 2, 0, 0, 0, jst)
	got := lastFinalizedDate(now)
	want := time.Date(2026, 9, 2, 0, 0, 0, 0, jst)
	if !got.Equal(want) {
		t.Fatalf("got %s want %s", formatDate(got), formatDate(want))
	}
}

func TestChunkDateRangesSingleChunk(t *testing.T) {
	from, _ := parseDate("2026-07-01")
	to, _ := parseDate("2026-07-10")
	chunks := chunkDateRanges(dateRange{from: from, to: to})
	if len(chunks) != 1 {
		t.Fatalf("chunks=%d want 1", len(chunks))
	}
}

func TestChunkDateRangesMultipleChunks(t *testing.T) {
	from, _ := parseDate("2026-01-01")
	to, _ := parseDate("2026-06-30")
	chunks := chunkDateRanges(dateRange{from: from, to: to})
	if len(chunks) != 2 {
		t.Fatalf("chunks=%d want 2", len(chunks))
	}
	if formatDate(chunks[0].from) != "2026-01-01" || formatDate(chunks[0].to) != "2026-04-02" {
		t.Fatalf("first chunk=%s..%s", formatDate(chunks[0].from), formatDate(chunks[0].to))
	}
	if formatDate(chunks[1].from) != "2026-04-03" || formatDate(chunks[1].to) != "2026-06-30" {
		t.Fatalf("second chunk=%s..%s", formatDate(chunks[1].from), formatDate(chunks[1].to))
	}
}

func TestFullRangeSpan(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, jst)
	r := fullDateRange(now)
	days := int(r.to.Sub(r.from).Hours()/24) + 1
	if days != maxReportDays {
		t.Fatalf("days=%d want %d", days, maxReportDays)
	}
}
