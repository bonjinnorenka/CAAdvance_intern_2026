package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
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

type item struct {
	ID          int        `json:"id"`
	ExternalID  string     `json:"externalId"`
	Title       string     `json:"title"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"createdAt"`
	ProcessedAt *time.Time `json:"processedAt"`
}

type jobLog struct {
	ID        int       `json:"id"`
	JobID     string    `json:"jobId"`
	JobType   string    `json:"jobType"`
	Detail    string    `json:"detail"`
	CreatedAt time.Time `json:"createdAt"`
}

type exampleResponse struct {
	Service  string    `json:"service"`
	Database dbInfo    `json:"database"`
	Queue    queueInfo `json:"queue"`
}

type dbInfo struct {
	Host      string    `json:"host"`
	Connected bool      `json:"connected"`
	Messages  []message `json:"messages"`
	Items     []item    `json:"items"`
	JobLogs   []jobLog  `json:"jobLogs"`
	Error     string    `json:"error,omitempty"`
}

type queueInfo struct {
	Addr      string `json:"addr"`
	Connected bool   `json:"connected"`
	Length    int64  `json:"length"`
	Error     string `json:"error,omitempty"`
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
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

func loadMessages(db *sql.DB) ([]message, error) {
	rows, err := db.Query(`SELECT id, body, created_at FROM messages ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]message, 0)
	for rows.Next() {
		var row message
		if err := rows.Scan(&row.ID, &row.Body, &row.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, row)
	}
	return items, rows.Err()
}

func loadItems(db *sql.DB) ([]item, error) {
	rows, err := db.Query(`SELECT id, external_id, title, status, created_at, processed_at FROM items ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]item, 0)
	for rows.Next() {
		var row item
		if err := rows.Scan(&row.ID, &row.ExternalID, &row.Title, &row.Status, &row.CreatedAt, &row.ProcessedAt); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func loadJobLogs(db *sql.DB) ([]jobLog, error) {
	rows, err := db.Query(`SELECT id, job_id, job_type, detail, created_at FROM job_logs ORDER BY id DESC LIMIT 20`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]jobLog, 0)
	for rows.Next() {
		var row jobLog
		if err := rows.Scan(&row.ID, &row.JobID, &row.JobType, &row.Detail, &row.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func main() {
	dbHost := env("DB_HOST", "db")
	dbPort := env("DB_PORT", "3306")
	dbUser := env("DB_USER", "app")
	dbPassword := env("DB_PASSWORD", "app")
	dbName := env("DB_NAME", "app")
	redisAddr := env("REDIS_ADDR", "queue:6379")
	dbHostPort := dbHost + ":" + dbPort

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&multiStatements=true&timeout=5s&readTimeout=5s&writeTimeout=5s", dbUser, dbPassword, dbHost, dbPort, dbName)
	db, err := connectDB(dsn)
	if err != nil {
		log.Fatalf("mysql connection failed: %v", err)
	}
	defer db.Close()
	log.Printf("connected to mysql at %s", dbHostPort)

	if err := runMigrations(db); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	rdb, err := connectRedis(redisAddr)
	if err != nil {
		log.Fatalf("redis connection failed: %v", err)
	}
	defer rdb.Close()
	log.Printf("connected to redis at %s", redisAddr)

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"status":  "ok",
			"service": "internal-api",
		})
	})

	api := &apiServer{
		store:     &dbStore{db: db},
		queue:     redisReportQueue{client: rdb},
		exportDir: env("EXPORT_DIR", "/data/exports"),
		now:       time.Now,
	}
	api.register(mux)

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
				Items:    []item{},
				JobLogs:  []jobLog{},
			},
			Queue: queueInfo{Addr: redisAddr},
		}

		if err := db.Ping(); err != nil {
			resp.Database.Error = err.Error()
		} else {
			resp.Database.Connected = true
			if messages, queryErr := loadMessages(db); queryErr != nil {
				resp.Database.Error = queryErr.Error()
			} else {
				resp.Database.Messages = messages
			}
			if items, queryErr := loadItems(db); queryErr != nil {
				resp.Database.Error = queryErr.Error()
			} else {
				resp.Database.Items = items
			}
			if logs, queryErr := loadJobLogs(db); queryErr != nil {
				resp.Database.Error = queryErr.Error()
			} else {
				resp.Database.JobLogs = logs
			}
		}

		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()
		if err := rdb.Ping(ctx).Err(); err != nil {
			resp.Queue.Error = err.Error()
		} else {
			resp.Queue.Connected = true
			length, lenErr := queueLength(ctx, rdb)
			if lenErr != nil {
				resp.Queue.Error = lenErr.Error()
			} else {
				resp.Queue.Length = length
			}
		}

		writeJSON(w, http.StatusOK, resp)
	})
	mux.HandleFunc("/api/jobs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}

		var body struct {
			Note string `json:"note"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err.Error() != "EOF" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid json"})
			return
		}
		if body.Note == "" {
			body.Note = "frontend からのジョブ"
		}

		ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
		defer cancel()
		job, err := enqueueJob(ctx, rdb, "example_job", map[string]any{"note": body.Note})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		writeJSON(w, http.StatusAccepted, map[string]any{
			"status": "queued",
			"job":    job,
		})
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
