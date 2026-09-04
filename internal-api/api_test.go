package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newTestAPI(t *testing.T) (*apiServer, *fakeStore, *fakeQueue) {
	t.Helper()
	store := newFakeStore()
	q := &fakeQueue{}
	api := &apiServer{
		store:     store,
		queue:     q,
		exportDir: t.TempDir(),
		now: func() time.Time {
			return time.Date(2026, 9, 3, 15, 45, 23, 0, jst)
		},
	}
	return api, store, q
}

func doRequest(t *testing.T, api *apiServer, method, path, userID, body string) *http.Response {
	t.Helper()
	mux := http.NewServeMux()
	api.register(mux)
	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, reader)
	if userID != "" {
		req.Header.Set("X-User-Id", userID)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w.Result()
}

func decodeAPIError(t *testing.T, res *http.Response) apiErrorDetail {
	t.Helper()
	defer res.Body.Close()
	var body apiErrorBody
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	return body.Error
}

func TestCreateReportAccepted(t *testing.T) {
	api, store, q := newTestAPI(t)
	res := doRequest(t, api, http.MethodPost, "/report", "1", `{
		"ad_account_ids": ["acc_00101", "acc_00102"],
		"date_from": "2026-08-01",
		"date_to": "2026-08-31",
		"margin_rate": 20
	}`)
	defer res.Body.Close()
	if res.StatusCode != http.StatusAccepted {
		t.Fatalf("status=%d", res.StatusCode)
	}
	var got reportCreateAccepted
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.JobID != 1 || got.Status != "queued" {
		t.Fatalf("got %+v", got)
	}
	if len(q.jobs) != 1 || q.jobs[0] != 1 {
		t.Fatalf("queued jobs=%v", q.jobs)
	}
	rec := store.reports[1]
	if dateOnly(rec.DateFrom) != "2026-08-01" || dateOnly(rec.DateTo) != "2026-08-31" {
		t.Fatalf("dates from=%s to=%s", dateOnly(rec.DateFrom), dateOnly(rec.DateTo))
	}
}

func TestParseDateOnlyIsCalendarDate(t *testing.T) {
	got, err := parseDateOnly("2026-08-01")
	if err != nil {
		t.Fatal(err)
	}
	if dateOnly(got) != "2026-08-01" {
		t.Fatalf("dateOnly=%s", dateOnly(got))
	}
	if got.Location() != time.UTC {
		t.Fatalf("loc=%s", got.Location())
	}
	jstMidnight := time.Date(2026, 8, 1, 0, 0, 0, 0, jst)
	if dateOnly(jstMidnight.UTC()) == "2026-08-01" {
		t.Fatal("JST midnight converted to UTC must not be used as a DATE value")
	}
}

func TestCreateReportMarginRateInvalid(t *testing.T) {
	api, _, _ := newTestAPI(t)
	res := doRequest(t, api, http.MethodPost, "/report", "1", `{
		"ad_account_ids": ["acc_00101"],
		"date_from": "2026-08-01",
		"date_to": "2026-08-31",
		"margin_rate": 100
	}`)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", res.StatusCode)
	}
	got := decodeAPIError(t, res)
	if got.Code != "invalid_request" {
		t.Fatalf("code=%s", got.Code)
	}
}

func TestCreateReportForbiddenAccounts(t *testing.T) {
	api, _, q := newTestAPI(t)
	res := doRequest(t, api, http.MethodPost, "/report", "1", `{
		"ad_account_ids": ["acc_00101", "acc_00109"],
		"date_from": "2026-08-01",
		"date_to": "2026-08-31",
		"margin_rate": 10
	}`)
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("status=%d", res.StatusCode)
	}
	got := decodeAPIError(t, res)
	if got.Code != "forbidden" {
		t.Fatalf("code=%s", got.Code)
	}
	if len(got.UnauthorizedAccountIDs) != 1 || got.UnauthorizedAccountIDs[0] != "acc_00109" {
		t.Fatalf("unauthorized=%v", got.UnauthorizedAccountIDs)
	}
	if len(q.jobs) != 0 {
		t.Fatalf("expected no queued jobs, got %v", q.jobs)
	}
}

func TestDownloadReportNotOwner(t *testing.T) {
	api, store, _ := newTestAPI(t)
	store.reports[1] = reportRecord{
		ID: 1, Name: "20260903_154523", Status: "completed", CreatedBy: 1, FilePath: "report-1.csv",
	}
	res := doRequest(t, api, http.MethodGet, "/report?id=1", "2", "")
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("status=%d", res.StatusCode)
	}
	_ = decodeAPIError(t, res)
}

func TestDownloadReportConflict(t *testing.T) {
	api, store, _ := newTestAPI(t)
	store.reports[1] = reportRecord{
		ID: 1, Name: "20260903_154523", Status: "queued", CreatedBy: 1,
	}
	res := doRequest(t, api, http.MethodGet, "/report?id=1", "1", "")
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("status=%d", res.StatusCode)
	}
	got := decodeAPIError(t, res)
	if got.Code != "conflict" {
		t.Fatalf("code=%s", got.Code)
	}
}

func TestDownloadReportCSV(t *testing.T) {
	api, store, _ := newTestAPI(t)
	path := filepath.Join(api.exportDir, "report-1.csv")
	if err := os.WriteFile(path, []byte("アカウントID\nacc_00101\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	store.reports[1] = reportRecord{
		ID:        1,
		Name:      "20260903_154523",
		Status:    "completed",
		CreatedBy: 1,
		FilePath:  "report-1.csv",
	}
	res := doRequest(t, api, http.MethodGet, "/report?id=1", "1", "")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.Contains(ct, "text/csv") {
		t.Fatalf("content-type=%s", ct)
	}
	if disp := res.Header.Get("Content-Disposition"); !strings.Contains(disp, "20260903_154523.csv") {
		t.Fatalf("disposition=%s", disp)
	}
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "acc_00101") {
		t.Fatalf("body=%s", body)
	}
}

func TestMyReports(t *testing.T) {
	api, store, _ := newTestAPI(t)
	reason := "失敗"
	store.reports[1] = reportRecord{
		ID: 1, Name: "20260903_154523", Status: "completed", CreatedBy: 1,
		CreatedAt: time.Date(2026, 9, 3, 15, 45, 23, 0, jst),
	}
	store.reports[2] = reportRecord{
		ID: 2, Name: "20260903_154601", Status: "failed", Reason: &reason, CreatedBy: 1,
		CreatedAt: time.Date(2026, 9, 3, 15, 46, 1, 0, jst),
	}
	store.reports[3] = reportRecord{
		ID: 3, Name: "other", Status: "completed", CreatedBy: 2,
	}
	store.nextReport = 4
	res := doRequest(t, api, http.MethodGet, "/me/reports", "1", "")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}
	var got []reportResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
	if got[0].ID != 2 || got[1].ID != 1 {
		t.Fatalf("order=%+v", got)
	}
}

func TestUsersForbiddenForMember(t *testing.T) {
	api, _, _ := newTestAPI(t)
	res := doRequest(t, api, http.MethodGet, "/users", "2", "")
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("status=%d", res.StatusCode)
	}
	got := decodeAPIError(t, res)
	if got.Code != "forbidden" {
		t.Fatalf("code=%s", got.Code)
	}
}

func TestUsersListForAdmin(t *testing.T) {
	api, _, _ := newTestAPI(t)
	res := doRequest(t, api, http.MethodGet, "/users", "1", "")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}
	var got []userSummaryResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len=%d", len(got))
	}
}

func TestCreateAndDeleteUser(t *testing.T) {
	api, _, _ := newTestAPI(t)
	res := doRequest(t, api, http.MethodPost, "/user", "1", `{
		"name": "NewUser",
		"role": "member",
		"ad_account_ids": ["acc_00101"]
	}`)
	res.Body.Close()
	if res.StatusCode != http.StatusCreated {
		t.Fatalf("create status=%d", res.StatusCode)
	}

	del := doRequest(t, api, http.MethodDelete, "/user?id=3", "1", "")
	del.Body.Close()
	if del.StatusCode != http.StatusNoContent {
		t.Fatalf("delete status=%d", del.StatusCode)
	}
}

func TestMissingUserHeader(t *testing.T) {
	api, _, _ := newTestAPI(t)
	res := doRequest(t, api, http.MethodGet, "/me/ad_accounts", "", "")
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("status=%d", res.StatusCode)
	}
	_ = decodeAPIError(t, res)
}

func TestUnknownUser(t *testing.T) {
	api, _, _ := newTestAPI(t)
	res := doRequest(t, api, http.MethodGet, "/me/ad_accounts", "99", "")
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status=%d", res.StatusCode)
	}
	got := decodeAPIError(t, res)
	if got.Code != "unauthorized" {
		t.Fatalf("code=%s", got.Code)
	}
}

func TestMeAdAccounts(t *testing.T) {
	api, _, _ := newTestAPI(t)
	res := doRequest(t, api, http.MethodGet, "/me/ad_accounts", "2", "")
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}
	var got []adAccount
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].ID != "acc_00106" {
		t.Fatalf("got=%+v", got)
	}
}

func TestEnqueueFailureMarksFailed(t *testing.T) {
	api, store, q := newTestAPI(t)
	q.err = errors.New("redis down")
	res := doRequest(t, api, http.MethodPost, "/report", "1", `{
		"ad_account_ids": ["acc_00101"],
		"date_from": "2026-08-01",
		"date_to": "2026-08-31",
		"margin_rate": 0
	}`)
	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", res.StatusCode)
	}
	_ = decodeAPIError(t, res)
	rec := store.reports[1]
	if rec.Status != "failed" {
		t.Fatalf("status=%s", rec.Status)
	}
}

func TestEnqueueFailureMarksFailedWhenRequestCanceled(t *testing.T) {
	api, store, q := newTestAPI(t)
	ctx, cancel := context.WithCancel(context.Background())
	q.cancelOnEnqueue = cancel
	q.err = errors.New("redis down")

	mux := http.NewServeMux()
	api.register(mux)
	req := httptest.NewRequest(http.MethodPost, "/report", strings.NewReader(`{
		"ad_account_ids": ["acc_00101"],
		"date_from": "2026-08-01",
		"date_to": "2026-08-31",
		"margin_rate": 0
	}`)).WithContext(ctx)
	req.Header.Set("X-User-Id", "1")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	res := w.Result()
	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("status=%d", res.StatusCode)
	}
	_ = decodeAPIError(t, res)
	rec := store.reports[1]
	if rec.Status != "failed" {
		t.Fatalf("status=%s, want failed after enqueue error even if the request context is canceled", rec.Status)
	}
}

func TestUpdateUser(t *testing.T) {
	api, _, _ := newTestAPI(t)
	res := doRequest(t, api, http.MethodPatch, "/user?id=2", "1", `{
		"name": "UpdatedUser",
		"role": "admin",
		"ad_account_ids": ["acc_00101"]
	}`)
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status=%d", res.StatusCode)
	}
	var got userUpdateResponse
	if err := json.NewDecoder(res.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Name != "UpdatedUser" || got.Role != "admin" || len(got.AdAccountIDs) != 1 {
		t.Fatalf("got=%+v", got)
	}
}

func TestAdAccountsAdminOnly(t *testing.T) {
	api, _, _ := newTestAPI(t)
	res := doRequest(t, api, http.MethodGet, "/ad_accounts", "2", "")
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("member status=%d", res.StatusCode)
	}
	_ = decodeAPIError(t, res)

	ok := doRequest(t, api, http.MethodGet, "/ad_accounts", "1", "")
	defer ok.Body.Close()
	if ok.StatusCode != http.StatusOK {
		t.Fatalf("admin status=%d", ok.StatusCode)
	}
}
