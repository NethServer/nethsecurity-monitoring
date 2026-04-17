package stats

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type Payload struct {
	LogTimeEnd int64       `json:"log_time_end"`
	Stats      []Statistic `json:"stats"`
}

type Statistic struct {
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

type Store struct {
	db     *sql.DB
	dbPath string
	cache  *ReverseDNSCache
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
		localName := s.cache.Resolve(ctx, stat.LocalIp)
		otherName := s.cache.Resolve(ctx, stat.OtherIp)

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

func (s *Store) PruneReverseDNSCache(now time.Time) {
	if s == nil {
		return
	}

	s.cache.Prune(now)
}

func boolToInt(v bool) int {
	if v {
		return 1
	}

	return 0
}
