package stats

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

const initSchema = `
CREATE TABLE IF NOT EXISTS hourly_traffic (
	hour_bucket INTEGER NOT NULL,
	detected_application INTEGER NOT NULL,
	detected_application_name TEXT NOT NULL,
	detected_protocol INTEGER NOT NULL,
	detected_protocol_name TEXT NOT NULL,
	source_ip TEXT NOT NULL,
	destination_ip TEXT NOT NULL,
	local_bytes INTEGER NOT NULL DEFAULT 0,
	other_bytes INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (
		hour_bucket,
		detected_application,
		detected_application_name,
		detected_protocol,
		detected_protocol_name,
		source_ip,
		destination_ip
	)
);

CREATE INDEX IF NOT EXISTS idx_hourly_traffic_hour_bucket ON hourly_traffic (hour_bucket);
`

func Open(ctx context.Context, dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	store := &Store{db: db, dbPath: dbPath}
	if err := store.configure(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *Store) configure(ctx context.Context) error {
	settings := []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA synchronous = NORMAL;",
	}

	if !isInMemoryDB(s.dbPath) {
		settings = append(settings, "PRAGMA journal_mode = WAL;")
	}

	for _, pragma := range settings {
		if _, err := s.db.ExecContext(ctx, pragma); err != nil {
			return fmt.Errorf("apply sqlite pragma %q: %w", pragma, err)
		}
	}

	if _, err := s.db.ExecContext(ctx, initSchema); err != nil {
		return fmt.Errorf("initialize stats schema: %w", err)
	}

	return nil
}

func isInMemoryDB(dbPath string) bool {
	return dbPath == ":memory:" || strings.HasPrefix(dbPath, "file::memory:")
}
