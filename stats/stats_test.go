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

	store, err := NewStore(context.Background(), dbPath)
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

		payload := AggregatorPayload{
			LogTimeEnd: 3661,
			Stats: []AggregatorEntry{
				{
					DetectedApplication:     10033,
					DetectedApplicationName: "netify.netify",
					DetectedProtocol:        196,
					DetectedProtocolName:    "HTTP/S",
					IpProtocol:              6,
					IpVersion:               4,
					LocalBytes:              100,
					LocalIp:                 "10.0.0.1",
					LocalMac:                "XX:XX.XX:XX:XX:XX",
					LocalOrigin:             true,
					OtherBytes:              200,
					OtherIp:                 "10.0.0.2",
					OtherPort:               80,
					OtherType:               "remote",
				},
				{
					DetectedApplication:     20001,
					DetectedApplicationName: "app2",
					DetectedProtocol:        200,
					DetectedProtocolName:    "proto2",
					IpProtocol:              0,
					IpVersion:               4,
					LocalBytes:              50,
					LocalIp:                 "10.0.0.3",
					LocalMac:                "YY:YY.YY:YY:YY:YY",
					LocalOrigin:             false,
					OtherBytes:              75,
					OtherIp:                 "10.0.0.4",
					OtherPort:               80,
					OtherType:               "unknown",
				},
			},
		}

		if err := store.Save(context.Background(), payload); err != nil {
			t.Fatal(err)
		}

		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM aggregator_stats").Scan(&count)
		if err != nil {
			t.Fatal(err)
		}
		if count != 2 {
			t.Fatalf("expected 2 rows, got %d", count)
		}
	})
}

func TestStoreDeleteOlderThan(t *testing.T) {
	store, db := setupStore(t)
	defer store.Close() //nolint:errcheck
	defer db.Close()    //nolint:errcheck

	oldPayload := AggregatorPayload{
		LogTimeEnd: 1800,
		Stats: []AggregatorEntry{{
			DetectedApplication:     10033,
			DetectedApplicationName: "old",
			DetectedProtocol:        196,
			DetectedProtocolName:    "old",
			IpProtocol:              6,
			IpVersion:               4,
			LocalBytes:              100,
			LocalIp:                 "10.0.0.1",
			LocalMac:                "XX:XX.XX:XX:XX:XX",
			LocalOrigin:             true,
			OtherBytes:              200,
			OtherIp:                 "10.0.0.2",
			OtherPort:               80,
			OtherType:               "remote",
		}},
	}

	newPayload := AggregatorPayload{
		LogTimeEnd: 36000,
		Stats: []AggregatorEntry{{
			DetectedApplication:     20001,
			DetectedApplicationName: "new",
			DetectedProtocol:        200,
			DetectedProtocolName:    "new",
			IpProtocol:              0,
			IpVersion:               4,
			LocalBytes:              50,
			LocalIp:                 "10.0.0.3",
			LocalMac:                "YY:YY.YY:YY:YY:YY",
			LocalOrigin:             false,
			OtherBytes:              75,
			OtherIp:                 "10.0.0.4",
			OtherPort:               80,
			OtherType:               "unknown",
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
	err := db.QueryRow("SELECT COUNT(*) FROM aggregator_stats").Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 row after delete, got %d", count)
	}

	var appName string
	err = db.QueryRow("SELECT detected_application_name FROM aggregator_stats").Scan(&appName)
	if err != nil {
		t.Fatal(err)
	}
	if appName != "new" {
		t.Fatalf("expected 'new' to remain, got %q", appName)
	}
}
