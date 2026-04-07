package stats

import (
	"context"
	"database/sql"
	"fmt"
)

type Payload struct {
	LogTimeEnd   int64       `json:"log_time_end"`
	LogTimeStart int64       `json:"log_time_start"`
	Stats        []Statistic `json:"stats"`
}

type Statistic struct {
	DetectedApplication     int      `json:"detected_application"`
	DetectedApplicationName string   `json:"detected_application_name"`
	DetectedProtocol        int      `json:"detected_protocol"`
	DetectedProtocolName    string   `json:"detected_protocol_name"`
	Digests                 []string `json:"digests"`
	Interface               string   `json:"interface"`
	Internal                bool     `json:"internal"`
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

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return &Store{db: db}
}

func (s *Store) Init(ctx context.Context) error {
	const schema = `
PRAGMA foreign_keys = ON;

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
	local_bytes INTEGER NOT NULL,
	local_ip TEXT NOT NULL,
	local_origin INTEGER NOT NULL,
	other_bytes INTEGER NOT NULL,
	other_ip TEXT NOT NULL,
	other_type TEXT NOT NULL,
	FOREIGN KEY(stats_timestamp_id) REFERENCES stats_timestamps(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_stats_timestamps_log_time_start ON stats_timestamps (log_time_start);
CREATE INDEX IF NOT EXISTS idx_stats_timestamps_log_time_end ON stats_timestamps (log_time_end);
CREATE INDEX IF NOT EXISTS idx_stats_stats_timestamp_id ON stats (stats_timestamp_id);
`

	_, err := s.db.ExecContext(ctx, schema)
	if err != nil {
		return fmt.Errorf("create stats schema: %w", err)
	}

	return nil
}

func (s *Store) Save(ctx context.Context, payload Payload) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin stats transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	_, err = tx.ExecContext(ctx, `
INSERT OR IGNORE INTO stats_timestamps (
	log_time_start,
	log_time_end
) VALUES (?, ?)
`, payload.LogTimeStart, payload.LogTimeEnd)
	if err != nil {
		return fmt.Errorf("insert stats timestamps: %w", err)
	}

	var timestampID int64
	err = tx.QueryRowContext(ctx, `
SELECT id FROM stats_timestamps
WHERE log_time_start = ? AND log_time_end = ?
`, payload.LogTimeStart, payload.LogTimeEnd).Scan(&timestampID)
	if err != nil {
		return fmt.Errorf("load stats timestamp: %w", err)
	}

	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO stats (
	stats_timestamp_id,
	detected_application,
	detected_application_name,
	detected_protocol,
	detected_protocol_name,
	internal,
	local_bytes,
	local_ip,
	local_origin,
	other_bytes,
	other_ip,
	other_type
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`)
	if err != nil {
		return fmt.Errorf("prepare stats insert: %w", err)
	}
	defer stmt.Close()

	for _, stat := range payload.Stats {
		_, err = stmt.ExecContext(
			ctx,
			timestampID,
			stat.DetectedApplication,
			stat.DetectedApplicationName,
			stat.DetectedProtocol,
			stat.DetectedProtocolName,
			boolToInt(stat.Internal),
			stat.LocalBytes,
			stat.LocalIp,
			boolToInt(stat.LocalOrigin),
			stat.OtherBytes,
			stat.OtherIp,
			stat.OtherType,
		)
		if err != nil {
			return fmt.Errorf("insert stats entry: %w", err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit stats transaction: %w", err)
	}

	return nil
}

type Receiver struct {
	store *Store
}

func NewReceiver(store *Store) *Receiver {
	return &Receiver{store: store}
}

func (r *Receiver) Handle(ctx context.Context, payload Payload) error {
	return r.store.Save(ctx, payload)
}

func boolToInt(value bool) int {
	if value {
		return 1
	}

	return 0
}
