package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func testServerAt(now time.Time) *server {
	return &server{
		now:     func() time.Time { return now },
		limiter: newRateLimiter(60, time.Minute),
	}
}

func authedGet(handler http.Handler, path string) *http.Response {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+demoAPIKey)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	return w.Result()
}

func TestHealthNoAuth(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	newServer().routes().ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}
}

func TestAccountsUnauthorized(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/accounts", nil)
	w := httptest.NewRecorder()
	newServer().routes().ServeHTTP(w, req)
	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", res.StatusCode)
	}
}

func TestAccountsOK(t *testing.T) {
	res := authedGet(newServer().routes(), "/v1/accounts")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}
	var payload accountsResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Total != 10 || len(payload.Data) != 10 || payload.Page != 1 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if payload.Data[0].AccountID != "acc_00101" {
		t.Fatalf("first account=%s", payload.Data[0].AccountID)
	}
}

func TestAccountsPagination(t *testing.T) {
	res := authedGet(newServer().routes(), "/v1/accounts?page=2&limit=3")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}
	var payload accountsResponse
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Page != 2 || payload.Total != 10 || len(payload.Data) != 3 {
		t.Fatalf("unexpected payload: %+v", payload)
	}
	if payload.Data[0].AccountID != "acc_00104" {
		t.Fatalf("page 2 first account=%s", payload.Data[0].AccountID)
	}
}

func TestReportsUnknownAccount(t *testing.T) {
	res := authedGet(newServer().routes(), "/v1/reports?account_id=acc_99999&date_from=2026-07-01&date_to=2026-07-02")
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", res.StatusCode)
	}
}

func TestReportsInvalidRange(t *testing.T) {
	res := authedGet(newServer().routes(), "/v1/reports?account_id=acc_00101&date_from=2026-07-02&date_to=2026-07-01")
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", res.StatusCode)
	}
}

func TestReportsClipsUnfinalizedDates(t *testing.T) {
	now := time.Date(2026, 9, 3, 1, 59, 0, 0, jst)
	res := authedGet(testServerAt(now).routes(), "/v1/reports?account_id=acc_00101&date_from=2026-08-31&date_to=2026-09-05")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("status=%d body=%s", res.StatusCode, body)
	}
	var payload struct {
		AccountID string `json:"account_id"`
		Rows      []struct {
			Date string `json:"date"`
		} `json:"rows"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Rows) != 2 {
		t.Fatalf("rows=%d want 2 (08-31 and 09-01)", len(payload.Rows))
	}
	if payload.Rows[0].Date != "2026-08-31" || payload.Rows[1].Date != "2026-09-01" {
		t.Fatalf("dates=%s,%s", payload.Rows[0].Date, payload.Rows[1].Date)
	}
}

func TestReportsFieldsFilter(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, jst)
	res := authedGet(testServerAt(now).routes(), "/v1/reports?account_id=acc_00101&date_from=2026-09-01&date_to=2026-09-01&fields=impressions,clicks")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}
	var payload map[string]any
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	rows := payload["rows"].([]any)
	if len(rows) != 1 {
		t.Fatalf("rows=%d", len(rows))
	}
	row := rows[0].(map[string]any)
	if _, ok := row["date"]; !ok {
		t.Fatal("date missing")
	}
	if _, ok := row["impressions"]; !ok {
		t.Fatal("impressions missing")
	}
	if _, ok := row["clicks"]; !ok {
		t.Fatal("clicks missing")
	}
	if _, ok := row["cost"]; ok {
		t.Fatal("cost should be omitted")
	}
	if _, ok := row["conversions"]; ok {
		t.Fatal("conversions should be omitted")
	}
}

func TestRateLimitHTTP(t *testing.T) {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, jst)
	srv := testServerAt(now)
	h := srv.routes()
	for i := 0; i < 60; i++ {
		res := authedGet(h, "/v1/accounts")
		res.Body.Close()
		if res.StatusCode != http.StatusOK {
			t.Fatalf("request %d status=%d", i+1, res.StatusCode)
		}
	}
	res := authedGet(h, "/v1/accounts")
	defer res.Body.Close()
	if res.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status=%d", res.StatusCode)
	}
	health := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, health)
	if w.Result().StatusCode != http.StatusOK {
		t.Fatal("health should not be rate limited")
	}
}
