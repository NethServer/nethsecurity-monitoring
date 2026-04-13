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
	db     *sql.DB
	dbPath string
}

type Saver interface {
	Save(context.Context, Payload) error
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}

	return s.db.Close()
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
	ip_protocol,
	ip_version,
	local_bytes,
	local_ip,
	local_mac,
	local_origin,
	other_bytes,
	other_ip,
	other_port,
	other_type,
	packets
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
			stat.IpProtocol,
			stat.IpVersion,
			stat.LocalBytes,
			stat.LocalIp,
			stat.LocalMac,
			boolToInt(stat.LocalOrigin),
			stat.OtherBytes,
			stat.OtherIp,
			stat.OtherPort,
			stat.OtherType,
			stat.Packets,
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

func (s *Store) DeleteOlderThan(ctx context.Context, cutoff int64) error {
	if _, err := s.db.ExecContext(
		ctx,
		`DELETE FROM stats_timestamps WHERE log_time_end < ?`,
		cutoff,
	); err != nil {
		return fmt.Errorf("delete expired stats: %w", err)
	}

	return nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}

	return 0
}
