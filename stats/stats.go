package stats

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

type AggregatorPayload struct {
	LogTimeEnd int64             `json:"log_time_end"`
	Stats      []AggregatorEntry `json:"stats"`
}

type AggregatorEntry struct {
	DetectedApplication     int      `json:"detected_application"`
	DetectedApplicationName string   `json:"detected_application_name"`
	DetectedProtocol        int      `json:"detected_protocol"`
	DetectedProtocolName    string   `json:"detected_protocol_name"`
	Digests                 []string `json:"digests"`
	Interface               string   `json:"interface"`
	IpProtocol              int      `json:"ip_protocol"`
	IpVersion               int      `json:"ip_version"`
	LocalBytes              int64    `json:"local_bytes"`
	LocalIp                 string   `json:"local_ip"`
	LocalMac                string   `json:"local_mac"`
	LocalOrigin             bool     `json:"local_origin"`
	OtherBytes              int64    `json:"other_bytes"`
	OtherIp                 string   `json:"other_ip"`
	OtherPort               int      `json:"other_port"`
	OtherType               string   `json:"other_type"`
	Packets                 int      `json:"packets"`
}

const initSchema = `
CREATE TABLE IF NOT EXISTS stats (
    hour_bucket integer NOT NULL,
    local_ip text NOT NULL,
    local_name text,
    other_ip text NOT NULL,
    other_name text,
	bytes integer NOT NULL,
	detected_application integer NOT NULL,
	detected_application_name integer NOT NULL,
	detected_protocol integer NOT NULL,
	detected_protocol_name integer NOT NULL,
	PRIMARY KEY (hour_bucket, local_ip, other_ip, detected_application, detected_protocol)
);
`

type Store struct {
	db *sql.DB
}

type Saver interface {
	Save(context.Context, AggregatorPayload) error
}

func NewStore(ctx context.Context, dbPath string) (*Store, error) {
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

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}

	return s.db.Close()
}

func (s *Store) Save(ctx context.Context, payload AggregatorPayload) error {
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
INSERT INTO stats (
   hour_bucket,
   local_ip,
   other_ip,
   bytes,
   detected_application,
   detected_application_name,
   detected_protocol,
   detected_protocol_name
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT (hour_bucket, local_ip, other_ip, detected_application, detected_protocol)
DO UPDATE SET bytes = bytes + excluded.bytes
`)
	if err != nil {
		return fmt.Errorf("prepare hourly traffic upsert: %w", err)
	}
	defer stmt.Close() //nolint:errcheck

	hourBucket := (payload.LogTimeEnd / 3600) * 3600
	for _, stat := range payload.Stats {
		_, err = stmt.ExecContext(
			ctx,
			hourBucket,
			stat.LocalIp,
			stat.OtherIp,
			stat.LocalBytes+stat.OtherBytes,
			stat.DetectedApplication,
			stat.DetectedApplicationName,
			stat.DetectedProtocol,
			stat.DetectedProtocolName,
		)
		if err != nil {
			return fmt.Errorf("upsert hourly traffic: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit stats transaction: %w", err)
	}
	return nil
}

func (s *Store) DeleteOlderThan(ctx context.Context, cutoff int64) error {
	if _, err := s.db.ExecContext(
		ctx,
		`DELETE FROM stats WHERE hour_bucket < ?`,
		cutoff,
	); err != nil {
		return fmt.Errorf("delete expired traffic: %w", err)
	}

	return nil
}

