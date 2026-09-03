package main

import (
	"fmt"
	"time"
)

const (
	dateLayout   = "2006-01-02"
	finalizeHour = 2
	maxReportDays = 92
)

var jst = time.FixedZone("JST", 9*3600)

func lastFinalizedDate(now time.Time) time.Time {
	now = now.In(jst)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, jst)
	if now.Hour() < finalizeHour {
		return today.AddDate(0, 0, -2)
	}
	return today.AddDate(0, 0, -1)
}

func parseDate(raw string) (time.Time, error) {
	t, err := time.ParseInLocation(dateLayout, raw, jst)
	if err != nil || t.Format(dateLayout) != raw {
		return time.Time{}, fmt.Errorf("invalid date %q, expected YYYY-MM-DD", raw)
	}
	return t, nil
}

func formatDate(t time.Time) string {
	return t.In(jst).Format(dateLayout)
}

type dateRange struct {
	from time.Time
	to   time.Time
}

func defaultDateRange(now time.Time) dateRange {
	d := lastFinalizedDate(now)
	return dateRange{from: d, to: d}
}

func singleDateRange(date string) (dateRange, error) {
	d, err := parseDate(date)
	if err != nil {
		return dateRange{}, err
	}
	return dateRange{from: d, to: d}, nil
}

func fullDateRange(now time.Time) dateRange {
	to := lastFinalizedDate(now)
	from := to.AddDate(0, 0, -(maxReportDays - 1))
	return dateRange{from: from, to: to}
}

func chunkDateRanges(r dateRange) []dateRange {
	chunks := make([]dateRange, 0)
	for from := r.from; !from.After(r.to); {
		to := from.AddDate(0, 0, maxReportDays-1)
		if to.After(r.to) {
			to = r.to
		}
		chunks = append(chunks, dateRange{from: from, to: to})
		from = to.AddDate(0, 0, 1)
	}
	return chunks
}
