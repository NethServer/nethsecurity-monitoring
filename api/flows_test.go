package api

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-playground/assert/v2"
	"github.com/gofiber/fiber/v3"
	"github.com/nethserver/nethsecurity-monitoring/flows"
)

type MockFlowAccessor struct {
	events map[string]flows.FlowEvent
}

func (m *MockFlowAccessor) GetEvents() map[string]flows.FlowEvent {
	return m.events
}

type MockFlowIngestor struct {
	processedEvents []flows.FlowEvent
}

func (m *MockFlowIngestor) Process(event flows.FlowEvent) {
	m.processedEvents = append(m.processedEvents, event)
}

func setupApi(t *testing.T, accessor *MockFlowAccessor, ingestor *MockFlowIngestor) *fiber.App {
	t.Helper()
	app := fiber.New()
	flowApi := NewFlowApi(accessor, ingestor)
	flowApi.Setup(app)
	return app
}

func TestFlows(t *testing.T) {
	t.Run("flows endpoint is good", func(t *testing.T) {
		accessor := &MockFlowAccessor{
			events: map[string]flows.FlowEvent{
				"f-001": {
					Type: flows.FlowTypeDpiComplete,
					Flow: flows.FlowComplete{
						FlowBase: flows.FlowBase{
							Digest: "f-001",
						},
						LocalOrigin: false,
					},
				},
				"f-002": {
					Type: flows.FlowTypeDpiComplete,
					Flow: flows.FlowComplete{
						FlowBase: flows.FlowBase{
							Digest: "f-002",
						},
						LocalOrigin: true,
						Stats:       flows.Stats{LocalRate: 3000, OtherRate: 200},
					},
				},
				"f-003": {
					Type: flows.FlowTypeDpiComplete,
					Flow: flows.FlowComplete{
						FlowBase:    flows.FlowBase{Digest: "f-003"},
						LocalOrigin: true,
					},
				},
				"f-004": {
					Type: flows.FlowTypeDpiComplete,
					Flow: flows.FlowComplete{
						FlowBase:    flows.FlowBase{Digest: "f-004"},
						LocalOrigin: false,
						Stats:       flows.Stats{TotalBytes: 5000, LocalRate: 10, OtherRate: 0.0},
					},
				},
			},
		}
		ingestor := &MockFlowIngestor{}
		app := setupApi(t, accessor, ingestor)
		req := httptest.NewRequest(http.MethodGet, "/flows", nil)
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
		expected := map[string]bool{"f-001": true, "f-002": true, "f-003": true, "f-004": true}
		for _, ev := range body.Data {
			var digest string
			switch ev.Flow.(type) {
			case flows.FlowComplete:
				digest = ev.Flow.(flows.FlowComplete).Digest
			case flows.FlowStats:
				digest = ev.Flow.(flows.FlowStats).Digest
			case flows.FlowPurge:
				digest = ev.Flow.(flows.FlowPurge).Digest
			default:
				t.Fatalf("unexpected flow type: %T", ev.Flow)
			}
			if !expected[digest] {
				t.Errorf("unexpected or duplicate digest in response: %s", digest)
			}
			delete(expected, digest)
		}
		if len(expected) > 0 {
			t.Errorf("missing digests in response: %v", expected)
		}
	})
}

func TestFlowsPost(t *testing.T) {
	tests := []struct {
		name             string
		payload          string
		expectedStatus   int
		shouldBeIngested bool
		description      string
	}{
		{
			name:             "valid flow_dpi_complete event",
			payload:          `{"type":"flow_dpi_complete","flow":{"digest":"test-001","local_origin":true,"local_ip":"10.0.0.1","local_port":8080,"other_ip":"1.2.3.4","other_port":80,"other_type":"remote","ip_protocol":6,"ip_version":4,"detected_protocol":7,"detected_protocol_name":"HTTP","detected_application":119,"detected_application_name":"HTTP","conntrack":{"id":0,"mark":0,"reply_dst_ip":"","reply_dst_port":0,"reply_src_ip":"","reply_src_port":0}}}`,
			expectedStatus:   200,
			shouldBeIngested: true,
			description:      "should accept and ingest valid flow_dpi_complete",
		},
		{
			name:             "valid flow_purge event",
			payload:          `{"type":"flow_purge","flow":{"digest":"test-002","last_seen_at":1234567890}}`,
			expectedStatus:   200,
			shouldBeIngested: true,
			description:      "should accept and ingest valid flow_purge",
		},
		{
			name:             "unsupported flow type",
			payload:          `{"type":"flow_unknown","flow":{"digest":"test-003"}}`,
			expectedStatus:   200,
			shouldBeIngested: false,
			description:      "should silently ignore unsupported flow type with 200",
		},
		{
			name:             "malformed json",
			payload:          `{"type":"flow_dpi_complete","flow":not valid json}`,
			expectedStatus:   400,
			shouldBeIngested: false,
			description:      "should reject malformed JSON with 400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			accessor := &MockFlowAccessor{events: make(map[string]flows.FlowEvent)}
			ingestor := &MockFlowIngestor{}
			app := setupApi(t, accessor, ingestor)

			req := httptest.NewRequest(http.MethodPost, "/flows", bytes.NewBufferString(tt.payload))
			req.Header.Set("Content-Type", "application/json")

			res, err := app.Test(req)
			if err != nil {
				t.Fatal(err)
			}

			assert.Equal(t, tt.expectedStatus, res.StatusCode)

			if tt.shouldBeIngested {
				assert.Equal(t, 1, len(ingestor.processedEvents))
			} else {
				assert.Equal(t, 0, len(ingestor.processedEvents))
			}
		})
	}
}
