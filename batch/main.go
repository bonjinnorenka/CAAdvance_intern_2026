package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
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

func main() {
	fullRangeFlag := flag.Bool("full-range", false, "fetch all finalized dates (up to 92 days)")
	dateFlag := flag.String("date", "", "fetch a specific date (YYYY-MM-DD)")
	flag.Parse()

	if *fullRangeFlag && *dateFlag != "" {
		log.Fatal("--full-range and --date are mutually exclusive")
	}

	log.Printf("batch started (manual run represents the scheduled time)")

	dbHost := env("DB_HOST", "db")
	dbPort := env("DB_PORT", "3306")
	dbUser := env("DB_USER", "app")
	dbPassword := env("DB_PASSWORD", "app")
	dbName := env("DB_NAME", "app")
	externalURL := env("EXTERNAL_API_URL", "http://example-ads-api:8081")
	apiKey := os.Getenv("EXTERNAL_API_KEY")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&charset=utf8mb4&multiStatements=true&timeout=5s&readTimeout=5s&writeTimeout=5s",
		dbUser, dbPassword, dbHost, dbPort, dbName)
	db, err := connectDB(dsn)
	if err != nil {
		log.Fatalf("mysql connection failed: %v", err)
	}
	defer db.Close()
	log.Printf("connected to mysql at %s:%s", dbHost, dbPort)

	if err := runMigrations(db); err != nil {
		log.Fatalf("migrations failed: %v", err)
	}

	now := time.Now()
	var dr dateRange
	switch {
	case *dateFlag != "":
		dr, err = singleDateRange(*dateFlag)
		if err != nil {
			log.Fatalf("invalid --date: %v", err)
		}
		log.Printf("mode: specific date %s", formatDate(dr.from))
	case *fullRangeFlag:
		dr = fullDateRange(now)
		log.Printf("mode: full range %s..%s", formatDate(dr.from), formatDate(dr.to))
	default:
		dr = defaultDateRange(now)
		log.Printf("mode: previous day %s", formatDate(dr.from))
	}

	ranges := chunkDateRanges(dr)
	client := NewExampleAdsClient(externalURL, apiKey)
	ctx := context.Background()

	if _, err := syncAccounts(ctx, db, client); err != nil {
		log.Fatalf("account sync failed: %v", err)
	}
	log.Printf("accounts synced from %s", externalURL)

	accountIDs, err := loadActiveAccountIDs(ctx, db)
	if err != nil {
		log.Fatalf("load active accounts failed: %v", err)
	}
	log.Printf("fetching reports for %d active accounts", len(accountIDs))

	saved, err := fetchAndSaveReports(ctx, db, client, accountIDs, ranges)
	if err != nil {
		log.Fatalf("report fetch failed: %v", err)
	}

	log.Printf("batch finished: accounts=%d rows_saved=%d", len(accountIDs), saved)
}
