package main

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func TestCreateQueuedReportStoresCalendarDates(t *testing.T) {
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN not set")
	}
	if !strings.Contains(dsn, "parseTime=true") {
		if strings.Contains(dsn, "?") {
			dsn += "&parseTime=true"
		} else {
			dsn += "?parseTime=true"
		}
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Fatalf("mysql ping: %v", err)
	}

	ctx := context.Background()
	store := &dbStore{db: db}

	from, err := parseDateOnly("2026-08-01")
	if err != nil {
		t.Fatal(err)
	}
	to, err := parseDateOnly("2026-08-31")
	if err != nil {
		t.Fatal(err)
	}

	id, err := store.CreateQueuedReport(ctx, reportInsert{
		Name:         "date-regression",
		CreatedBy:    1,
		CreatedAt:    time.Date(2026, 9, 3, 15, 45, 23, 0, jst),
		DateFrom:     from,
		DateTo:       to,
		MarginRate:   20,
		AdAccountIDs: []string{"acc_00101"},
	})
	if err != nil {
		t.Fatalf("CreateQueuedReport: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(`DELETE FROM report_ad_account WHERE report_id = ?`, id)
		_, _ = db.Exec(`DELETE FROM report WHERE id = ?`, id)
	})

	var dateFrom, dateTo string
	if err := db.QueryRow(`SELECT DATE_FORMAT(date_from, '%Y-%m-%d'), DATE_FORMAT(date_to, '%Y-%m-%d') FROM report WHERE id = ?`, id).Scan(&dateFrom, &dateTo); err != nil {
		t.Fatal(err)
	}
	if dateFrom != "2026-08-01" || dateTo != "2026-08-31" {
		t.Fatalf("stored dates from=%s to=%s, want 2026-08-01..2026-08-31", dateFrom, dateTo)
	}
}
