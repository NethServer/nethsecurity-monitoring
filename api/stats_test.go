package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-playground/assert/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/nethserver/nethsecurity-monitoring/stats"
	_ "modernc.org/sqlite"
)

func setupStatsApi(t *testing.T) (*fiber.App, *sql.DB) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "stats.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}

	store := stats.NewStore(db)
	if err := store.Init(t.Context()); err != nil {
		t.Fatal(err)
	}

	app := fiber.New()
	statsApi := NewStatsApi(stats.NewReceiver(store))
	statsApi.Setup(app)

	return app, db
}

func TestStats(t *testing.T) {
	t.Run("stats endpoint accepts sample payload", func(t *testing.T) {
		app, db := setupStatsApi(t)
		defer db.Close() //nolint:errcheck

		payloadPath := filepath.Join("..", "logs-testing.json")
		payload, err := os.ReadFile(payloadPath)
		if err != nil {
			t.Fatal(err)
		}

		var sample stats.Payload
		if err := json.Unmarshal(payload, &sample); err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodPost, "/stats", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")

		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, 200, res.StatusCode)

		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM stats").Scan(&count); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, len(sample.Stats), count)

		if err := db.QueryRow("SELECT COUNT(*) FROM stats_timestamps").Scan(&count); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, 1, count)
	})

	t.Run("stats endpoint rejects malformed payload", func(t *testing.T) {
		app, db := setupStatsApi(t)
		defer db.Close() //nolint:errcheck

		req := httptest.NewRequest(http.MethodPost, "/stats", bytes.NewBufferString(`{"log_time_start":"bad"}`))
		req.Header.Set("Content-Type", "application/json")

		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, 400, res.StatusCode)
	})

	t.Run("stats endpoint stores same timestamps twice", func(t *testing.T) {
		app, db := setupStatsApi(t)
		defer db.Close() //nolint:errcheck

		payload := stats.Payload{
			LogTimeStart: 1,
			LogTimeEnd:   2,
			Stats: []stats.Statistic{
				{DetectedApplication: 1, DetectedApplicationName: "a", DetectedProtocol: 2, DetectedProtocolName: "b", Internal: true, LocalBytes: 3, LocalIp: "1.1.1.1", LocalOrigin: false, OtherBytes: 4, OtherIp: "2.2.2.2", OtherType: "remote"},
			},
		}

		body, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}

		for range 2 {
			req := httptest.NewRequest(http.MethodPost, "/stats", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			res, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, 200, res.StatusCode)
		}

		var count int
		if err := db.QueryRow("SELECT COUNT(*) FROM stats").Scan(&count); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, 2, count)

		if err := db.QueryRow("SELECT COUNT(*) FROM stats_timestamps").Scan(&count); err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, 1, count)
	})
}
