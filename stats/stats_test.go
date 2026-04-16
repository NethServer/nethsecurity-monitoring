package stats

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func setupStore(t *testing.T) (*Store, *sql.DB) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "stats.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}

	store, err := Open(context.Background(), dbPath)
	if err != nil {
		t.Fatal(err)
	}

	return store, db
}

func TestStoreSave(t *testing.T) {
	t.Run("stores hourly traffic aggregated by key", func(t *testing.T) {
		store, db := setupStore(t)
		defer store.Close() //nolint:errcheck
		defer db.Close()    //nolint:errcheck

		payload := Payload{
			LogTimeEnd: 3661, // Hour bucket: (3661 / 3600) * 3600 = 3600
			Stats: []Statistic{
				{
					DetectedApplication:     10033,
					DetectedApplicationName: "netify.netify",
					DetectedProtocol:        196,
					DetectedProtocolName:    "HTTP/S",
					LocalIp:                 "10.0.0.1",
					OtherIp:                 "10.0.0.2",
					LocalBytes:              100,
					OtherBytes:              200,
					LocalOrigin:             true,
				},
				{
					DetectedApplication:     20001,
					DetectedApplicationName: "app2",
					DetectedProtocol:        200,
					DetectedProtocolName:    "proto2",
					LocalIp:                 "10.0.0.3",
					OtherIp:                 "10.0.0.4",
					LocalBytes:              50,
					OtherBytes:              75,
					LocalOrigin:             false,
				},
			},
		}

		if err := store.Save(context.Background(), payload); err != nil {
			t.Fatal(err)
		}

		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM hourly_traffic").Scan(&count)
		if err != nil {
			t.Fatal(err)
		}
		if count != 2 {
			t.Fatalf("expected 2 rows, got %d", count)
		}
	})

	t.Run("upserts aggregating bytes for duplicate keys", func(t *testing.T) {
		store, db := setupStore(t)
		defer store.Close() //nolint:errcheck
		defer db.Close()    //nolint:errcheck

		// First payload
		payload1 := Payload{
			LogTimeEnd: 3661,
			Stats: []Statistic{
				{
					DetectedApplication:     10033,
					DetectedApplicationName: "netify.netify",
					DetectedProtocol:        196,
					DetectedProtocolName:    "HTTP/S",
					LocalIp:                 "10.0.0.1",
					OtherIp:                 "10.0.0.2",
					LocalBytes:              100,
					OtherBytes:              200,
					LocalOrigin:             true,
				},
			},
		}

		// Second payload with same key, should aggregate
		payload2 := Payload{
			LogTimeEnd: 3700,
			Stats: []Statistic{
				{
					DetectedApplication:     10033,
					DetectedApplicationName: "netify.netify",
					DetectedProtocol:        196,
					DetectedProtocolName:    "HTTP/S",
					LocalIp:                 "10.0.0.1",
					OtherIp:                 "10.0.0.2",
					LocalBytes:              50,
					OtherBytes:              75,
					LocalOrigin:             true,
				},
			},
		}

		if err := store.Save(context.Background(), payload1); err != nil {
			t.Fatal(err)
		}
		if err := store.Save(context.Background(), payload2); err != nil {
			t.Fatal(err)
		}

		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM hourly_traffic").Scan(&count)
		if err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("expected 1 aggregated row, got %d", count)
		}

		var localBytes, otherBytes int64
		err = db.QueryRow(
			`SELECT local_bytes, other_bytes FROM hourly_traffic
			WHERE detected_application = ? AND detected_protocol = ?
			AND source_ip = ? AND destination_ip = ?`,
			10033,
			196,
			"10.0.0.1",
			"10.0.0.2",
		).Scan(&localBytes, &otherBytes)
		if err != nil {
			t.Fatal(err)
		}
		if localBytes != 150 {
			t.Fatalf("expected 150 local_bytes, got %d", localBytes)
		}
		if otherBytes != 275 {
			t.Fatalf("expected 275 other_bytes, got %d", otherBytes)
		}
	})

	t.Run("applies direction logic correctly", func(t *testing.T) {
		store, db := setupStore(t)
		defer store.Close() //nolint:errcheck
		defer db.Close()    //nolint:errcheck

		payload := Payload{
			LogTimeEnd: 3661,
			Stats: []Statistic{
				{
					DetectedApplication:     10033,
					DetectedApplicationName: "netify.netify",
					DetectedProtocol:        196,
					DetectedProtocolName:    "HTTP/S",
					LocalIp:                 "10.0.0.1",
					OtherIp:                 "10.0.0.2",
					LocalBytes:              100,
					OtherBytes:              200,
					LocalOrigin:             true,
				},
				{
					DetectedApplication:     10033,
					DetectedApplicationName: "netify.netify",
					DetectedProtocol:        196,
					DetectedProtocolName:    "HTTP/S",
					LocalIp:                 "10.0.0.2",
					OtherIp:                 "10.0.0.1",
					LocalBytes:              50,
					OtherBytes:              75,
					LocalOrigin:             false,
				},
			},
		}

		if err := store.Save(context.Background(), payload); err != nil {
			t.Fatal(err)
		}

		// Both should aggregate to the same row (same source/dest regardless of direction)
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM hourly_traffic").Scan(&count)
		if err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("expected 1 aggregated row, got %d", count)
		}

		var sourceIp, destIp string
		err = db.QueryRow(
			`SELECT source_ip, destination_ip FROM hourly_traffic
			WHERE detected_application = ? AND detected_protocol = ?`,
			10033,
			196,
		).Scan(&sourceIp, &destIp)
		if err != nil {
			t.Fatal(err)
		}
		if sourceIp != "10.0.0.1" || destIp != "10.0.0.2" {
			t.Fatalf(
				"expected source 10.0.0.1 and dest 10.0.0.2, got source %s and dest %s",
				sourceIp,
				destIp,
			)
		}
	})

	t.Run("calculates hour bucket correctly", func(t *testing.T) {
		store, db := setupStore(t)
		defer store.Close() //nolint:errcheck
		defer db.Close()    //nolint:errcheck

		payload := Payload{
			LogTimeEnd: 7322, // (7322 / 3600) * 3600 = 7200
			Stats: []Statistic{
				{
					DetectedApplication:     10033,
					DetectedApplicationName: "netify.netify",
					DetectedProtocol:        196,
					DetectedProtocolName:    "HTTP/S",
					LocalIp:                 "10.0.0.1",
					OtherIp:                 "10.0.0.2",
					LocalBytes:              100,
					OtherBytes:              200,
					LocalOrigin:             true,
				},
			},
		}

		if err := store.Save(context.Background(), payload); err != nil {
			t.Fatal(err)
		}

		var hourBucket int64
		err := db.QueryRow("SELECT hour_bucket FROM hourly_traffic").Scan(&hourBucket)
		if err != nil {
			t.Fatal(err)
		}
		if hourBucket != 7200 {
			t.Fatalf("expected hour bucket 7200, got %d", hourBucket)
		}
	})
}

func TestStoreDeleteOlderThan(t *testing.T) {
	store, db := setupStore(t)
	defer store.Close() //nolint:errcheck
	defer db.Close()    //nolint:errcheck

	oldPayload := Payload{
		LogTimeEnd: 1800, // Hour bucket: 0
		Stats: []Statistic{{
			DetectedApplication:     10033,
			DetectedApplicationName: "old",
			DetectedProtocol:        196,
			DetectedProtocolName:    "old",
			LocalIp:                 "10.0.0.1",
			OtherIp:                 "10.0.0.2",
			LocalBytes:              100,
			OtherBytes:              200,
			LocalOrigin:             true,
		}},
	}

	newPayload := Payload{
		LogTimeEnd: 36000, // Hour bucket: 36000
		Stats: []Statistic{{
			DetectedApplication:     20001,
			DetectedApplicationName: "new",
			DetectedProtocol:        200,
			DetectedProtocolName:    "new",
			LocalIp:                 "10.0.0.3",
			OtherIp:                 "10.0.0.4",
			LocalBytes:              50,
			OtherBytes:              75,
			LocalOrigin:             false,
		}},
	}

	if err := store.Save(context.Background(), oldPayload); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(context.Background(), newPayload); err != nil {
		t.Fatal(err)
	}

	if err := store.DeleteOlderThan(context.Background(), 7200); err != nil {
		t.Fatal(err)
	}

	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM hourly_traffic").Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row after delete, got %d", count)
	}

	var appName string
	err = db.QueryRow("SELECT detected_application_name FROM hourly_traffic").Scan(&appName)
	if err != nil {
		t.Fatal(err)
	}
	if appName != "new" {
		t.Fatalf("expected 'new' to remain, got %q", appName)
	}
}
