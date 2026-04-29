package stats

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
)

// HourReport represents the aggregated statistics for a single hour and local IP.
type HourReport struct {
	Total       int64            `json:"total"`
	Protocol    map[string]int64 `json:"protocol"`
	Application map[string]int64 `json:"application"`
	Host        map[string]int64 `json:"host"`
}

// BuildReport aggregates HourRow entries into a HourReport.
func BuildReport(rows []HourRow) HourReport {
	report := HourReport{
		Protocol:    make(map[string]int64),
		Application: make(map[string]int64),
		Host:        make(map[string]int64),
	}

	for _, row := range rows {
		report.Total += row.TotalBytes

		// Aggregate by protocol
		if row.ProtocolName != "" {
			report.Protocol[row.ProtocolName] += row.TotalBytes
		}

		// Aggregate by application
		if row.ApplicationName != "" {
			report.Application[row.ApplicationName] += row.TotalBytes
		}

		// Aggregate by host (resolved hostname or IP)
		if row.Host != "" {
			report.Host[row.Host] += row.TotalBytes
		}
	}

	return report
}

// Exporter exports hourly statistics to JSON files.
type Exporter struct {
	outputDir   string
	windowHours int
}

// NewExporter creates a new Exporter with the given output directory and window size in hours.
func NewExporter(outputDir string, windowHours int) *Exporter {
	return &Exporter{
		outputDir:   outputDir,
		windowHours: windowHours,
	}
}

// ExportAll exports the last N hours from the store to JSON files.
func (e *Exporter) ExportAll(ctx context.Context, store *Store) error {
	startTime := time.Now()
	slog.Debug("Starting stats export", "time", startTime.Format(time.RFC3339))

	hours, err := store.QueryableHours(ctx, e.windowHours)
	if err != nil {
		return fmt.Errorf("query hours: %w", err)
	}

	if len(hours) == 0 {
		slog.Debug("No hours to export")
		return nil
	}

	for _, hourEpoch := range hours {
		hourEnd := hourEpoch + 3600
		rows, err := store.QueryHour(ctx, hourEpoch, hourEnd)
		if err != nil {
			return fmt.Errorf("query hour %d: %w", hourEpoch, err)
		}

		if len(rows) == 0 {
			continue
		}

		// Group rows by local_ip
		reportsByIP := make(map[string][]HourRow)
		for _, row := range rows {
			reportsByIP[row.LocalIP] = append(reportsByIP[row.LocalIP], row)
		}

		// Write report for each local_ip
		for localIP, ipRows := range reportsByIP {
			report := BuildReport(ipRows)
			if err := e.writeReport(hourEpoch, localIP, report); err != nil {
				return fmt.Errorf("write report for %s at %d: %w", localIP, hourEpoch, err)
			}
		}
	}

	duration := time.Since(startTime)
	slog.Debug("Completed stats export", "hours_exported", len(hours), "duration", duration)
	return nil
}

// writeReport writes a HourReport to a JSON file at the appropriate path.
// The path follows the pattern: {outputDir}/{year}/{month:02d}/{day:02d}/{local_ip}/{hour:02d}.json
// Timestamps are converted to the system's local timezone.
func (e *Exporter) writeReport(hourEpoch int64, localIP string, report HourReport) error {
	t := time.Unix(hourEpoch, 0).Local()
	year := t.Year()
	month := int(t.Month())
	day := t.Day()
	hour := t.Hour()

	// Construct directory path
	dirPath := filepath.Join(
		e.outputDir,
		fmt.Sprintf("%d", year),
		fmt.Sprintf("%02d", month),
		fmt.Sprintf("%02d", day),
		localIP,
	)

	// Create directories if they don't exist
	if err := os.MkdirAll(dirPath, 0o755); err != nil {
		return fmt.Errorf("create directory %s: %w", dirPath, err)
	}

	// File path
	filePath := filepath.Join(dirPath, fmt.Sprintf("%02d.json", hour))

	// Marshal report to JSON
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal report: %w", err)
	}

	// Write to temporary file first (atomic write)
	tmpFile := filePath + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0o644); err != nil {
		return fmt.Errorf("write temp file %s: %w", tmpFile, err)
	}

	// Atomically rename temp file to final path
	if err := os.Rename(tmpFile, filePath); err != nil {
		_ = os.Remove(tmpFile) // best effort cleanup
		return fmt.Errorf("rename temp file to %s: %w", filePath, err)
	}

	return nil
}
