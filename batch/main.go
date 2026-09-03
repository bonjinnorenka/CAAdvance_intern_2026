package main

import (
	"context"
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

type externalItem struct {
	ExternalID string `json:"externalId"`
	Title      string `json:"title"`
}

type itemsResponse struct {
	Source string         `json:"source"`
	Items  []externalItem `json:"items"`
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

func fetchItems(externalURL, apiKey string) ([]externalItem, error) {
	req, err := http.NewRequest(http.MethodGet, externalURL+"/items", nil)
	if err != nil {
		return nil, err
	}
	if apiKey != "" {
		req.Header.Set("X-API-Key", apiKey)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("external-api status %d: %s", res.StatusCode, string(body))
	}

	var payload itemsResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	return payload.Items, nil
}

func upsertItem(db *sql.DB, item externalItem) (int64, error) {
	_, err := db.Exec(
		`INSERT INTO items (external_id, title, status) VALUES (?, ?, 'pending')
		 ON DUPLICATE KEY UPDATE title = VALUES(title)`,
		item.ExternalID,
		item.Title,
	)
	if err != nil {
		return 0, err
	}

	var id int64
	if err := db.QueryRow(`SELECT id FROM items WHERE external_id = ?`, item.ExternalID).Scan(&id); err != nil {
		return 0, err
	}
	return id, nil
}

func main() {
	log.Printf("batch started (manual run represents the scheduled time)")

	dbHost := env("DB_HOST", "db")
	dbPort := env("DB_PORT", "3306")
	dbUser := env("DB_USER", "app")
	dbPassword := env("DB_PASSWORD", "app")
	dbName := env("DB_NAME", "app")
	redisAddr := env("REDIS_ADDR", "queue:6379")
	externalURL := env("EXTERNAL_API_URL", "http://external-api:8081")
	apiKey := os.Getenv("EXTERNAL_API_KEY")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&timeout=5s&readTimeout=5s&writeTimeout=5s", dbUser, dbPassword, dbHost, dbPort, dbName)
	db, err := connectDB(dsn)
	if err != nil {
		log.Fatalf("mysql connection failed: %v", err)
	}
	defer db.Close()
	log.Printf("connected to mysql at %s:%s", dbHost, dbPort)

	rdb, err := connectRedis(redisAddr)
	if err != nil {
		log.Fatalf("redis connection failed: %v", err)
	}
	defer rdb.Close()
	log.Printf("connected to redis at %s", redisAddr)

	items, err := fetchItems(externalURL, apiKey)
	if err != nil {
		log.Fatalf("external-api fetch failed: %v", err)
	}
	log.Printf("fetched %d items from %s", len(items), externalURL)

	ctx := context.Background()
	enqueued := 0
	for _, item := range items {
		id, err := upsertItem(db, item)
		if err != nil {
			log.Fatalf("failed to save item %s: %v", item.ExternalID, err)
		}

		var status string
		if err := db.QueryRow(`SELECT status FROM items WHERE id = ?`, id).Scan(&status); err != nil {
			log.Fatalf("failed to read item status: %v", err)
		}
		if status == "processed" {
			log.Printf("skip already processed item %s", item.ExternalID)
			continue
		}

		job, err := enqueueJob(ctx, rdb, "process_item", map[string]any{
			"item_id":     id,
			"external_id": item.ExternalID,
			"title":       item.Title,
		})
		if err != nil {
			log.Fatalf("failed to enqueue item %s: %v", item.ExternalID, err)
		}
		enqueued++
		log.Printf("queued %s as %s", item.ExternalID, job.ID)
	}

	log.Printf("batch finished: fetched=%d enqueued=%d", len(items), enqueued)
}
