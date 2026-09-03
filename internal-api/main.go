package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type message struct {
	ID        int       `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

type exampleResponse struct {
	Service     string  `json:"service"`
	Database    dbInfo  `json:"database"`
	ExternalAPI apiInfo `json:"externalApi"`
}

type dbInfo struct {
	Host      string    `json:"host"`
	Connected bool      `json:"connected"`
	Messages  []message `json:"messages"`
	Error     string    `json:"error,omitempty"`
}

type apiInfo struct {
	URL     string `json:"url"`
	OK      bool   `json:"ok"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("encode json: %v", err)
	}
}

func connectDB(dsn string) (*sql.DB, error) {
	var lastErr error
	for i := 0; i < 30; i++ {
		db, err := sql.Open("mysql", dsn)
		if err == nil {
			err = db.Ping()
		}
		if err == nil {
			db.SetMaxOpenConns(5)
			db.SetConnMaxLifetime(5 * time.Minute)
			return db, nil
		}
		lastErr = err
		log.Printf("waiting for mysql (%d/30): %v", i+1, lastErr)
		time.Sleep(2 * time.Second)
	}
	return nil, lastErr
}

func fetchExternal(url, apiKey string) apiInfo {
	info := apiInfo{URL: url}
	req, err := http.NewRequest(http.MethodGet, url+"/quote", nil)
	if err != nil {
		info.Error = err.Error()
		return info
	}
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		info.Error = err.Error()
		return info
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		info.Error = err.Error()
		return info
	}
	if res.StatusCode >= 400 {
		info.Error = fmt.Sprintf("status %d: %s", res.StatusCode, string(body))
		return info
	}

	var payload struct {
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		info.Error = err.Error()
		return info
	}

	info.OK = true
	info.Message = payload.Message
	return info
}

func loadMessages(db *sql.DB) ([]message, error) {
	rows, err := db.Query(`SELECT id, body, created_at FROM messages ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]message, 0)
	for rows.Next() {
		var item message
		if err := rows.Scan(&item.ID, &item.Body, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func main() {
	dbHost := env("DB_HOST", "db")
	dbPort := env("DB_PORT", "3306")
	dbUser := env("DB_USER", "app")
	dbPassword := env("DB_PASSWORD", "app")
	dbName := env("DB_NAME", "app")
	externalURL := env("EXTERNAL_API_URL", "http://external-api:8081")
	apiKey := os.Getenv("EXTERNAL_API_KEY")
	dbHostPort := dbHost + ":" + dbPort

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4", dbUser, dbPassword, dbHost, dbPort, dbName)
	db, err := connectDB(dsn)
	if err != nil {
		log.Fatalf("mysql connection failed: %v", err)
	}
	defer db.Close()
	log.Printf("connected to mysql at %s", dbHostPort)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "internal-api",
		})
	})
	mux.HandleFunc("/api/example", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		resp := exampleResponse{
			Service: "internal-api",
			Database: dbInfo{
				Host:     dbHostPort,
				Messages: []message{},
			},
			ExternalAPI: fetchExternal(externalURL, apiKey),
		}

		if err := db.Ping(); err != nil {
			resp.Database.Error = err.Error()
		} else {
			resp.Database.Connected = true
			items, queryErr := loadMessages(db)
			if queryErr != nil {
				resp.Database.Error = queryErr.Error()
			} else {
				resp.Database.Messages = items
			}
		}

		writeJSON(w, http.StatusOK, resp)
	})

	addr := ":8080"
	log.Printf("internal-api listening on %s", addr)
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
