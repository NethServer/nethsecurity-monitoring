package stats

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildReport(t *testing.T) {
	t.Run("aggregates bytes correctly across protocols, applications, and IPs", func(t *testing.T) {
		rows := []HourRow{
			{
				LocalIP:         "10.0.0.1",
				ProtocolName:    "http/s",
				ApplicationName: "netify.google",
				Host:            "10.0.0.2",
				TotalBytes:      1000,
			},
			{
				LocalIP:         "10.0.0.1",
				ProtocolName:    "http/s",
				ApplicationName: "netify.google",
				Host:            "10.0.0.2",
				TotalBytes:      500,
			},
			{
				LocalIP:         "10.0.0.1",
				ProtocolName:    "quic",
				ApplicationName: "netify.youtube",
				Host:            "10.0.0.3",
				TotalBytes:      2000,
			},
		}

		report := BuildReport(rows)

		if report.Total != 3500 {
			t.Fatalf("expected total 3500, got %d", report.Total)
		}

		if report.Protocol["http/s"] != 1500 {
			t.Fatalf("expected http/s 1500, got %d", report.Protocol["http/s"])
		}
		if report.Protocol["quic"] != 2000 {
			t.Fatalf("expected quic 2000, got %d", report.Protocol["quic"])
		}

		if report.Application["netify.google"] != 1500 {
			t.Fatalf("expected netify.google 1500, got %d", report.Application["netify.google"])
		}
		if report.Application["netify.youtube"] != 2000 {
			t.Fatalf("expected netify.youtube 2000, got %d", report.Application["netify.youtube"])
		}

		if report.Host["10.0.0.2"] != 1500 {
			t.Fatalf("expected 10.0.0.2 1500, got %d", report.Host["10.0.0.2"])
		}
		if report.Host["10.0.0.3"] != 2000 {
			t.Fatalf("expected 10.0.0.3 2000, got %d", report.Host["10.0.0.3"])
		}
	})

	t.Run("returns zero-value with empty maps for empty rows", func(t *testing.T) {
		rows := []HourRow{}

		report := BuildReport(rows)

		if report.Total != 0 {
			t.Fatalf("expected total 0, got %d", report.Total)
		}
		if len(report.Protocol) != 0 {
			t.Fatalf("expected empty protocol map, got %d entries", len(report.Protocol))
		}
		if len(report.Application) != 0 {
			t.Fatalf("expected empty application map, got %d entries", len(report.Application))
		}
		if len(report.Host) != 0 {
			t.Fatalf("expected empty host map, got %d entries", len(report.Host))
		}
	})

	t.Run("skips empty protocol and application names", func(t *testing.T) {
		rows := []HourRow{
			{
				LocalIP:         "10.0.0.1",
				ProtocolName:    "",
				ApplicationName: "",
				Host:            "10.0.0.2",
				TotalBytes:      1000,
			},
		}

		report := BuildReport(rows)

		if report.Total != 1000 {
			t.Fatalf("expected total 1000, got %d", report.Total)
		}
		if len(report.Protocol) != 0 {
			t.Fatalf("expected empty protocol map, got %d", len(report.Protocol))
		}
		if len(report.Application) != 0 {
			t.Fatalf("expected empty application map, got %d", len(report.Application))
		}
	})
}

func TestExportAll(t *testing.T) {
	t.Run("exports data to correct file paths and format", func(t *testing.T) {
		store, _ := setupStore(t)
		defer store.Close() //nolint:errcheck

		tmpDir := t.TempDir()

		// Insert test data
		payload := AggregatorPayload{
			LogTimeEnd: 1800, // Jan 1, 1970, 00:30 UTC -> hour 0
			Stats: []AggregatorEntry{
				{
					DetectedApplication:     10033,
					DetectedApplicationName: "netify.google",
					DetectedProtocol:        196,
					DetectedProtocolName:    "http/s",
					IpProtocol:              6,
					IpVersion:               4,
					LocalBytes:              500,
					LocalIp:                 "192.168.1.1",
					LocalMac:                "XX:XX.XX:XX:XX:XX",
					LocalOrigin:             true,
					OtherBytes:              1000,
					OtherIp:                 "8.8.8.8",
					OtherPort:               443,
					OtherType:               "remote",
				},
			},
		}

		if err := store.Save(context.Background(), payload); err != nil {
			t.Fatal(err)
		}

		exporter := NewExporter(tmpDir, 24)
		if err := exporter.ExportAll(context.Background(), store); err != nil {
			t.Fatal(err)
		}

		// Verify file exists at correct path
		// Hour 0 on Jan 1, 1970 -> 1970/01/01/192.168.1.1/00.json
		filePath := filepath.Join(tmpDir, "1970", "01", "01", "192.168.1.1", "00.json")
		if _, err := os.Stat(filePath); err != nil {
			t.Fatalf("expected file at %s, got error: %v", filePath, err)
		}

		// Verify JSON content
		data, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatal(err)
		}

		var report HourReport
		if err := json.Unmarshal(data, &report); err != nil {
			t.Fatal(err)
		}

		if report.Total != 1500 {
			t.Fatalf("expected total 1500, got %d", report.Total)
		}
		if report.Protocol["http/s"] != 1500 {
			t.Fatalf("expected http/s 1500, got %d", report.Protocol["http/s"])
		}
		if report.Application["netify.google"] != 1500 {
			t.Fatalf("expected netify.google 1500, got %d", report.Application["netify.google"])
		}
		if report.Host["8.8.8.8"] != 1500 {
			t.Fatalf("expected 8.8.8.8 1500, got %d", report.Host["8.8.8.8"])
		}
	})

	t.Run("exports multiple local IPs to separate files", func(t *testing.T) {
		store, _ := setupStore(t)
		defer store.Close() //nolint:errcheck

		tmpDir := t.TempDir()

		payload := AggregatorPayload{
			LogTimeEnd: 1800,
			Stats: []AggregatorEntry{
				{
					DetectedApplication:     10033,
					DetectedApplicationName: "netify.google",
					DetectedProtocol:        196,
					DetectedProtocolName:    "http/s",
					IpProtocol:              6,
					IpVersion:               4,
					LocalBytes:              500,
					LocalIp:                 "192.168.1.1",
					LocalMac:                "XX:XX.XX:XX:XX:XX",
					LocalOrigin:             true,
					OtherBytes:              1000,
					OtherIp:                 "8.8.8.8",
					OtherPort:               443,
					OtherType:               "remote",
				},
				{
					DetectedApplication:     20001,
					DetectedApplicationName: "netify.youtube",
					DetectedProtocol:        200,
					DetectedProtocolName:    "quic",
					IpProtocol:              17,
					IpVersion:               4,
					LocalBytes:              300,
					LocalIp:                 "192.168.1.2",
					LocalMac:                "YY:YY.YY:YY:YY:YY",
					LocalOrigin:             true,
					OtherBytes:              700,
					OtherIp:                 "142.250.185.46",
					OtherPort:               443,
					OtherType:               "remote",
				},
			},
		}

		if err := store.Save(context.Background(), payload); err != nil {
			t.Fatal(err)
		}

		exporter := NewExporter(tmpDir, 24)
		if err := exporter.ExportAll(context.Background(), store); err != nil {
			t.Fatal(err)
		}

		// Verify both files exist
		filePath1 := filepath.Join(tmpDir, "1970", "01", "01", "192.168.1.1", "00.json")
		filePath2 := filepath.Join(tmpDir, "1970", "01", "01", "192.168.1.2", "00.json")

		if _, err := os.Stat(filePath1); err != nil {
			t.Fatalf("expected file at %s", filePath1)
		}
		if _, err := os.Stat(filePath2); err != nil {
			t.Fatalf("expected file at %s", filePath2)
		}

		// Verify distinct content
		data1, _ := os.ReadFile(filePath1)
		data2, _ := os.ReadFile(filePath2)

		var report1, report2 HourReport
		json.Unmarshal(data1, &report1) //nolint:errcheck
		json.Unmarshal(data2, &report2) //nolint:errcheck

		if report1.Application["netify.google"] != 1500 {
			t.Fatalf(
				"expected report1 netify.google 1500, got %d",
				report1.Application["netify.google"],
			)
		}
		if report2.Application["netify.youtube"] != 1000 {
			t.Fatalf(
				"expected report2 netify.youtube 1000, got %d",
				report2.Application["netify.youtube"],
			)
		}
	})

	t.Run("is idempotent (overwrites on second run)", func(t *testing.T) {
		store, _ := setupStore(t)
		defer store.Close() //nolint:errcheck

		tmpDir := t.TempDir()

		payload := AggregatorPayload{
			LogTimeEnd: 1800,
			Stats: []AggregatorEntry{
				{
					DetectedApplication:     10033,
					DetectedApplicationName: "netify.google",
					DetectedProtocol:        196,
					DetectedProtocolName:    "http/s",
					IpProtocol:              6,
					IpVersion:               4,
					LocalBytes:              500,
					LocalIp:                 "192.168.1.1",
					LocalMac:                "XX:XX.XX:XX:XX:XX",
					LocalOrigin:             true,
					OtherBytes:              1000,
					OtherIp:                 "8.8.8.8",
					OtherPort:               443,
					OtherType:               "remote",
				},
			},
		}

		if err := store.Save(context.Background(), payload); err != nil {
			t.Fatal(err)
		}

		exporter := NewExporter(tmpDir, 24)

		// First export
		if err := exporter.ExportAll(context.Background(), store); err != nil {
			t.Fatal(err)
		}

		filePath := filepath.Join(tmpDir, "1970", "01", "01", "192.168.1.1", "00.json")
		data1, _ := os.ReadFile(filePath)

		// Second export (should overwrite)
		if err := exporter.ExportAll(context.Background(), store); err != nil {
			t.Fatal(err)
		}

		data2, _ := os.ReadFile(filePath)

		if string(data1) != string(data2) {
			t.Fatalf("expected identical content after second export")
		}
	})

	t.Run("handles empty database gracefully", func(t *testing.T) {
		store, _ := setupStore(t)
		defer store.Close() //nolint:errcheck

		tmpDir := t.TempDir()

		exporter := NewExporter(tmpDir, 24)

		// Should not error with empty database
		if err := exporter.ExportAll(context.Background(), store); err != nil {
			t.Fatal(err)
		}

		// No files should be created
		entries, _ := os.ReadDir(tmpDir)
		if len(entries) != 0 {
			t.Fatalf("expected no files for empty database, got %d", len(entries))
		}
	})

	t.Run("handles multiple hours correctly", func(t *testing.T) {
		store, _ := setupStore(t)
		defer store.Close() //nolint:errcheck

		tmpDir := t.TempDir()

		// Hour 0
		payload1 := AggregatorPayload{
			LogTimeEnd: 1800,
			Stats: []AggregatorEntry{
				{
					DetectedApplication:     10033,
					DetectedApplicationName: "netify.google",
					DetectedProtocol:        196,
					DetectedProtocolName:    "http/s",
					IpProtocol:              6,
					IpVersion:               4,
					LocalBytes:              100,
					LocalIp:                 "192.168.1.1",
					LocalMac:                "XX:XX.XX:XX:XX:XX",
					LocalOrigin:             true,
					OtherBytes:              200,
					OtherIp:                 "8.8.8.8",
					OtherPort:               443,
					OtherType:               "remote",
				},
			},
		}

		// Hour 1
		payload2 := AggregatorPayload{
			LogTimeEnd: 5400,
			Stats: []AggregatorEntry{
				{
					DetectedApplication:     20001,
					DetectedApplicationName: "netify.youtube",
					DetectedProtocol:        200,
					DetectedProtocolName:    "quic",
					IpProtocol:              17,
					IpVersion:               4,
					LocalBytes:              300,
					LocalIp:                 "192.168.1.1",
					LocalMac:                "XX:XX.XX:XX:XX:XX",
					LocalOrigin:             true,
					OtherBytes:              700,
					OtherIp:                 "142.250.185.46",
					OtherPort:               443,
					OtherType:               "remote",
				},
			},
		}

		if err := store.Save(context.Background(), payload1); err != nil {
			t.Fatal(err)
		}
		if err := store.Save(context.Background(), payload2); err != nil {
			t.Fatal(err)
		}

		exporter := NewExporter(tmpDir, 24)

		if err := exporter.ExportAll(context.Background(), store); err != nil {
			t.Fatal(err)
		}

		// Verify both hours' files exist
		filePath0 := filepath.Join(tmpDir, "1970", "01", "01", "192.168.1.1", "00.json")
		filePath1 := filepath.Join(tmpDir, "1970", "01", "01", "192.168.1.1", "01.json")

		if _, err := os.Stat(filePath0); err != nil {
			t.Fatalf("expected hour 0 file at %s", filePath0)
		}
		if _, err := os.Stat(filePath1); err != nil {
			t.Fatalf("expected hour 1 file at %s", filePath1)
		}

		data0, _ := os.ReadFile(filePath0)
		data1, _ := os.ReadFile(filePath1)

		var report0, report1 HourReport
		json.Unmarshal(data0, &report0) //nolint:errcheck
		json.Unmarshal(data1, &report1) //nolint:errcheck

		if report0.Application["netify.google"] != 300 {
			t.Fatalf(
				"expected hour 0 netify.google 300, got %d",
				report0.Application["netify.google"],
			)
		}
		if report1.Application["netify.youtube"] != 1000 {
			t.Fatalf(
				"expected hour 1 netify.youtube 1000, got %d",
				report1.Application["netify.youtube"],
			)
		}
	})
}
