package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

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

func payloadString(payload map[string]any, key string) string {
	value, ok := payload[key]
	if !ok || value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

func payloadInt(payload map[string]any, key string) (int64, bool) {
	raw, ok := payload[key]
	if !ok || raw == nil {
		return 0, false
	}
	switch value := raw.(type) {
	case float64:
		return int64(value), true
	case int:
		return int64(value), true
	case int64:
		return value, true
	case string:
		n, err := strconv.ParseInt(value, 10, 64)
		return n, err == nil
	default:
		n, err := strconv.ParseInt(fmt.Sprint(value), 10, 64)
		return n, err == nil
	}
}

func writeLog(db *sql.DB, job Job, detail string) {
	_, err := db.Exec(`INSERT INTO job_logs (job_id, job_type, detail) VALUES (?, ?, ?)`, job.ID, job.Type, detail)
	if err != nil {
		log.Printf("failed to write job log: %v", err)
	}
}

func handleJob(db *sql.DB, job Job) error {
	switch job.Type {
	case "example_job":
		note := payloadString(job.Payload, "note")
		if note == "" {
			note = "example_job"
		}
		if _, err := db.Exec(`INSERT INTO messages (body) VALUES (?)`, "Worker がジョブを処理しました: "+note); err != nil {
			return err
		}
		writeLog(db, job, "example_job を処理しました")
		return nil
	case "process_item":
		itemID, ok := payloadInt(job.Payload, "item_id")
		if !ok {
			return fmt.Errorf("item_id is required")
		}
		result, err := db.Exec(`UPDATE items SET status = 'processed', processed_at = CURRENT_TIMESTAMP WHERE id = ?`, itemID)
		if err != nil {
			return err
		}
		affected, _ := result.RowsAffected()
		writeLog(db, job, fmt.Sprintf("item %d を processed に更新しました (affected=%d)", itemID, affected))
		return nil
	default:
		writeLog(db, job, "未対応の job type です")
		return fmt.Errorf("unknown job type: %s", job.Type)
	}
}

func main() {
	dbHost := env("DB_HOST", "db")
	dbPort := env("DB_PORT", "3306")
	dbUser := env("DB_USER", "app")
	dbPassword := env("DB_PASSWORD", "app")
	dbName := env("DB_NAME", "app")
	redisAddr := env("REDIS_ADDR", "queue:6379")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&multiStatements=true&timeout=5s&readTimeout=5s&writeTimeout=5s", dbUser, dbPassword, dbHost, dbPort, dbName)
	db, err := connectDB(dsn)
	if err != nil {
		log.Fatalf("mysql connection failed: %v", err)
	}
	defer db.Close()
	log.Printf("connected to mysql at %s:%s", dbHost, dbPort)

	if err := runMigrations(db); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	rdb, err := connectRedis(redisAddr)
	if err != nil {
		log.Fatalf("redis connection failed: %v", err)
	}
	defer rdb.Close()
	log.Printf("connected to redis at %s", redisAddr)
	log.Printf("worker waiting for jobs on %s (BLPOP, FIFO)", jobsKey)

	ctx := context.Background()
	for {
		job, ok, err := dequeueJob(ctx, rdb, 5*time.Second)
		if err != nil {
			log.Printf("dequeue error: %v", err)
			time.Sleep(time.Second)
			continue
		}
		if !ok {
			continue
		}

		log.Printf("received job %s type=%s", job.ID, job.Type)
		if err := handleJob(db, job); err != nil {
			log.Printf("job %s failed: %v", job.ID, err)
			continue
		}
		log.Printf("job %s done", job.ID)
	}
}
