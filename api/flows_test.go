package api

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/go-playground/assert/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/nethserver/nethsecurity-monitoring/flows"
)

type MockFlowAccessor struct {
	events map[string]flows.FlowEvent
}

func (m *MockFlowAccessor) GetEvents() map[string]flows.FlowEvent {
	return m.events
}

func setupApi(t *testing.T, mock *MockFlowAccessor) *fiber.App {
	t.Helper()
	app := fiber.New()
	flowApi := NewFlowApi(mock)
	flowApi.Setup(app)
	return app
}

func TestFlows(t *testing.T) {
	t.Run("flows endpoint is good", func(t *testing.T) {
		mock := &MockFlowAccessor{
			events: map[string]flows.FlowEvent{
				"f-001": {
					Type: flows.FlowTypeBegin,
					Flow: flows.FlowStart{
						FlowBase: flows.FlowBase{
							Digest: "f-001",
						},
						LocalOrigin: false,
					},
				},
				"f-002": {
					Type: flows.FlowTypeDpiComplete,
					Flow: flows.FlowComplete{
						FlowStart: flows.FlowStart{
							FlowBase: flows.FlowBase{
								Digest: "f-002",
							},
							LocalOrigin: true,
						},
						Stats: flows.Stats{LocalRate: 3000, OtherRate: 200},
					},
				},
				"f-003": {
					Type: flows.FlowTypeBegin,
					Flow: flows.FlowStart{
						FlowBase:    flows.FlowBase{Digest: "f-003"},
						LocalOrigin: true,
					},
				},
				"f-004": {
					Type: flows.FlowTypeDpiComplete,
					Flow: flows.FlowComplete{
						FlowStart: flows.FlowStart{
							FlowBase:    flows.FlowBase{Digest: "f-004"},
							LocalOrigin: false,
						},
						Stats: flows.Stats{TotalBytes: 5000, LocalRate: 10, OtherRate: 0.0},
					},
				},
			},
		}
		app := setupApi(t, mock)
		req, _ := http.NewRequest("GET", "/flows", nil)
		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, 200, res.StatusCode)
		var body FlowsResponse
		err = json.NewDecoder(res.Body).Decode(&body)
		if err != nil && err != io.EOF {
			t.Fatal(err)
		}
		assert.Equal(t, 4, len(body.Data))
		assert.Equal(t, 4, body.Total)
		assert.Equal(t, 10, body.PerPage)
		assert.Equal(t, 1, body.CurrentPage)
		assert.Equal(t, 1, body.LastPage)
		order := []string{"f-004", "f-002", "f-001", "f-003"}
		for i, ev := range body.Data {
			var digest string
			switch ev.Flow.(type) {
			case flows.FlowStart:
				digest = ev.Flow.(flows.FlowStart).Digest
			case flows.FlowComplete:
				digest = ev.Flow.(flows.FlowComplete).Digest
			case flows.FlowStats:
				digest = ev.Flow.(flows.FlowStats).Digest
			case flows.FlowPurge:
				digest = ev.Flow.(flows.FlowPurge).Digest
			default:
				t.Fatalf("unexpected flow type: %T", ev.Flow)
			}
			assert.Equal(t, order[i], digest)
		}
	})

	t.Run("flows can be limited", func(t *testing.T) {
		mock := &MockFlowAccessor{
			events: map[string]flows.FlowEvent{
				"f-001": {
					Type: flows.FlowTypeBegin,
					Flow: flows.FlowStart{
						FlowBase: flows.FlowBase{
							Digest: "f-001",
						},
					},
				},
				"f-002": {
					Type: flows.FlowTypeDpiComplete,
					Flow: flows.FlowComplete{
						FlowStart: flows.FlowStart{
							FlowBase: flows.FlowBase{
								Digest: "f-002",
							},
						},
						Stats: flows.Stats{LocalRate: 3000, OtherRate: 200},
					},
				},
			},
		}
		app := setupApi(t, mock)
		req, _ := http.NewRequest("GET", "/flows?per_page=1", nil)
		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, 200, res.StatusCode)
		var body FlowsResponse
		err = json.NewDecoder(res.Body).Decode(&body)
		if err != nil && err != io.EOF {
			t.Fatal(err)
		}
		assert.Equal(t, 1, len(body.Data))
		assert.Equal(t, 2, body.Total)
		assert.Equal(t, 1, body.PerPage)
		assert.Equal(t, 1, body.CurrentPage)
		assert.Equal(t, 2, body.LastPage)
	})

	t.Run("defaults are applied when params are absent", func(t *testing.T) {
		mock := &MockFlowAccessor{events: map[string]flows.FlowEvent{}}
		app := setupApi(t, mock)
		req, _ := http.NewRequest("GET", "/flows", nil)
		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, 200, res.StatusCode)
		var body FlowsResponse
		err = json.NewDecoder(res.Body).Decode(&body)
		if err != nil && err != io.EOF {
			t.Fatal(err)
		}
		assert.Equal(t, 10, body.PerPage)
		assert.Equal(t, 1, body.CurrentPage)
	})

	for _, query := range []string{"page=0", "per_page=200", "sort_by=invalid"} {
		t.Run("invalid "+query+" returns 400", func(t *testing.T) {
			app := setupApi(t, &MockFlowAccessor{events: map[string]flows.FlowEvent{}})
			req, _ := http.NewRequest("GET", "/flows?"+query, nil)
			res, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, 400, res.StatusCode)
		})
	}
}
