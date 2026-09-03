package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestRunMigrationsConcurrent(t *testing.T) {
	dsn := os.Getenv("TEST_MYSQL_DSN")
	if dsn == "" {
		t.Skip("TEST_MYSQL_DSN not set")
	}
	if !strings.Contains(dsn, "multiStatements=true") {
		if strings.Contains(dsn, "?") {
			dsn += "&multiStatements=true"
		} else {
			dsn += "?multiStatements=true"
		}
	}

	dir := t.TempDir()
	const version = "001_lock_test.sql"
	body := `CREATE TABLE IF NOT EXISTS migration_lock_probe (
		id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
		marker VARCHAR(64) NOT NULL
	);
	INSERT INTO migration_lock_probe (marker) VALUES ('once');`
	if err := os.WriteFile(filepath.Join(dir, version), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MIGRATIONS_DIR", dir)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Fatalf("mysql ping: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec("DROP TABLE IF EXISTS migration_lock_probe")
		_, _ = db.Exec("DELETE FROM schema_migrations WHERE version = ?", version)
	})
	_, _ = db.Exec("DROP TABLE IF EXISTS migration_lock_probe")
	_, _ = db.Exec("DELETE FROM schema_migrations WHERE version = ?", version)

	const n = 8
	errCh := make(chan error, n)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errCh <- runMigrations(db)
		}()
	}
	close(start)
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Errorf("runMigrations: %v", err)
		}
	}

	var recorded int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, version).Scan(&recorded); err != nil {
		t.Fatal(err)
	}
	if recorded != 1 {
		t.Fatalf("schema_migrations rows = %d, want 1", recorded)
	}

	var markers int
	if err := db.QueryRow(`SELECT COUNT(*) FROM migration_lock_probe`).Scan(&markers); err != nil {
		t.Fatal(err)
	}
	if markers != 1 {
		t.Fatalf("probe rows = %d, want 1", markers)
	}
}
