package main

import (
	"context"
	"database/sql"
	"fmt"
)

func syncAccounts(ctx context.Context, db *sql.DB, client *ExampleAdsClient) ([]Account, error) {
	accounts, err := client.ListAccounts(ctx)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}

	seen := make(map[string]struct{}, len(accounts))
	for _, account := range accounts {
		seen[account.AccountID] = struct{}{}
		active := account.Status == "active"
		_, err := db.ExecContext(ctx, `
			INSERT INTO ad_accounts (id, name, currency, status, is_deleted)
			VALUES (?, ?, ?, ?, false)
			ON DUPLICATE KEY UPDATE
				name = VALUES(name),
				currency = VALUES(currency),
				status = VALUES(status),
				is_deleted = false`,
			account.AccountID,
			account.AccountName,
			account.Currency,
			active,
		)
		if err != nil {
			return nil, fmt.Errorf("upsert account %s: %w", account.AccountID, err)
		}
	}

	rows, err := db.QueryContext(ctx, `SELECT id FROM ad_accounts WHERE is_deleted = false OR is_deleted IS NULL`)
	if err != nil {
		return nil, fmt.Errorf("list db accounts: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		if _, ok := seen[id]; ok {
			continue
		}
		if _, err := db.ExecContext(ctx, `UPDATE ad_accounts SET is_deleted = true WHERE id = ?`, id); err != nil {
			return nil, fmt.Errorf("soft-delete account %s: %w", id, err)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return accounts, nil
}

func loadActiveAccountIDs(ctx context.Context, db *sql.DB) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id FROM ad_accounts
		WHERE (is_deleted = false OR is_deleted IS NULL)
		  AND status = true`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
