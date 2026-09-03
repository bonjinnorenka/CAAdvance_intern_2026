package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListAccountsPagination(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Fatalf("unexpected auth: %s", r.Header.Get("Authorization"))
		}
		switch r.URL.Path {
		case "/v1/accounts":
			page := r.URL.Query().Get("page")
			if page == "1" {
				_ = json.NewEncoder(w).Encode(accountsResponse{
					Data:  []Account{{AccountID: "acc_00101", AccountName: "A", Currency: "JPY", Status: "active"}},
					Page:  1,
					Total: 2,
				})
				return
			}
			_ = json.NewEncoder(w).Encode(accountsResponse{
				Data:  []Account{{AccountID: "acc_00102", AccountName: "B", Currency: "JPY", Status: "active"}},
				Page:  2,
				Total: 2,
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := NewExampleAdsClient(srv.URL, "test-key")
	accounts, err := client.ListAccounts(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(accounts) != 2 || accounts[0].AccountID != "acc_00101" || accounts[1].AccountID != "acc_00102" {
		t.Fatalf("unexpected accounts: %+v", accounts)
	}
}

func TestGetReportsNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"missing"}}`))
	}))
	defer srv.Close()

	client := NewExampleAdsClient(srv.URL, "test-key")
	_, err := client.GetReports(context.Background(), "acc_99999", "2026-09-01", "2026-09-01")
	if !isNotFound(err) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func TestGetReportsOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(reportsResponse{
			AccountID: "acc_00101",
			Rows: []ReportRow{{
				Date:        "2026-09-01",
				Impressions: 100,
				Clicks:      10,
				Cost:        500,
				Conversions: 2,
			}},
		})
	}))
	defer srv.Close()

	client := NewExampleAdsClient(srv.URL, "test-key")
	rows, err := client.GetReports(context.Background(), "acc_00101", "2026-09-01", "2026-09-01")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Impressions != 100 {
		t.Fatalf("unexpected rows: %+v", rows)
	}
}
