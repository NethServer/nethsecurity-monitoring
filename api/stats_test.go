package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/assert/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/nethserver/nethsecurity-monitoring/stats"
)

type mockSaver struct {
	payloads []stats.AggregatorPayload
	err      error
}

func (m *mockSaver) Save(_ context.Context, payload stats.AggregatorPayload) error {
	m.payloads = append(m.payloads, payload)
	return m.err
}

func setupStatsApi(t *testing.T, saver stats.Saver) *fiber.App {
	t.Helper()

	app := fiber.New()
	statsApi := NewStatsApi(saver)
	statsApi.Setup(app)

	return app
}

func TestStats(t *testing.T) {
	t.Run("stats endpoint accepts sample payload", func(t *testing.T) {
		saver := &mockSaver{}
		app := setupStatsApi(t, saver)

		sample := stats.AggregatorPayload{
			LogTimeEnd: 3661,
			Stats: []stats.AggregatorEntry{
				{
					DetectedApplication:     10033,
					DetectedApplicationName: "netify.netify",
					DetectedProtocol:        196,
					DetectedProtocolName:    "HTTP/S",
					LocalBytes:              30,
					LocalIp:                 "10.0.0.1",
					LocalOrigin:             true,
					OtherBytes:              40,
					OtherIp:                 "10.0.0.2",
				},
			},
		}

		payload, err := json.Marshal(sample)
		if err != nil {
			t.Fatal(err)
		}

		req := httptest.NewRequest(http.MethodPost, "/stats", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")

		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, 200, res.StatusCode)
		assert.Equal(t, 1, len(saver.payloads))
		assert.Equal(t, sample.LogTimeEnd, saver.payloads[0].LogTimeEnd)
		assert.Equal(t, len(sample.Stats), len(saver.payloads[0].Stats))
	})

	t.Run("stats endpoint rejects malformed payload", func(t *testing.T) {
		app := setupStatsApi(t, &mockSaver{})

		req := httptest.NewRequest(
			http.MethodPost,
			"/stats",
			bytes.NewBufferString(`{"log_time_end":"bad"}`),
		)
		req.Header.Set("Content-Type", "application/json")

		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, 400, res.StatusCode)
	})

	t.Run("stats endpoint stores same timestamps twice", func(t *testing.T) {
		saver := &mockSaver{}
		app := setupStatsApi(t, saver)

		payload := stats.AggregatorPayload{
			LogTimeEnd: 3661,
			Stats: []stats.AggregatorEntry{
				{
					DetectedApplication:     10033,
					DetectedApplicationName: "netify.netify",
					DetectedProtocol:        196,
					DetectedProtocolName:    "HTTP/S",
					LocalBytes:              3,
					LocalIp:                 "1.1.1.1",
					LocalOrigin:             false,
					OtherBytes:              4,
					OtherIp:                 "2.2.2.2",
				},
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
		assert.Equal(t, 2, len(saver.payloads))
	})

	t.Run("stats endpoint propagates save errors", func(t *testing.T) {
		app := setupStatsApi(t, &mockSaver{err: errors.New("boom")})

		req := httptest.NewRequest(
			http.MethodPost,
			"/stats",
			bytes.NewReader([]byte(`{"log_time_end":3661,"stats":[]}`)),
		)
		req.Header.Set("Content-Type", "application/json")

		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, 500, res.StatusCode)
	})
}
