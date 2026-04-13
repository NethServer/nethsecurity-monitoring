package stats

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

const initSchema = `
CREATE TABLE IF NOT EXISTS stats_timestamps (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	log_time_start INTEGER NOT NULL,
	log_time_end INTEGER NOT NULL,
	UNIQUE(log_time_start, log_time_end)
);

CREATE TABLE IF NOT EXISTS stats (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	stats_timestamp_id INTEGER NOT NULL,
	detected_application INTEGER NOT NULL,
	detected_application_name TEXT NOT NULL,
	detected_protocol INTEGER NOT NULL,
	detected_protocol_name TEXT NOT NULL,
	internal INTEGER NOT NULL,
	ip_protocol INTEGER NOT NULL,
	ip_version INTEGER NOT NULL,
	local_bytes INTEGER NOT NULL,
	local_ip TEXT NOT NULL,
	local_mac TEXT NOT NULL,
	local_origin INTEGER NOT NULL,
	other_bytes INTEGER NOT NULL,
	other_ip TEXT NOT NULL,
	other_port INTEGER NOT NULL,
	other_type TEXT NOT NULL,
	packets INTEGER NOT NULL,
	FOREIGN KEY(stats_timestamp_id) REFERENCES stats_timestamps(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_stats_timestamps_log_time_start ON stats_timestamps (log_time_start);
CREATE INDEX IF NOT EXISTS idx_stats_timestamps_log_time_end ON stats_timestamps (log_time_end);
CREATE INDEX IF NOT EXISTS idx_stats_stats_timestamp_id ON stats (stats_timestamp_id);
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
