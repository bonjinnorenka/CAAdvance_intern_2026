package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	errNotFound  = errors.New("not found")
	errMissingID = errors.New("missing id")
	errInvalidID = errors.New("invalid id")
)

type Store interface {
	GetUser(ctx context.Context, id int64) (userRecord, error)
	ListUsers(ctx context.Context) ([]userRecord, error)
	GetUserDetail(ctx context.Context, id int64) (userRecord, []string, error)
	CreateUser(ctx context.Context, in userCreateInput) (int64, error)
	UpdateUser(ctx context.Context, id int64, in userUpdateInput) (userRecord, []string, error)
	SoftDeleteUser(ctx context.Context, id int64) error

	ListUserAdAccounts(ctx context.Context, userID int64) ([]adAccount, error)
	ListAllAdAccounts(ctx context.Context) ([]adAccount, error)
	UnauthorizedAccountIDs(ctx context.Context, userID int64, accountIDs []string) ([]string, error)
	MissingAdAccountIDs(ctx context.Context, accountIDs []string) ([]string, error)

	CreateQueuedReport(ctx context.Context, in reportInsert) (int64, error)
	MarkReportFailed(ctx context.Context, id int64, reason string) error
	GetReport(ctx context.Context, id int64) (reportRecord, error)
	ListReportsByUser(ctx context.Context, userID int64) ([]reportRecord, error)
}

type dbStore struct {
	db *sql.DB
}

func (s *dbStore) GetUser(ctx context.Context, id int64) (userRecord, error) {
	var user userRecord
	var name sql.NullString
	var role sql.NullString
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, role
		FROM users
		WHERE id = ? AND (is_deleted = false OR is_deleted IS NULL)`, id).Scan(&user.ID, &name, &role)
	if err == sql.ErrNoRows {
		return userRecord{}, errNotFound
	}
	if err != nil {
		return userRecord{}, err
	}
	user.Name = name.String
	user.Role = role.String
	return user, nil
}

func (s *dbStore) ListUsers(ctx context.Context) ([]userRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, role, created_at
		FROM users
		WHERE (is_deleted = false OR is_deleted IS NULL)
		ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]userRecord, 0)
	for rows.Next() {
		var user userRecord
		var name, role sql.NullString
		var created sql.NullTime
		if err := rows.Scan(&user.ID, &name, &role, &created); err != nil {
			return nil, err
		}
		user.Name = name.String
		user.Role = role.String
		user.CreatedAt = created.Time
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *dbStore) GetUserDetail(ctx context.Context, id int64) (userRecord, []string, error) {
	var user userRecord
	var name, role sql.NullString
	var created, updated sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, role, created_at, updated_at
		FROM users
		WHERE id = ? AND (is_deleted = false OR is_deleted IS NULL)`, id).Scan(
		&user.ID, &name, &role, &created, &updated,
	)
	if err == sql.ErrNoRows {
		return userRecord{}, nil, errNotFound
	}
	if err != nil {
		return userRecord{}, nil, err
	}
	user.Name = name.String
	user.Role = role.String
	user.CreatedAt = created.Time
	user.UpdatedAt = updated.Time

	ids, err := s.listPermissionIDs(ctx, s.db, id)
	if err != nil {
		return userRecord{}, nil, err
	}
	return user, ids, nil
}

func (s *dbStore) CreateUser(ctx context.Context, in userCreateInput) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now()
	res, err := tx.ExecContext(ctx, `
		INSERT INTO users (name, created_at, updated_at, role, is_deleted)
		VALUES (?, ?, ?, ?, false)`, in.Name, now, now, in.Role)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	if err := replacePermissionsTx(ctx, tx, id, in.AdAccountIDs); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *dbStore) UpdateUser(ctx context.Context, id int64, in userUpdateInput) (userRecord, []string, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return userRecord{}, nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var user userRecord
	var name, role sql.NullString
	var created, updated sql.NullTime
	err = tx.QueryRowContext(ctx, `
		SELECT id, name, role, created_at, updated_at
		FROM users
		WHERE id = ? AND (is_deleted = false OR is_deleted IS NULL)
		FOR UPDATE`, id).Scan(&user.ID, &name, &role, &created, &updated)
	if err == sql.ErrNoRows {
		return userRecord{}, nil, errNotFound
	}
	if err != nil {
		return userRecord{}, nil, err
	}
	user.Name = name.String
	user.Role = role.String
	user.CreatedAt = created.Time
	user.UpdatedAt = updated.Time

	if in.Name != nil {
		user.Name = *in.Name
	}
	if in.Role != nil {
		user.Role = *in.Role
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE users
		SET name = ?, role = ?, updated_at = ?
		WHERE id = ?`, user.Name, user.Role, time.Now(), id); err != nil {
		return userRecord{}, nil, err
	}

	accountIDs, err := listPermissionIDsTx(ctx, tx, id)
	if err != nil {
		return userRecord{}, nil, err
	}
	if in.AdAccountIDs != nil {
		if err := replacePermissionsTx(ctx, tx, id, *in.AdAccountIDs); err != nil {
			return userRecord{}, nil, err
		}
		accountIDs = uniqueStrings(*in.AdAccountIDs)
	}
	if err := tx.Commit(); err != nil {
		return userRecord{}, nil, err
	}
	return user, accountIDs, nil
}

func (s *dbStore) SoftDeleteUser(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET is_deleted = true, updated_at = ?
		WHERE id = ? AND (is_deleted = false OR is_deleted IS NULL)`, time.Now(), id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return errNotFound
	}
	return nil
}

func (s *dbStore) ListUserAdAccounts(ctx context.Context, userID int64) ([]adAccount, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT a.id, a.name
		FROM ad_accounts a
		INNER JOIN user_ad_account_permissions p ON p.ad_account_id = a.id
		WHERE p.user_id = ?
		  AND (a.is_deleted = false OR a.is_deleted IS NULL)
		ORDER BY a.id ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAdAccounts(rows)
}

func (s *dbStore) ListAllAdAccounts(ctx context.Context) ([]adAccount, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name
		FROM ad_accounts
		WHERE (is_deleted = false OR is_deleted IS NULL)
		ORDER BY id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanAdAccounts(rows)
}

func (s *dbStore) UnauthorizedAccountIDs(ctx context.Context, userID int64, accountIDs []string) ([]string, error) {
	ids := uniqueStrings(accountIDs)
	if len(ids) == 0 {
		return nil, nil
	}
	args := make([]any, 0, len(ids)+1)
	args = append(args, userID)
	for _, id := range ids {
		args = append(args, id)
	}
	q := fmt.Sprintf(`
		SELECT a.id
		FROM ad_accounts a
		INNER JOIN user_ad_account_permissions p
			ON p.ad_account_id = a.id AND p.user_id = ?
		WHERE a.id IN (%s)
		  AND (a.is_deleted = false OR a.is_deleted IS NULL)`, placeholders(len(ids)))
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allowed := make(map[string]struct{})
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		allowed[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return missingFrom(accountIDs, allowed), nil
}

func (s *dbStore) MissingAdAccountIDs(ctx context.Context, accountIDs []string) ([]string, error) {
	ids := uniqueStrings(accountIDs)
	if len(ids) == 0 {
		return nil, nil
	}
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	q := fmt.Sprintf(`
		SELECT id
		FROM ad_accounts
		WHERE id IN (%s)
		  AND (is_deleted = false OR is_deleted IS NULL)`, placeholders(len(ids)))
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	found := make(map[string]struct{})
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		found[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return missingFrom(accountIDs, found), nil
}

func (s *dbStore) CreateQueuedReport(ctx context.Context, in reportInsert) (int64, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	res, err := tx.ExecContext(ctx, `
		INSERT INTO report (
			created_at, updated_at, name, created_by, status, reason,
			date_from, date_to, margin_rate, is_deleted
		) VALUES (?, ?, ?, ?, 'queued', NULL, ?, ?, ?, false)`,
		in.CreatedAt, in.CreatedAt, in.Name, in.CreatedBy, in.DateFrom, in.DateTo, in.MarginRate)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	for _, accountID := range uniqueStrings(in.AdAccountIDs) {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO report_ad_account (report_id, ad_account_id) VALUES (?, ?)`, id, accountID); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *dbStore) MarkReportFailed(ctx context.Context, id int64, reason string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE report
		SET status = 'failed', reason = ?, updated_at = ?
		WHERE id = ?`, truncateReason(reason), time.Now(), id)
	return err
}

func (s *dbStore) GetReport(ctx context.Context, id int64) (reportRecord, error) {
	var rec reportRecord
	var reason sql.NullString
	var filePath sql.NullString
	var created sql.NullTime
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, status, reason, created_at, file_path, created_by
		FROM report
		WHERE id = ? AND (is_deleted = false OR is_deleted IS NULL)`, id).Scan(
		&rec.ID, &rec.Name, &rec.Status, &reason, &created, &filePath, &rec.CreatedBy,
	)
	if err == sql.ErrNoRows {
		return reportRecord{}, errNotFound
	}
	if err != nil {
		return reportRecord{}, err
	}
	if reason.Valid {
		rec.Reason = &reason.String
	}
	rec.CreatedAt = created.Time
	rec.FilePath = filePath.String
	return rec, nil
}

func (s *dbStore) ListReportsByUser(ctx context.Context, userID int64) ([]reportRecord, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, status, reason, created_at
		FROM report
		WHERE created_by = ? AND (is_deleted = false OR is_deleted IS NULL)
		ORDER BY id DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	reports := make([]reportRecord, 0)
	for rows.Next() {
		var rec reportRecord
		var reason sql.NullString
		var created sql.NullTime
		if err := rows.Scan(&rec.ID, &rec.Name, &rec.Status, &reason, &created); err != nil {
			return nil, err
		}
		if reason.Valid {
			rec.Reason = &reason.String
		}
		rec.CreatedAt = created.Time
		reports = append(reports, rec)
	}
	return reports, rows.Err()
}

func (s *dbStore) listPermissionIDs(ctx context.Context, db *sql.DB, userID int64) ([]string, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT ad_account_id
		FROM user_ad_account_permissions
		WHERE user_id = ?
		ORDER BY ad_account_id ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStrings(rows)
}

type execQuerier interface {
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

func listPermissionIDsTx(ctx context.Context, tx execQuerier, userID int64) ([]string, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT ad_account_id
		FROM user_ad_account_permissions
		WHERE user_id = ?
		ORDER BY ad_account_id ASC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanStrings(rows)
}

func replacePermissionsTx(ctx context.Context, tx execQuerier, userID int64, accountIDs []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_ad_account_permissions WHERE user_id = ?`, userID); err != nil {
		return err
	}
	for _, accountID := range uniqueStrings(accountIDs) {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO user_ad_account_permissions (user_id, ad_account_id) VALUES (?, ?)`, userID, accountID); err != nil {
			return err
		}
	}
	return nil
}

func scanAdAccounts(rows *sql.Rows) ([]adAccount, error) {
	accounts := make([]adAccount, 0)
	for rows.Next() {
		var acc adAccount
		var name sql.NullString
		if err := rows.Scan(&acc.ID, &name); err != nil {
			return nil, err
		}
		acc.Name = name.String
		accounts = append(accounts, acc)
	}
	return accounts, rows.Err()
}

func scanStrings(rows *sql.Rows) ([]string, error) {
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

func missingFrom(requested []string, found map[string]struct{}) []string {
	out := make([]string, 0)
	seen := make(map[string]struct{})
	for _, id := range requested {
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		if _, ok := found[id]; !ok {
			out = append(out, id)
		}
	}
	return out
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	return strings.Repeat("?,", n-1) + "?"
}

func truncateReason(reason string) string {
	const max = 255
	runes := []rune(reason)
	if len(runes) <= max {
		return reason
	}
	return string(runes[:max])
}
