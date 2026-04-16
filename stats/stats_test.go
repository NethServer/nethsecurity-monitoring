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

		payload1 := Payload{
			LogTimeEnd: 3661,
			Stats: []Statistic{{
				DetectedApplication:     10033,
				DetectedApplicationName: "netify.netify",
				DetectedProtocol:        196,
				DetectedProtocolName:    "HTTP/S",
				LocalIp:                 "10.0.0.1",
				OtherIp:                 "10.0.0.2",
				LocalBytes:              100,
				OtherBytes:              200,
				LocalOrigin:             true,
			}},
		}

		payload2 := Payload{
			LogTimeEnd: 3700,
			Stats: []Statistic{{
				DetectedApplication:     10033,
				DetectedApplicationName: "netify.netify",
				DetectedProtocol:        196,
				DetectedProtocolName:    "HTTP/S",
				LocalIp:                 "10.0.0.1",
				OtherIp:                 "10.0.0.2",
				LocalBytes:              50,
				OtherBytes:              75,
				LocalOrigin:             false,
			}},
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
		if count != 2 {
			t.Fatalf("expected 2 rows when directions differ, got %d", count)
		}

		var localBytes, otherBytes int64
		err = db.QueryRow(
			`SELECT local_bytes, other_bytes FROM hourly_traffic
			WHERE detected_application = ? AND detected_protocol = ?
			AND local_ip = ? AND other_ip = ? AND local_origin = ?`,
			10033,
			196,
			"10.0.0.1",
			"10.0.0.2",
			1,
		).Scan(&localBytes, &otherBytes)
		if err != nil {
			t.Fatal(err)
		}
		if localBytes != 100 || otherBytes != 200 {
			t.Fatalf(
				"expected first direction row to remain unchanged, got %d/%d",
				localBytes,
				otherBytes,
			)
		}

		err = db.QueryRow(
			`SELECT local_bytes, other_bytes FROM hourly_traffic
			WHERE detected_application = ? AND detected_protocol = ?
			AND local_ip = ? AND other_ip = ? AND local_origin = ?`,
			10033,
			196,
			"10.0.0.1",
			"10.0.0.2",
			0,
		).Scan(&localBytes, &otherBytes)
		if err != nil {
			t.Fatal(err)
		}
		if localBytes != 50 || otherBytes != 75 {
			t.Fatalf("expected second direction row, got %d/%d", localBytes, otherBytes)
		}
	})

	t.Run("calculates hour bucket correctly", func(t *testing.T) {
		store, db := setupStore(t)
		defer store.Close() //nolint:errcheck
		defer db.Close()    //nolint:errcheck

		payload := Payload{
			LogTimeEnd: 7322,
			Stats: []Statistic{{
				DetectedApplication:     10033,
				DetectedApplicationName: "netify.netify",
				DetectedProtocol:        196,
				DetectedProtocolName:    "HTTP/S",
				LocalIp:                 "10.0.0.1",
				OtherIp:                 "10.0.0.2",
				LocalBytes:              100,
				OtherBytes:              200,
				LocalOrigin:             true,
			}},
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
		LogTimeEnd: 1800,
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
		LogTimeEnd: 36000,
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
