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
	t.Run("stores raw stats entries with batch tracking", func(t *testing.T) {
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

		upsertPayload := AggregatorPayload{
			LogTimeEnd: 3700,
			Stats: []AggregatorEntry{
				{
					DetectedApplication:     10033,
					DetectedApplicationName: "netify.netify",
					DetectedProtocol:        196,
					DetectedProtocolName:    "HTTP/S",
					IpProtocol:              6,
					IpVersion:               4,
					LocalBytes:              25,
					LocalIp:                 "10.0.0.1",
					LocalMac:                "XX:XX.XX:XX:XX:XX",
					LocalOrigin:             true,
					OtherBytes:              75,
					OtherIp:                 "10.0.0.2",
					OtherPort:               80,
					OtherType:               "remote",
				},
			},
		}

		if err := store.Save(context.Background(), payload); err != nil {
			t.Fatal(err)
		}
		if err := store.Save(context.Background(), upsertPayload); err != nil {
			t.Fatal(err)
		}

		// Verify batches were created
		var batchCount int
		err := db.QueryRow("SELECT COUNT(*) FROM aggregator_batches").Scan(&batchCount)
		if err != nil {
			t.Fatal(err)
		}
		if batchCount != 2 {
			t.Fatalf("expected 2 batches, got %d", batchCount)
		}

		// Verify stats entries were created
		var statsCount int
		err = db.QueryRow("SELECT COUNT(*) FROM aggregator_stats").Scan(&statsCount)
		if err != nil {
			t.Fatal(err)
		}
		if statsCount != 3 {
			t.Fatalf("expected 3 stats entries, got %d", statsCount)
		}

		// Verify first entry data
		var appName string
		var localBytes, otherBytes int64
		err = db.QueryRow(
			`SELECT detected_application_name, local_bytes, other_bytes
			 FROM aggregator_stats
			 WHERE detected_application = ? AND detected_protocol = ?
			 LIMIT 1`,
			10033,
			196,
		).Scan(&appName, &localBytes, &otherBytes)
		if err != nil {
			t.Fatal(err)
		}
		if appName != "netify.netify" {
			t.Fatalf("expected 'netify.netify', got %q", appName)
		}
		if localBytes != 100 {
			t.Fatalf("expected local_bytes 100, got %d", localBytes)
		}
		if otherBytes != 200 {
			t.Fatalf("expected other_bytes 200, got %d", otherBytes)
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

	// Verify batch was deleted
	var batchCount int
	err := db.QueryRow("SELECT COUNT(*) FROM aggregator_batches").Scan(&batchCount)
	if err != nil {
		t.Fatal(err)
	}
	if batchCount != 1 {
		t.Fatalf("expected 1 batch after delete, got %d", batchCount)
	}

	// Verify stats entries were cascaded deleted
	var statsCount int
	err = db.QueryRow("SELECT COUNT(*) FROM aggregator_stats").Scan(&statsCount)
	if err != nil {
		t.Fatal(err)
	}
	if statsCount != 1 {
		t.Fatalf("expected 1 stats entry after delete, got %d", statsCount)
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

func TestQueryableHours(t *testing.T) {
	t.Run("returns distinct hour epochs in ascending order", func(t *testing.T) {
		store, _ := setupStore(t)
		defer store.Close() //nolint:errcheck

		// Insert batches at different times within different hours
		// Hour 0: 3600 seconds
		payload1 := AggregatorPayload{
			LogTimeEnd: 1800, // within hour 0 (0 * 3600)
			Stats: []AggregatorEntry{{
				DetectedApplication:     10033,
				DetectedApplicationName: "app1",
				DetectedProtocol:        196,
				DetectedProtocolName:    "proto1",
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

		// Hour 0 again: should not add new hour
		payload2 := AggregatorPayload{
			LogTimeEnd: 3000, // still within hour 0
			Stats: []AggregatorEntry{{
				DetectedApplication:     20001,
				DetectedApplicationName: "app2",
				DetectedProtocol:        200,
				DetectedProtocolName:    "proto2",
				IpProtocol:              6,
				IpVersion:               4,
				LocalBytes:              50,
				LocalIp:                 "10.0.0.3",
				LocalMac:                "YY:YY.YY:YY:YY:YY",
				LocalOrigin:             true,
				OtherBytes:              75,
				OtherIp:                 "10.0.0.4",
				OtherPort:               80,
				OtherType:               "remote",
			}},
		}

		// Hour 1: 7200 seconds
		payload3 := AggregatorPayload{
			LogTimeEnd: 5400, // within hour 1 (1 * 3600)
			Stats: []AggregatorEntry{{
				DetectedApplication:     30001,
				DetectedApplicationName: "app3",
				DetectedProtocol:        300,
				DetectedProtocolName:    "proto3",
				IpProtocol:              6,
				IpVersion:               4,
				LocalBytes:              150,
				LocalIp:                 "10.0.0.5",
				LocalMac:                "ZZ:ZZ.ZZ:ZZ:ZZ:ZZ",
				LocalOrigin:             true,
				OtherBytes:              250,
				OtherIp:                 "10.0.0.6",
				OtherPort:               80,
				OtherType:               "remote",
			}},
		}

		if err := store.Save(context.Background(), payload1); err != nil {
			t.Fatal(err)
		}
		if err := store.Save(context.Background(), payload2); err != nil {
			t.Fatal(err)
		}
		if err := store.Save(context.Background(), payload3); err != nil {
			t.Fatal(err)
		}

		hours, err := store.QueryableHours(context.Background(), 100)
		if err != nil {
			t.Fatal(err)
		}

		if len(hours) != 2 {
			t.Fatalf("expected 2 distinct hours, got %d", len(hours))
		}
		if hours[0] != 0 {
			t.Fatalf("expected first hour to be 0, got %d", hours[0])
		}
		if hours[1] != 3600 {
			t.Fatalf("expected second hour to be 3600, got %d", hours[1])
		}
	})

	t.Run("returns empty slice for empty database", func(t *testing.T) {
		store, _ := setupStore(t)
		defer store.Close() //nolint:errcheck

		hours, err := store.QueryableHours(context.Background(), 100)
		if err != nil {
			t.Fatal(err)
		}

		if len(hours) != 0 {
			t.Fatalf("expected 0 hours for empty database, got %d", len(hours))
		}
	})
}

func TestQueryHour(t *testing.T) {
	t.Run("aggregates bytes by local_ip, protocol, application, and other_ip", func(t *testing.T) {
		store, _ := setupStore(t)
		defer store.Close() //nolint:errcheck

		// Two entries with same local_ip, protocol, app, other_ip -> should be summed
		// One entry with different other_ip -> separate row
		payload := AggregatorPayload{
			LogTimeEnd: 1800, // hour 0
			Stats: []AggregatorEntry{
				{
					DetectedApplication:     10033,
					DetectedApplicationName: "app1",
					DetectedProtocol:        196,
					DetectedProtocolName:    "proto1",
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
					DetectedApplication:     10033,
					DetectedApplicationName: "app1",
					DetectedProtocol:        196,
					DetectedProtocolName:    "proto1",
					IpProtocol:              6,
					IpVersion:               4,
					LocalBytes:              50,
					LocalIp:                 "10.0.0.1",
					LocalMac:                "XX:XX.XX:XX:XX:XX",
					LocalOrigin:             true,
					OtherBytes:              150,
					OtherIp:                 "10.0.0.2",
					OtherPort:               80,
					OtherType:               "remote",
				},
				{
					DetectedApplication:     10033,
					DetectedApplicationName: "app1",
					DetectedProtocol:        196,
					DetectedProtocolName:    "proto1",
					IpProtocol:              6,
					IpVersion:               4,
					LocalBytes:              75,
					LocalIp:                 "10.0.0.1",
					LocalMac:                "XX:XX.XX:XX:XX:XX",
					LocalOrigin:             true,
					OtherBytes:              125,
					OtherIp:                 "10.0.0.3", // different IP
					OtherPort:               443,
					OtherType:               "remote",
				},
			},
		}

		if err := store.Save(context.Background(), payload); err != nil {
			t.Fatal(err)
		}

		rows, err := store.QueryHour(context.Background(), 0, 3600)
		if err != nil {
			t.Fatal(err)
		}

		if len(rows) != 2 {
			t.Fatalf("expected 2 rows (different other_ip), got %d", len(rows))
		}

		// First row: 10.0.0.2, sum = (100+200)+(50+150) = 500
		if rows[0].LocalIP != "10.0.0.1" {
			t.Fatalf("expected local_ip 10.0.0.1, got %s", rows[0].LocalIP)
		}
		if rows[0].Host != "10.0.0.2" {
			t.Fatalf("expected host 10.0.0.2, got %s", rows[0].Host)
		}
		if rows[0].TotalBytes != 500 {
			t.Fatalf("expected 500 bytes for first row, got %d", rows[0].TotalBytes)
		}

		// Second row: 10.0.0.3, sum = 75+125 = 200
		if rows[1].Host != "10.0.0.3" {
			t.Fatalf("expected host 10.0.0.3, got %s", rows[1].Host)
		}
		if rows[1].TotalBytes != 200 {
			t.Fatalf("expected 200 bytes for second row, got %d", rows[1].TotalBytes)
		}
	})

	t.Run("respects hour time range", func(t *testing.T) {
		store, _ := setupStore(t)
		defer store.Close() //nolint:errcheck

		// Insert at hour 0 and hour 1
		payload1 := AggregatorPayload{
			LogTimeEnd: 1800, // hour 0
			Stats: []AggregatorEntry{{
				DetectedApplication:     10033,
				DetectedApplicationName: "app1",
				DetectedProtocol:        196,
				DetectedProtocolName:    "proto1",
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

		payload2 := AggregatorPayload{
			LogTimeEnd: 5400, // hour 1
			Stats: []AggregatorEntry{{
				DetectedApplication:     20001,
				DetectedApplicationName: "app2",
				DetectedProtocol:        200,
				DetectedProtocolName:    "proto2",
				IpProtocol:              6,
				IpVersion:               4,
				LocalBytes:              50,
				LocalIp:                 "10.0.0.3",
				LocalMac:                "YY:YY.YY:YY:YY:YY",
				LocalOrigin:             true,
				OtherBytes:              75,
				OtherIp:                 "10.0.0.4",
				OtherPort:               80,
				OtherType:               "remote",
			}},
		}

		if err := store.Save(context.Background(), payload1); err != nil {
			t.Fatal(err)
		}
		if err := store.Save(context.Background(), payload2); err != nil {
			t.Fatal(err)
		}

		// Query only hour 1 (3600 to 7200)
		rows, err := store.QueryHour(context.Background(), 3600, 7200)
		if err != nil {
			t.Fatal(err)
		}

		if len(rows) != 1 {
			t.Fatalf("expected 1 row for hour 1, got %d", len(rows))
		}
		if rows[0].ApplicationName != "app2" {
			t.Fatalf("expected app2, got %s", rows[0].ApplicationName)
		}
	})

	t.Run("returns empty slice for hour with no data", func(t *testing.T) {
		store, _ := setupStore(t)
		defer store.Close() //nolint:errcheck

		rows, err := store.QueryHour(context.Background(), 0, 3600)
		if err != nil {
			t.Fatal(err)
		}

		if len(rows) != 0 {
			t.Fatalf("expected 0 rows for empty hour, got %d", len(rows))
		}
	})
}
