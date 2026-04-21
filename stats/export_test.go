package stats

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreListUnresolvedIPsAndResolveIP(t *testing.T) {
	store, db := setupStore(t)
	defer store.Close() //nolint:errcheck
	defer db.Close()    //nolint:errcheck

	payload := AggregatorPayload{
		LogTimeEnd: 3661,
		Stats: []AggregatorEntry{{
			DetectedApplication:     10033,
			DetectedApplicationName: "netify.netify",
			DetectedProtocol:        196,
			DetectedProtocolName:    "HTTP/S",
			LocalBytes:              100,
			LocalIp:                 "10.0.0.1",
			OtherBytes:              200,
			OtherIp:                 "10.0.0.2",
		}},
	}

	if err := store.Save(context.Background(), payload); err != nil {
		t.Fatal(err)
	}

	ips, err := store.ListUnresolvedIPs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(ips) != 2 {
		t.Fatalf("expected 2 unresolved IPs, got %d", len(ips))
	}

	if err := store.ResolveIP(context.Background(), "10.0.0.1", "host-a"); err != nil {
		t.Fatal(err)
	}
	if err := store.ResolveIP(context.Background(), "10.0.0.2", "host-b"); err != nil {
		t.Fatal(err)
	}

	ips, err = store.ListUnresolvedIPs(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(ips) != 0 {
		t.Fatalf("expected all IPs to be resolved, got %v", ips)
	}
}

func TestStoreBuildSummary(t *testing.T) {
	store, db := setupStore(t)
	defer store.Close() //nolint:errcheck
	defer db.Close()    //nolint:errcheck

	payload := AggregatorPayload{
		LogTimeEnd: 3661,
		Stats: []AggregatorEntry{
			{
				DetectedApplication:     10033,
				DetectedApplicationName: "netify.netify",
				DetectedProtocol:        196,
				DetectedProtocolName:    "HTTP/S",
				LocalBytes:              100,
				LocalIp:                 "10.0.0.1",
				OtherBytes:              200,
				OtherIp:                 "10.0.0.2",
			},
			{
				DetectedApplication:     10033,
				DetectedApplicationName: "netify.netify",
				DetectedProtocol:        196,
				DetectedProtocolName:    "HTTP/S",
				LocalBytes:              50,
				LocalIp:                 "10.0.0.1",
				OtherBytes:              25,
				OtherIp:                 "10.0.0.3",
			},
		},
	}

	if err := store.Save(context.Background(), payload); err != nil {
		t.Fatal(err)
	}
	if err := store.ResolveIP(context.Background(), "10.0.0.1", "host-a"); err != nil {
		t.Fatal(err)
	}
	if err := store.ResolveIP(context.Background(), "10.0.0.2", "host-b"); err != nil {
		t.Fatal(err)
	}
	if err := store.ResolveIP(context.Background(), "10.0.0.3", "host-c"); err != nil {
		t.Fatal(err)
	}

	summary, err := store.BuildSummary(context.Background(), 3600, "10.0.0.1")
	if err != nil {
		t.Fatal(err)
	}

	if summary.Total != 375 {
		t.Fatalf("expected total 375, got %d", summary.Total)
	}
	if summary.Protocol["HTTP/S"] != 375 {
		t.Fatalf("expected protocol total 375, got %d", summary.Protocol["HTTP/S"])
	}
	if summary.Application["netify.netify"] != 375 {
		t.Fatalf("expected application total 375, got %d", summary.Application["netify.netify"])
	}
	if summary.Host["host-a"] != 375 || summary.Host["host-b"] != 300 ||
		summary.Host["host-c"] != 75 {
		t.Fatalf("unexpected host summary: %#v", summary.Host)
	}
}

func TestWriteHourSummary(t *testing.T) {
	root := t.TempDir()
	summary := &Summary{
		Total: 123,
		Protocol: map[string]int64{
			"dns": 123,
		},
		Application: map[string]int64{
			"unknown": 123,
		},
		Host: map[string]int64{
			"host-a": 123,
		},
	}

	ts := time.Date(2026, time.April, 20, 14, 0, 0, 0, time.UTC)
	if err := WriteHourSummary(root, ts, "10.0.0.1", summary); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join(root, "2026", "04", "20", "10.0.0.1", "14.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var got Summary
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.Total != 123 {
		t.Fatalf("expected total 123, got %d", got.Total)
	}
	if got.Protocol["dns"] != 123 {
		t.Fatalf("expected dns 123, got %d", got.Protocol["dns"])
	}
}
