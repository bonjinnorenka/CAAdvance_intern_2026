package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

type reportJob struct {
	ID         int64
	DateFrom   time.Time
	DateTo     time.Time
	MarginRate int
}

func reportFileName(id int64) string {
	return fmt.Sprintf("report-%d.csv", id)
}

func truncateReason(reason string) string {
	const max = 255
	runes := []rune(reason)
	if len(runes) <= max {
		return reason
	}
	return string(runes[:max])
}

func markReportProcessing(ctx context.Context, db *sql.DB, id int64) error {
	_, err := db.ExecContext(ctx, `
		UPDATE report
		SET status = 'processing', reason = NULL, updated_at = NOW()
		WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("mark processing: %w", err)
	}
	return nil
}

func markReportCompleted(ctx context.Context, db *sql.DB, id int64, filePath string) error {
	_, err := db.ExecContext(ctx, `
		UPDATE report
		SET status = 'completed', file_path = ?, reason = NULL, updated_at = NOW()
		WHERE id = ?`, filePath, id)
	if err != nil {
		return fmt.Errorf("mark completed: %w", err)
	}
	return nil
}

func markReportFailed(ctx context.Context, db *sql.DB, id int64, reason string) {
	_, err := db.ExecContext(ctx, `
		UPDATE report
		SET status = 'failed', reason = ?, updated_at = NOW()
		WHERE id = ?`, truncateReason(reason), id)
	if err != nil {
		log.Printf("failed to mark report %d as failed: %v", id, err)
	}
}

func loadReport(ctx context.Context, db *sql.DB, id int64) (reportJob, error) {
	var job reportJob
	err := db.QueryRowContext(ctx, `
		SELECT id, date_from, date_to, margin_rate
		FROM report
		WHERE id = ? AND (is_deleted = false OR is_deleted IS NULL)`, id).Scan(
		&job.ID, &job.DateFrom, &job.DateTo, &job.MarginRate,
	)
	if err == sql.ErrNoRows {
		return reportJob{}, fmt.Errorf("report %d not found", id)
	}
	if err != nil {
		return reportJob{}, fmt.Errorf("load report: %w", err)
	}
	return job, nil
}

func loadReportRows(ctx context.Context, db *sql.DB, job reportJob) ([]reportCSVRow, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT a.id, a.name, d.date, d.impression, d.click, d.conversion, d.cost
		FROM report_ad_account ra
		INNER JOIN ad_accounts a ON a.id = ra.ad_account_id
		INNER JOIN ad_data d ON d.ad_account_id = ra.ad_account_id
		WHERE ra.report_id = ?
		  AND d.date BETWEEN ? AND ?
		ORDER BY a.id ASC, d.date ASC`,
		job.ID, job.DateFrom.Format("2006-01-02"), job.DateTo.Format("2006-01-02"),
	)
	if err != nil {
		return nil, fmt.Errorf("query ad_data: %w", err)
	}
	defer rows.Close()

	result := make([]reportCSVRow, 0)
	for rows.Next() {
		var row reportCSVRow
		var name sql.NullString
		var impression, click, conversion, cost sql.NullInt64
		if err := rows.Scan(&row.AccountID, &name, &row.Date, &impression, &click, &conversion, &cost); err != nil {
			return nil, fmt.Errorf("scan ad_data: %w", err)
		}
		row.AccountName = name.String
		row.Impressions = int(impression.Int64)
		row.Clicks = int(click.Int64)
		row.Conversions = int(conversion.Int64)
		row.Cost = cost.Int64
		row.MarginRate = job.MarginRate
		charge, err := customerChargeJPY(row.Cost, job.MarginRate)
		if err != nil {
			return nil, err
		}
		row.CustomerCharge = charge
		result = append(result, row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func writeReportCSV(exportDir string, id int64, body []byte) (string, error) {
	if err := os.MkdirAll(exportDir, 0o755); err != nil {
		return "", fmt.Errorf("mkdir export dir: %w", err)
	}
	name := reportFileName(id)
	path := filepath.Join(exportDir, name)
	if err := os.WriteFile(path, body, 0o644); err != nil {
		return "", fmt.Errorf("write csv: %w", err)
	}
	return name, nil
}

func generateReport(ctx context.Context, db *sql.DB, exportDir string, reportID int64) error {
	job, err := loadReport(ctx, db, reportID)
	if err != nil {
		return err
	}
	if err := markReportProcessing(ctx, db, reportID); err != nil {
		return err
	}

	dataRows, err := loadReportRows(ctx, db, job)
	if err != nil {
		return err
	}
	body, err := buildReportCSV(dataRows)
	if err != nil {
		return err
	}
	filePath, err := writeReportCSV(exportDir, reportID, body)
	if err != nil {
		return err
	}
	return markReportCompleted(ctx, db, reportID, filePath)
}

func handleGenerateReport(ctx context.Context, db *sql.DB, exportDir string, job Job) error {
	reportID, ok := payloadInt(job.Payload, "report_id")
	if !ok || reportID <= 0 {
		return fmt.Errorf("report_id is required")
	}

	if err := generateReport(ctx, db, exportDir, reportID); err != nil {
		markReportFailed(ctx, db, reportID, err.Error())
		return err
	}
	return nil
}
