package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type reportQueue interface {
	EnqueueGenerate(ctx context.Context, reportID int64) error
}

func (s *apiServer) handleCreateReport(w http.ResponseWriter, r *http.Request, user userRecord) {
	var body reportCreateRequest
	if err := decodeJSON(r, &body); err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "リクエストボディが不正です")
		return
	}
	if len(body.AdAccountIDs) == 0 {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "ad_account_ids は1件以上指定してください")
		return
	}
	dateFrom, err := parseDateOnly(body.DateFrom)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "date_from の形式が不正です")
		return
	}
	dateTo, err := parseDateOnly(body.DateTo)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "date_to の形式が不正です")
		return
	}
	if dateFrom.After(dateTo) {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "date_from は date_to 以前である必要があります")
		return
	}
	if body.MarginRate < 0 || body.MarginRate >= 100 {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "margin_rate は 0 以上 100 未満です")
		return
	}

	unauthorized, err := s.store.UnauthorizedAccountIDs(r.Context(), user.ID, body.AdAccountIDs)
	if err != nil {
		log.Printf("check report permissions: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "内部エラーが発生しました")
		return
	}
	if len(unauthorized) > 0 {
		writeForbiddenAccounts(w, unauthorized)
		return
	}

	now := s.now().In(jst)
	reportID, err := s.store.CreateQueuedReport(r.Context(), reportInsert{
		Name:         now.Format("20060102_150405"),
		CreatedBy:    user.ID,
		CreatedAt:    now,
		DateFrom:     dateFrom,
		DateTo:       dateTo,
		MarginRate:   body.MarginRate,
		AdAccountIDs: body.AdAccountIDs,
	})
	if err != nil {
		log.Printf("create report: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "内部エラーが発生しました")
		return
	}

	if err := s.queue.EnqueueGenerate(r.Context(), reportID); err != nil {
		log.Printf("enqueue report %d: %v", reportID, err)
		if markErr := s.store.MarkReportFailed(r.Context(), reportID, "キューへの投入に失敗しました"); markErr != nil {
			log.Printf("mark report %d failed: %v", reportID, markErr)
		}
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "レポート作成の受付に失敗しました")
		return
	}

	writeJSON(w, http.StatusAccepted, reportCreateAccepted{JobID: reportID, Status: "queued"})
}

func (s *apiServer) handleDownloadReport(w http.ResponseWriter, r *http.Request, user userRecord) {
	id, err := parseIDQuery(r)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "invalid_request", "id クエリが不正です")
		return
	}
	rec, err := s.store.GetReport(r.Context(), id)
	if err == errNotFound {
		writeAPIError(w, http.StatusNotFound, "not_found", "レポートが見つかりません")
		return
	}
	if err != nil {
		log.Printf("get report %d: %v", id, err)
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "内部エラーが発生しました")
		return
	}
	if rec.CreatedBy != user.ID {
		writeAPIError(w, http.StatusNotFound, "not_found", "レポートが見つかりません")
		return
	}
	if rec.Status != "completed" {
		writeAPIError(w, http.StatusConflict, "conflict", "レポートがダウンロード可能な状態ではありません")
		return
	}
	if rec.FilePath == "" {
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "レポートファイルが見つかりません")
		return
	}

	path := filepath.Join(s.exportDir, filepath.Base(rec.FilePath))
	if _, err := os.Stat(path); err != nil {
		log.Printf("stat report file %s: %v", path, err)
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "レポートファイルが見つかりません")
		return
	}

	filename := rec.Name + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	http.ServeFile(w, r, path)
}

func (s *apiServer) handleMyReports(w http.ResponseWriter, r *http.Request, user userRecord) {
	reports, err := s.store.ListReportsByUser(r.Context(), user.ID)
	if err != nil {
		log.Printf("list reports: %v", err)
		writeAPIError(w, http.StatusInternalServerError, "internal_error", "内部エラーが発生しました")
		return
	}
	out := make([]reportResponse, 0, len(reports))
	for _, rec := range reports {
		out = append(out, reportResponse{
			ID:        rec.ID,
			Name:      rec.Name,
			Status:    rec.Status,
			Reason:    rec.Reason,
			CreatedAt: formatDateTime(rec.CreatedAt),
		})
	}
	writeJSON(w, http.StatusOK, out)
}
