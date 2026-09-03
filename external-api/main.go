package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("encode json: %v", err)
	}
}

func main() {
	expectedKey := os.Getenv("EXTERNAL_API_KEY")

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "external-api",
		})
	})
	mux.HandleFunc("/quote", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		if expectedKey != "" && r.Header.Get("X-API-Key") != expectedKey {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid api key"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"source":  "external-api",
			"message": "External API からの応答です。Docker 内部では http://external-api:8081 を利用しています。",
		})
	})

	addr := ":8081"
	log.Printf("external-api listening on %s", addr)
	if err := http.ListenAndServe(addr, logging(mux)); err != nil {
		log.Fatal(err)
	}
}

func logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start))
	})
}
