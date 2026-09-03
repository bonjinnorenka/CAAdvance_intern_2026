package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-sql-driver/mysql"
)

const (
	migrationLockName = "app_schema_migrations"
	// DSN readTimeout is 5s, so wait in short GET_LOCK slices and retry.
	migrationLockWaitSec  = 2
	migrationLockAttempts = 30
)

func runMigrations(db *sql.DB) error {
	dir := env("MIGRATIONS_DIR", "/migrations")
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		log.Printf("migrations dir %s not found, skipping", dir)
		return nil
	}

	ctx := context.Background()
	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("migration connection: %w", err)
	}
	defer conn.Close()

	if err := acquireMigrationLock(ctx, conn); err != nil {
		return err
	}
	defer releaseMigrationLock(ctx, conn)

	if _, err := conn.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) NOT NULL PRIMARY KEY,
		applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		var count int
		if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, name).Scan(&count); err != nil {
			return fmt.Errorf("check migration %s: %w", name, err)
		}
		if count > 0 {
			continue
		}

		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		tx, err := conn.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("begin migration %s: %w", name, err)
		}
		if _, err := tx.Exec(string(body)); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", name, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, name); err != nil {
			_ = tx.Rollback()
			if isDuplicateKey(err) {
				log.Printf("migration %s already recorded by another process", name)
				continue
			}
			return fmt.Errorf("record migration %s: %w", name, err)
		}
		if err := tx.Commit(); err != nil {
			if isDuplicateKey(err) {
				log.Printf("migration %s already recorded by another process", name)
				continue
			}
			return fmt.Errorf("commit migration %s: %w", name, err)
		}
		log.Printf("applied migration %s", name)
	}

	return nil
}

func acquireMigrationLock(ctx context.Context, conn *sql.Conn) error {
	for i := 0; i < migrationLockAttempts; i++ {
		var acquired sql.NullInt64
		if err := conn.QueryRowContext(ctx, `SELECT GET_LOCK(?, ?)`, migrationLockName, migrationLockWaitSec).Scan(&acquired); err != nil {
			return fmt.Errorf("acquire migration lock: %w", err)
		}
		if acquired.Valid && acquired.Int64 == 1 {
			return nil
		}
	}
	return fmt.Errorf("acquire migration lock: timed out")
}

func releaseMigrationLock(ctx context.Context, conn *sql.Conn) {
	if _, err := conn.ExecContext(ctx, `SELECT RELEASE_LOCK(?)`, migrationLockName); err != nil {
		log.Printf("release migration lock: %v", err)
	}
}

func isDuplicateKey(err error) bool {
	return errors.Is(err, &mysql.MySQLError{Number: 1062})
}
