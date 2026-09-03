package main

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
)

const demoAPIKey = "example-ads-demo-key"

type errorResponse struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type server struct {
	now     func() time.Time
	limiter *rateLimiter
}

func newServer() *server {
	return &server{
		now:     time.Now,
		limiter: newRateLimiter(60, time.Minute),
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(payload); err != nil {
		log.Printf("encode json: %v", err)
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, errorResponse{Error: errorDetail{Code: code, Message: message}})
}

func authorize(r *http.Request) bool {
	return r.Header.Get("Authorization") == "Bearer "+demoAPIKey
}

func (s *server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{
		"status":  "ok",
		"service": "exampleAdsAPI",
	})
}

func (s *server) protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.limiter.allow(s.now()) {
			writeError(w, http.StatusTooManyRequests, "rate_limited", "レート制限を超過しました（60リクエスト/分）")
			return
		}
		if !authorize(r) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "APIキーが不正です")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *server) handleV1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	switch r.URL.Path {
	case "/v1/accounts":
		s.handleAccounts(w, r)
	case "/v1/reports":
		s.handleReports(w, r)
	default:
		writeError(w, http.StatusNotFound, "not_found", "エンドポイントが見つかりません")
	}
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.handleHealth)
	mux.Handle("/v1/", s.protect(http.HandlerFunc(s.handleV1)))
	return logging(mux)
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}

func main() {
	addr := ":8081"
	log.Printf("exampleAdsAPI listening on %s", addr)
	if err := http.ListenAndServe(addr, newServer().routes()); err != nil {
		log.Fatal(err)
	}
}
