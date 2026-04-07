package stats

import (
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

	store := NewStore(db)
	if err := store.Init(t.Context()); err != nil {
		t.Fatal(err)
	}

	return store, db
}

func TestStoreSave(t *testing.T) {
	t.Run("stores one row per stat", func(t *testing.T) {
		store, db := setupStore(t)
		defer db.Close() //nolint:errcheck

		payload := Payload{
			LogTimeStart: 1,
			LogTimeEnd:   2,
			Stats: []Statistic{
				{DetectedApplication: 10, DetectedApplicationName: "app", DetectedProtocol: 20, DetectedProtocolName: "proto", Internal: true, LocalBytes: 30, LocalIp: "10.0.0.1", LocalOrigin: true, OtherBytes: 40, OtherIp: "10.0.0.2", OtherType: "remote"},
				{DetectedApplication: 11, DetectedApplicationName: "app2", DetectedProtocol: 21, DetectedProtocolName: "proto2", Internal: false, LocalBytes: 31, LocalIp: "10.0.0.3", LocalOrigin: false, OtherBytes: 41, OtherIp: "10.0.0.4", OtherType: "local"},
			},
		}

		if err := store.Save(t.Context(), payload); err != nil {
			t.Fatal(err)
		}

		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM stats").Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 2 {
			t.Fatalf("expected 2 rows, got %d", count)
		}

		if err := db.QueryRow("SELECT COUNT(*) FROM stats_timestamps").Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("expected 1 timestamp row, got %d", count)
		}
	})

	t.Run("reuses repeated timestamps", func(t *testing.T) {
		store, db := setupStore(t)
		defer db.Close() //nolint:errcheck

		payload := Payload{
			LogTimeStart: 5,
			LogTimeEnd:   6,
			Stats:        []Statistic{{DetectedApplication: 1, DetectedApplicationName: "a", DetectedProtocol: 2, DetectedProtocolName: "b", Internal: true, LocalBytes: 3, LocalIp: "1.1.1.1", LocalOrigin: false, OtherBytes: 4, OtherIp: "2.2.2.2", OtherType: "remote"}},
		}

		if err := store.Save(t.Context(), payload); err != nil {
			t.Fatal(err)
		}
		if err := store.Save(t.Context(), payload); err != nil {
			t.Fatal(err)
		}

		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM stats").Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 2 {
			t.Fatalf("expected 2 rows, got %d", count)
		}

		if err := db.QueryRow("SELECT COUNT(*) FROM stats_timestamps").Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("expected 1 timestamp row, got %d", count)
		}
	})

	t.Run("cascades stats when timestamp is deleted", func(t *testing.T) {
		store, db := setupStore(t)
		defer db.Close() //nolint:errcheck

		payload := Payload{
			LogTimeStart: 9,
			LogTimeEnd:   10,
			Stats:        []Statistic{{DetectedApplication: 1, DetectedApplicationName: "a", DetectedProtocol: 2, DetectedProtocolName: "b", Internal: true, LocalBytes: 3, LocalIp: "1.1.1.1", LocalOrigin: false, OtherBytes: 4, OtherIp: "2.2.2.2", OtherType: "remote"}},
		}

		if err := store.Save(t.Context(), payload); err != nil {
			t.Fatal(err)
		}

		if _, err := db.Exec("DELETE FROM stats_timestamps WHERE log_time_start = ? AND log_time_end = ?", payload.LogTimeStart, payload.LogTimeEnd); err != nil {
			t.Fatal(err)
		}

		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM stats").Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("expected 0 rows after cascade, got %d", count)
		}
	})
}
