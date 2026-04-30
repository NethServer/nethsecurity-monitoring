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
CREATE TABLE IF NOT EXISTS aggregator_batches (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    log_time_end INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS aggregator_stats (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    batch_id INTEGER NOT NULL REFERENCES aggregator_batches(id) ON DELETE CASCADE,
    detected_application INTEGER,
    detected_application_name TEXT,
    detected_protocol INTEGER,
    detected_protocol_name TEXT,
    interface TEXT,
    ip_protocol INTEGER,
    ip_version INTEGER,
    local_bytes BIGINT,
    local_ip VARCHAR,
    local_mac VARCHAR,
    local_origin BOOLEAN,
    other_bytes BIGINT,
    other_ip VARCHAR,
    other_host TEXT,
    other_port INTEGER,
    other_type TEXT,
    packets INTEGER
);

CREATE INDEX IF NOT EXISTS idx_stats_batch_id
    ON aggregator_stats(batch_id);

CREATE INDEX IF NOT EXISTS idx_stats_covering
    ON aggregator_stats(
        batch_id,
        local_ip,
        detected_protocol_name,
        detected_application_name,
        other_ip,
        local_bytes,
        other_bytes
    );

CREATE INDEX IF NOT EXISTS idx_stats_other_ip_unresolved
    ON aggregator_stats(other_ip)
    WHERE other_host IS NULL;

CREATE INDEX IF NOT EXISTS idx_batches_log_time_end
    ON aggregator_batches(log_time_end);
`

type Store struct {
	db *sql.DB
}

type Saver interface {
	Save(context.Context, AggregatorPayload) error
}

type HourRow struct {
	LocalIP         string
	ProtocolName    string
	ApplicationName string
	Host            string
	TotalBytes      int64
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

// stripAppNameID removes a leading "<digits>." prefix from the application name
// to normalize both old-format ("10910.netify.google-chat") and
// new-format ("netify.google-chat") values sent from upstream.
// If no leading digits followed by "." are found, the name is returned unchanged.
func stripAppNameID(name string) string {
	for i, c := range name {
		if c == '.' {
			// Found a dot, check if everything before it is digits
			if i > 0 && isAllDigits(name[:i]) {
				return name[i+1:]
			}
			break
		}
	}
	return name
}

// isAllDigits checks if a string contains only digit characters.
func isAllDigits(s string) bool {
	if len(s) == 0 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
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

	// Insert batch record
	result, err := tx.ExecContext(ctx, `
INSERT INTO aggregator_batches (log_time_end)
VALUES (?)
`, payload.LogTimeEnd)
	if err != nil {
		return fmt.Errorf("insert batch record: %w", err)
	}

	batchID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("get batch id: %w", err)
	}

	// Prepare statement for stats entries
	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO aggregator_stats (
    batch_id,
    detected_application,
    detected_application_name,
    detected_protocol,
    detected_protocol_name,
    interface,
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
	defer stmt.Close() //nolint:errcheck

	// Insert all entries from the payload
	for _, stat := range payload.Stats {
		_, err = stmt.ExecContext(
			ctx,
			batchID,
			stat.DetectedApplication,
			stripAppNameID(stat.DetectedApplicationName),
			stat.DetectedProtocol,
			stat.DetectedProtocolName,
			stat.Interface,
			stat.IpProtocol,
			stat.IpVersion,
			stat.LocalBytes,
			stat.LocalIp,
			stat.LocalMac,
			stat.LocalOrigin,
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
		`DELETE FROM aggregator_batches WHERE log_time_end < ?`,
		cutoff,
	); err != nil {
		return fmt.Errorf("delete expired batches: %w", err)
	}

	return nil
}

func (s *Store) QueryableHours(ctx context.Context, limit int) ([]int64, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT DISTINCT (log_time_end / 3600) * 3600 as hour_epoch
FROM aggregator_batches
ORDER BY hour_epoch DESC
LIMIT ?
	`, limit)
	if err != nil {
		return nil, fmt.Errorf("query hours: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var hours []int64
	for rows.Next() {
		var hourEpoch int64
		if err := rows.Scan(&hourEpoch); err != nil {
			return nil, fmt.Errorf("scan hour epoch: %w", err)
		}
		hours = append(hours, hourEpoch)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate hours: %w", err)
	}

	// Sort ascending since we selected DESC
	for i, j := 0, len(hours)-1; i < j; i, j = i+1, j-1 {
		hours[i], hours[j] = hours[j], hours[i]
	}

	return hours, nil
}

func (s *Store) QueryHour(ctx context.Context, hourStart, hourEnd int64) ([]HourRow, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT
    s.local_ip,
    s.detected_protocol_name,
    s.detected_application_name,
    COALESCE(s.other_host, s.other_ip) as host,
    SUM(s.local_bytes + s.other_bytes) as total_bytes
FROM aggregator_stats s
JOIN aggregator_batches b ON s.batch_id = b.id
WHERE b.log_time_end >= ? AND b.log_time_end < ?
GROUP BY s.local_ip, s.detected_protocol_name, s.detected_application_name, COALESCE(s.other_host, s.other_ip)
ORDER BY s.local_ip, s.detected_protocol_name, s.detected_application_name, COALESCE(s.other_host, s.other_ip)
	`, hourStart, hourEnd)
	if err != nil {
		return nil, fmt.Errorf("query hour %d-%d: %w", hourStart, hourEnd, err)
	}
	defer rows.Close() //nolint:errcheck

	var result []HourRow
	for rows.Next() {
		var row HourRow
		if err := rows.Scan(
			&row.LocalIP,
			&row.ProtocolName,
			&row.ApplicationName,
			&row.Host,
			&row.TotalBytes,
		); err != nil {
			return nil, fmt.Errorf("scan hour row: %w", err)
		}
		result = append(result, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate hour rows: %w", err)
	}

	return result, nil
}

func (s *Store) QueryUnresolvedIPs(ctx context.Context) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT DISTINCT other_ip
FROM aggregator_stats
WHERE other_host IS NULL
ORDER BY other_ip
	`)
	if err != nil {
		return nil, fmt.Errorf("query unresolved IPs: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	var ips []string
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return nil, fmt.Errorf("scan IP: %w", err)
		}
		ips = append(ips, ip)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate IPs: %w", err)
	}

	return ips, nil
}

func (s *Store) SaveResolvedHost(ctx context.Context, ip, hostname string) error {
	if _, err := s.db.ExecContext(ctx, `
UPDATE aggregator_stats
SET other_host = ?
WHERE other_ip = ? AND other_host IS NULL
	`, hostname, ip); err != nil {
		return fmt.Errorf("save resolved host for %q: %w", ip, err)
	}
	return nil
}
