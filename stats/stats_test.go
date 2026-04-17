package stats

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func setupStore(t *testing.T) (*Store, *sql.DB) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "stats.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}

	store, err := Open(context.Background(), dbPath, NewReverseDNSCache())
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

func TestReverseDNSCache(t *testing.T) {
	t.Run("returns cached name and increments requests", func(t *testing.T) {
		cache := NewReverseDNSCacheWithResolver(func(context.Context, string) ([]string, error) {
			return []string{"example.local."}, nil
		})

		if got := cache.Resolve(context.Background(), "10.0.0.1"); got != "example.local" {
			t.Fatalf("expected resolved name, got %q", got)
		}
		if got := cache.Resolve(context.Background(), "10.0.0.1"); got != "example.local" {
			t.Fatalf("expected cached name, got %q", got)
		}

		cache.mu.Lock()
		defer cache.mu.Unlock()
		if cache.entries["10.0.0.1"].requests != 2 {
			t.Fatalf("expected 2 requests, got %d", cache.entries["10.0.0.1"].requests)
		}
	})

	t.Run("falls back to ip when lookup fails", func(t *testing.T) {
		cache := NewReverseDNSCacheWithResolver(func(context.Context, string) ([]string, error) {
			return nil, errors.New("boom")
		})

		if got := cache.Resolve(context.Background(), "10.0.0.2"); got != "10.0.0.2" {
			t.Fatalf("expected ip fallback, got %q", got)
		}
	})

	t.Run("prunes cold and expired entries", func(t *testing.T) {
		cache := NewReverseDNSCacheWithResolver(func(context.Context, string) ([]string, error) {
			return []string{"example.local."}, nil
		})
		cache.maxRecords = 2

		cache.mu.Lock()
		cache.entries["10.0.0.1"] = &reverseDNSCacheEntry{
			ip:       "10.0.0.1",
			name:     "a",
			requests: 10,
			addedAt:  time.Now().Add(-11 * time.Minute),
		}
		cache.entries["10.0.0.2"] = &reverseDNSCacheEntry{
			ip:       "10.0.0.2",
			name:     "b",
			requests: 1,
			addedAt:  time.Now(),
		}
		cache.entries["10.0.0.3"] = &reverseDNSCacheEntry{
			ip:       "10.0.0.3",
			name:     "c",
			requests: 2,
			addedAt:  time.Now(),
		}
		cache.mu.Unlock()

		cache.Prune(time.Now())

		cache.mu.Lock()
		defer cache.mu.Unlock()
		if _, ok := cache.entries["10.0.0.1"]; ok {
			t.Fatal("expected expired entry to be removed")
		}
		if len(cache.entries) != 2 {
			t.Fatalf("expected 2 entries after prune, got %d", len(cache.entries))
		}
	})
}

func TestStoreSaveResolvesNames(t *testing.T) {
	store, db := setupStore(t)
	defer store.Close() //nolint:errcheck
	defer db.Close()    //nolint:errcheck

	store.cache = NewReverseDNSCacheWithResolver(
		func(_ context.Context, ip string) ([]string, error) {
			switch ip {
			case "10.0.0.1":
				return []string{"local.example."}, nil
			case "10.0.0.2":
				return nil, errors.New("no ptr")
			default:
				return nil, errors.New("unexpected ip")
			}
		},
	)

	payload := Payload{
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

	if err := store.Save(context.Background(), payload); err != nil {
		t.Fatal(err)
	}

	var localName, otherName string
	err := db.QueryRow("SELECT local_name, other_name FROM hourly_traffic").
		Scan(&localName, &otherName)
	if err != nil {
		t.Fatal(err)
	}
	if localName != "local.example" {
		t.Fatalf("expected resolved local name, got %q", localName)
	}
	if otherName != "10.0.0.2" {
		t.Fatalf("expected ip fallback for other name, got %q", otherName)
	}
}
