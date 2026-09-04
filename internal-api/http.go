package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	roleAdmin  = "admin"
	roleMember = "member"

	maxNameLen = 50
)

var jst = time.FixedZone("JST", 9*60*60)

type apiErrorBody struct {
	Error apiErrorDetail `json:"error"`
}

type apiErrorDetail struct {
	Code                   string   `json:"code"`
	Message                string   `json:"message"`
	UnauthorizedAccountIDs []string `json:"unauthorized_account_ids,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(payload); err != nil {
		log.Printf("encode json: %v", err)
	}
}

func writeAPIError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, apiErrorBody{Error: apiErrorDetail{Code: code, Message: message}})
}

func writeForbiddenAccounts(w http.ResponseWriter, ids []string) {
	writeJSON(w, http.StatusForbidden, apiErrorBody{Error: apiErrorDetail{
		Code:                   "forbidden",
		Message:                "権限のない広告アカウントが指定されています",
		UnauthorizedAccountIDs: ids,
	}})
}

func decodeJSON(r *http.Request, dst any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	return dec.Decode(dst)
}

func parseIDQuery(r *http.Request) (int64, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("id"))
	if raw == "" {
		return 0, errMissingID
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errInvalidID
	}
	return id, nil
}

func formatDateTime(t time.Time) string {
	return t.In(jst).Format(time.RFC3339)
}

func uniqueStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func tooLong(value string, max int) bool {
	return utf8.RuneCountInString(value) > max
}

func validRole(role string) bool {
	return role == roleAdmin || role == roleMember
}
