package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
)

func fetchAndSaveReports(ctx context.Context, db *sql.DB, client *ExampleAdsClient, accountIDs []string, ranges []dateRange) (int, error) {
	saved := 0
	for _, accountID := range accountIDs {
		for _, r := range ranges {
			rows, err := client.GetReports(ctx, accountID, formatDate(r.from), formatDate(r.to))
			if isNotFound(err) {
				log.Printf("skip account %s: not found", accountID)
				break
			}
			if err != nil {
				return saved, fmt.Errorf("get reports for %s (%s..%s): %w", accountID, formatDate(r.from), formatDate(r.to), err)
			}

			for _, row := range rows {
				if err := upsertAdData(ctx, db, accountID, row); err != nil {
					return saved, err
				}
				saved++
			}
			log.Printf("saved %d rows for account %s (%s..%s)", len(rows), accountID, formatDate(r.from), formatDate(r.to))
		}
	}
	return saved, nil
}

func upsertAdData(ctx context.Context, db *sql.DB, accountID string, row ReportRow) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO ad_data (ad_account_id, date, impression, click, cost, conversion)
		VALUES (?, ?, ?, ?, ?, ?)
		ON DUPLICATE KEY UPDATE
			impression = VALUES(impression),
			click = VALUES(click),
			cost = VALUES(cost),
			conversion = VALUES(conversion)`,
		accountID,
		row.Date,
		row.Impressions,
		row.Clicks,
		row.Cost,
		row.Conversions,
	)
	if err != nil {
		return fmt.Errorf("upsert ad_data %s %s: %w", accountID, row.Date, err)
	}
	return nil
}
