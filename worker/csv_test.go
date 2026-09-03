package main

import (
	"bytes"
	"encoding/csv"
	"strings"
	"testing"
	"time"
)

func TestCustomerChargeJPY(t *testing.T) {
	got, err := customerChargeJPY(100000, 20)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 125000 {
		t.Fatalf("got %d, want 125000", got)
	}
}

func TestCustomerChargeJPYZeroMargin(t *testing.T) {
	got, err := customerChargeJPY(100000, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != 100000 {
		t.Fatalf("got %d, want 100000", got)
	}
}

func TestCustomerChargeJPYInvalidRate(t *testing.T) {
	if _, err := customerChargeJPY(100000, 100); err == nil {
		t.Fatal("expected error for margin_rate 100")
	}
	if _, err := customerChargeJPY(100000, -1); err == nil {
		t.Fatal("expected error for negative margin_rate")
	}
}

func parseCSVBody(t *testing.T, body []byte) [][]string {
	t.Helper()
	if !bytes.HasPrefix(body, utf8BOM) {
		t.Fatal("csv is missing UTF-8 BOM")
	}
	r := csv.NewReader(bytes.NewReader(body[len(utf8BOM):]))
	records, err := r.ReadAll()
	if err != nil {
		t.Fatalf("parse csv: %v", err)
	}
	return records
}

func TestBuildReportCSVEmptyIsHeaderOnly(t *testing.T) {
	body, err := buildReportCSV(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	records := parseCSVBody(t, body)
	if len(records) != 1 {
		t.Fatalf("got %d records, want header only", len(records))
	}
	if strings.Join(records[0], ",") != strings.Join(csvHeader, ",") {
		t.Fatalf("header = %#v, want %#v", records[0], csvHeader)
	}
}

func TestBuildReportCSVColumns(t *testing.T) {
	date := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	body, err := buildReportCSV([]reportCSVRow{{
		AccountID:      "acc_00101",
		AccountName:    "アカウントA",
		Date:           date,
		Impressions:    10,
		Clicks:         2,
		Conversions:    1,
		Cost:           100000,
		MarginRate:     20,
		CustomerCharge: 125000,
	}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	records := parseCSVBody(t, body)
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}
	wantHeader := []string{"アカウントID", "アカウント名", "日付", "表示回数", "クリック数", "コンバージョン数", "媒体費用", "マージン料率(%)", "顧客請求額"}
	if strings.Join(records[0], ",") != strings.Join(wantHeader, ",") {
		t.Fatalf("header = %#v, want %#v", records[0], wantHeader)
	}
	wantRow := []string{"acc_00101", "アカウントA", "2026-07-01", "10", "2", "1", "100000", "20", "125000"}
	if strings.Join(records[1], ",") != strings.Join(wantRow, ",") {
		t.Fatalf("row = %#v, want %#v", records[1], wantRow)
	}
}
