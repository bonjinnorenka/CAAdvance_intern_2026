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

func handleJob(ctx context.Context, db *sql.DB, exportDir string, job Job) error {
	if job.Type != "generate_report" {
		return fmt.Errorf("unknown job type: %s", job.Type)
	}
	return handleGenerateReport(ctx, db, exportDir, job)
}

func main() {
	dbHost := env("DB_HOST", "db")
	dbPort := env("DB_PORT", "3306")
	dbUser := env("DB_USER", "app")
	dbPassword := env("DB_PASSWORD", "app")
	dbName := env("DB_NAME", "app")
	redisAddr := env("REDIS_ADDR", "queue:6379")
	exportDir := env("EXPORT_DIR", "/data/exports")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&multiStatements=true&timeout=5s&readTimeout=30s&writeTimeout=30s", dbUser, dbPassword, dbHost, dbPort, dbName)
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
	log.Printf("reportGenerateBatch waiting for jobs on %s (BLPOP, FIFO)", jobsKey)
	log.Printf("export dir: %s", exportDir)

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
		if err := handleJob(ctx, db, exportDir, job); err != nil {
			log.Printf("job %s failed: %v", job.ID, err)
			continue
		}
		log.Printf("job %s done", job.ID)
	}
}
