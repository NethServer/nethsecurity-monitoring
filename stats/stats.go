package stats

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/nethserver/nethsecurity-monitoring/reverse_dns"
)

type AggregatorPayload struct {
	LogTimeEnd int64             `json:"log_time_end"`
	Stats      []AggregatorEntry `json:"stats"`
}

type AggregatorEntry struct {
	DetectedApplication     int    `json:"detected_application"`
	DetectedApplicationName string `json:"detected_application_name"`
	DetectedProtocol        int    `json:"detected_protocol"`
	DetectedProtocolName    string `json:"detected_protocol_name"`
	LocalIp                 string `json:"local_ip"`
	OtherIp                 string `json:"other_ip"`
	LocalBytes              int64  `json:"local_bytes"`
	OtherBytes              int64  `json:"other_bytes"`
	LocalOrigin             bool   `json:"local_origin"`
}

const initSchema = `
CREATE TABLE IF NOT EXISTS hourly_traffic (
	hour_bucket INTEGER NOT NULL,
	detected_application INTEGER NOT NULL,
	detected_application_name TEXT NOT NULL,
	detected_protocol INTEGER NOT NULL,
	detected_protocol_name TEXT NOT NULL,
	local_ip TEXT NOT NULL,
	local_name TEXT NOT NULL,
	other_ip TEXT NOT NULL,
	other_name TEXT NOT NULL,
	local_origin INTEGER NOT NULL,
	local_bytes INTEGER NOT NULL DEFAULT 0,
	other_bytes INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (
		hour_bucket,
		detected_application,
		detected_application_name,
		detected_protocol,
		detected_protocol_name,
		local_ip,
		other_ip,
		local_origin
	)
);

CREATE INDEX IF NOT EXISTS idx_hourly_traffic_hour_bucket ON hourly_traffic (hour_bucket);
`

type Store struct {
	db    *sql.DB
	cache *reverse_dns.Resolver
}

type Saver interface {
	Save(context.Context, AggregatorPayload) error
}

func NewStore(ctx context.Context, dbPath string, cache *reverse_dns.Resolver) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	defer func() {
		if err != nil {
			_ = db.Close()
		}
	}()

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	settings := []string{
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA synchronous = NORMAL;",
	}

	if dbPath != ":memory:" || !strings.HasPrefix(dbPath, "file::memory:") {
		settings = append(settings, "PRAGMA journal_mode = WAL;")
	}

	for _, pragma := range settings {
		if _, err := db.ExecContext(ctx, pragma); err != nil {
			return nil, fmt.Errorf("apply sqlite pragma %q: %w", pragma, err)
		}
	}

	if _, err := db.ExecContext(ctx, initSchema); err != nil {
		return nil, fmt.Errorf("initialize stats schema: %w", err)
	}

	return &Store{db: db, cache: cache}, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}

	return s.db.Close()
}

func (s *Store) Save(ctx context.Context, payload AggregatorPayload) error {
	hourBucket := (payload.LogTimeEnd / 3600) * 3600

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin stats transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO hourly_traffic (
	hour_bucket,
	detected_application,
	detected_application_name,
	detected_protocol,
	detected_protocol_name,
	local_ip,
	local_name,
	other_ip,
	other_name,
	local_origin,
	local_bytes,
	other_bytes
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(
	hour_bucket,
	detected_application,
	detected_application_name,
	detected_protocol,
	detected_protocol_name,
	local_ip,
	other_ip,
	local_origin
)
DO UPDATE SET
	local_name = excluded.local_name,
	other_name = excluded.other_name,
	local_bytes = local_bytes + excluded.local_bytes,
	other_bytes = other_bytes + excluded.other_bytes
`)
	if err != nil {
		return fmt.Errorf("prepare hourly traffic upsert: %w", err)
	}
	defer stmt.Close() //nolint:errcheck

	for _, stat := range payload.Stats {
		localName := s.cache.Lookup(ctx, stat.LocalIp)
		otherName := s.cache.Lookup(ctx, stat.OtherIp)

		_, err = stmt.ExecContext(
			ctx,
			hourBucket,
			stat.DetectedApplication,
			stat.DetectedApplicationName,
			stat.DetectedProtocol,
			stat.DetectedProtocolName,
			stat.LocalIp,
			localName,
			stat.OtherIp,
			otherName,
			boolToInt(stat.LocalOrigin),
			stat.LocalBytes,
			stat.OtherBytes,
		)
		if err != nil {
			return fmt.Errorf("upsert hourly traffic: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit stats transaction: %w", err)
	}

	stats := s.cache.Stats()
	slog.Debug(
		"DNS cache stats",
		"size", stats.Size,
		"hits", stats.Hits,
		"misses", stats.Misses,
		"miss_rate", stats.MissRate,
	)
	return nil
}

func (s *Store) DeleteOlderThan(ctx context.Context, cutoff int64) error {
	if _, err := s.db.ExecContext(
		ctx,
		`DELETE FROM hourly_traffic WHERE hour_bucket < ?`,
		cutoff,
	); err != nil {
		return fmt.Errorf("delete expired traffic: %w", err)
	}

	return nil
}

func boolToInt(v bool) int {
	if v {
		return 1
	}

	return 0
}
