package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	dateLayout     = "2006-01-02"
	maxReportDays  = 92
	finalizeHour   = 2
	impressionsKey = "impressions"
	clicksKey      = "clicks"
	costKey        = "cost"
	conversionsKey = "conversions"
)

var (
	jst           = time.FixedZone("JST", 9*3600)
	validFields   = []string{impressionsKey, clicksKey, costKey, conversionsKey}
	validFieldSet = map[string]struct{}{
		impressionsKey: {},
		clicksKey:      {},
		costKey:        {},
		conversionsKey: {},
	}
)

type reportRow struct {
	Date    string
	Metrics metrics
	Fields  []string
}

func (r reportRow) MarshalJSON() ([]byte, error) {
	include := map[string]bool{
		impressionsKey: true,
		clicksKey:      true,
		costKey:        true,
		conversionsKey: true,
	}
	if len(r.Fields) > 0 {
		include = map[string]bool{}
		for _, f := range r.Fields {
			include[f] = true
		}
	}

	var buf bytes.Buffer
	buf.WriteString(`{"date":`)
	dateJSON, err := json.Marshal(r.Date)
	if err != nil {
		return nil, err
	}
	buf.Write(dateJSON)
	if include[impressionsKey] {
		fmt.Fprintf(&buf, `,"impressions":%d`, r.Metrics.Impressions)
	}
	if include[clicksKey] {
		fmt.Fprintf(&buf, `,"clicks":%d`, r.Metrics.Clicks)
	}
	if include[costKey] {
		fmt.Fprintf(&buf, `,"cost":%d`, r.Metrics.Cost)
	}
	if include[conversionsKey] {
		fmt.Fprintf(&buf, `,"conversions":%d`, r.Metrics.Conversions)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}

type reportsResponse struct {
	AccountID string      `json:"account_id"`
	Rows      []reportRow `json:"rows"`
}

func lastFinalizedDate(now time.Time) time.Time {
	now = now.In(jst)
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, jst)
	if now.Hour() < finalizeHour {
		return today.AddDate(0, 0, -2)
	}
	return today.AddDate(0, 0, -1)
}

func parseDate(raw, name string) (time.Time, string) {
	if raw == "" {
		return time.Time{}, name + " は必須です"
	}
	t, err := time.ParseInLocation(dateLayout, raw, jst)
	if err != nil || t.Format(dateLayout) != raw {
		return time.Time{}, name + " は YYYY-MM-DD 形式で指定してください"
	}
	return t, ""
}

func parseFields(raw string) ([]string, string) {
	if strings.TrimSpace(raw) == "" {
		return append([]string{}, validFields...), ""
	}
	parts := strings.Split(raw, ",")
	fields := make([]string, 0, len(parts))
	seen := map[string]struct{}{}
	for _, part := range parts {
		name := strings.TrimSpace(part)
		if name == "" {
			continue
		}
		if _, ok := validFieldSet[name]; !ok {
			return nil, "fields に不明な指標が含まれています"
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		fields = append(fields, name)
	}
	if len(fields) == 0 {
		return nil, "fields に不明な指標が含まれています"
	}
	return fields, ""
}

func (s *server) handleReports(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	accountID := strings.TrimSpace(q.Get("account_id"))
	if accountID == "" {
		writeError(w, http.StatusBadRequest, "invalid_request", "account_id は必須です")
		return
	}
	if _, ok := findAccount(accountID); !ok {
		writeError(w, http.StatusNotFound, "not_found", "指定された account_id は存在しません")
		return
	}

	dateFrom, msg := parseDate(q.Get("date_from"), "date_from")
	if msg != "" {
		writeError(w, http.StatusBadRequest, "invalid_request", msg)
		return
	}
	dateTo, msg := parseDate(q.Get("date_to"), "date_to")
	if msg != "" {
		writeError(w, http.StatusBadRequest, "invalid_request", msg)
		return
	}
	if dateTo.Before(dateFrom) {
		writeError(w, http.StatusBadRequest, "invalid_request", "date_to は date_from 以降を指定してください")
		return
	}
	days := int(dateTo.Sub(dateFrom).Hours()/24) + 1
	if days > maxReportDays {
		writeError(w, http.StatusBadRequest, "invalid_request", "取得期間は最大 92 日間です")
		return
	}

	fields, msg := parseFields(q.Get("fields"))
	if msg != "" {
		writeError(w, http.StatusBadRequest, "invalid_request", msg)
		return
	}

	lastFinalized := lastFinalizedDate(s.now())
	rows := make([]reportRow, 0)
	for d := dateFrom; !d.After(dateTo); d = d.AddDate(0, 0, 1) {
		if d.After(lastFinalized) {
			continue
		}
		date := d.Format(dateLayout)
		rows = append(rows, reportRow{
			Date:    date,
			Metrics: generateMetrics(accountID, date),
			Fields:  fields,
		})
	}

	writeJSON(w, http.StatusOK, reportsResponse{
		AccountID: accountID,
		Rows:      rows,
	})
}
