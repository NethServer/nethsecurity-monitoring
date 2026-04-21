package stats

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Summary struct {
	Total       int64            `json:"total"`
	Protocol    map[string]int64 `json:"protocol"`
	Application map[string]int64 `json:"application"`
	Host        map[string]int64 `json:"host"`
}

type ExportTarget struct {
	HourBucket int64
	LocalIP    string
}

func (s *Store) ListExportTargets(
	ctx context.Context,
	startHour, endHour int64,
) ([]ExportTarget, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT DISTINCT hour_bucket, local_ip
FROM stats
WHERE hour_bucket >= ? AND hour_bucket <= ?
ORDER BY hour_bucket, local_ip
`, startHour, endHour)
	if err != nil {
		return nil, fmt.Errorf("list export targets: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	targets := make([]ExportTarget, 0)
	for rows.Next() {
		var target ExportTarget
		if err := rows.Scan(&target.HourBucket, &target.LocalIP); err != nil {
			return nil, fmt.Errorf("scan export target: %w", err)
		}
		targets = append(targets, target)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate export targets: %w", err)
	}

	return targets, nil
}

func (s *Store) BuildSummary(
	ctx context.Context,
	hourBucket int64,
	localIP string,
) (*Summary, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT
	detected_protocol_name,
	detected_application_name,
	local_ip,
	COALESCE(other_name, other_ip),
	SUM(bytes)
FROM stats
WHERE hour_bucket = ? AND local_ip = ?
GROUP BY detected_protocol_name, detected_application_name, other_name, local_ip, other_ip
`, hourBucket, localIP)
	if err != nil {
		return nil, fmt.Errorf("build summary: %w", err)
	}
	defer rows.Close() //nolint:errcheck

	summary := &Summary{
		Protocol:    make(map[string]int64),
		Application: make(map[string]int64),
		Host:        make(map[string]int64),
	}

	for rows.Next() {
		var protocol string
		var application string
		var localName string
		var remoteName string
		var bytes int64
		if err := rows.Scan(&protocol, &application, &localName, &remoteName, &bytes); err != nil {
			return nil, fmt.Errorf("scan summary row: %w", err)
		}

		protocol = normalizeName(protocol)
		application = normalizeName(application)
		localName = normalizeName(localName)
		remoteName = normalizeName(remoteName)

		summary.Total += bytes
		summary.Protocol[protocol] += bytes
		summary.Application[application] += bytes
		summary.Host[localName] += bytes
		summary.Host[remoteName] += bytes
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate summary rows: %w", err)
	}

	return summary, nil
}

func WriteHourSummary(root string, ts time.Time, localIP string, summary *Summary) error {
	path := filepath.Join(
		root,
		fmt.Sprintf("%04d", ts.Year()),
		fmt.Sprintf("%02d", int(ts.Month())),
		fmt.Sprintf("%02d", ts.Day()),
		localIP,
		fmt.Sprintf("%02d.json", ts.Hour()),
	)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create export directory: %w", err)
	}

	tmp := path + ".tmp"
	file, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create export file: %w", err)
	}

	enc := json.NewEncoder(file)
	enc.SetIndent("", "  ")
	if err := enc.Encode(summary); err != nil {
		_ = file.Close()
		_ = os.Remove(tmp)
		return fmt.Errorf("encode export json: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("close export file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("move export file: %w", err)
	}

	return nil
}

func normalizeName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "unknown"
	}
	return name
}
