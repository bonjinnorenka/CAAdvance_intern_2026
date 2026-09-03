package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

type externalItem struct {
	ExternalID string `json:"externalId"`
	Title      string `json:"title"`
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("encode json: %v", err)
	}
}

func authorize(r *http.Request, expectedKey string) bool {
	return expectedKey == "" || r.Header.Get("X-API-Key") == expectedKey
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
	mux.HandleFunc("/items", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		if !authorize(r, expectedKey) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid api key"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"source":    "external-api",
			"fetchedAt": time.Now().UTC(),
			"items": []externalItem{
				{ExternalID: "ext-101", Title: "外部データのサンプル 1"},
				{ExternalID: "ext-102", Title: "外部データのサンプル 2"},
				{ExternalID: "ext-103", Title: "外部データのサンプル 3"},
			},
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
