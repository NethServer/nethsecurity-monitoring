package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nethserver/nethsecurity-monitoring/flows"
)

func TestHandleFlows(t *testing.T) {
	t.Run("returns empty array when no flows", func(t *testing.T) {
		processor := flows.NewFlowProcessor()
		req := httptest.NewRequest(http.MethodGet, "/flows", nil)
		rec := httptest.NewRecorder()

		handler := handleFlows(processor)
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var result FlowsResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if result.Total != 0 {
			t.Errorf("expected total 0, got %d", result.Total)
		}

		if len(result.Flows) != 0 {
			t.Errorf("expected empty flows array, got %d items", len(result.Flows))
		}
	})

	t.Run("returns flows when present", func(t *testing.T) {
		processor := flows.NewFlowProcessor()

		// Add a test flow
		now := time.Now()
		testEvent := flows.FlowEvent{
			Type:      "flow",
			Interface: "lan",
			Internal:  true,
			Flow: flows.FlowStart{
				FlowBase: flows.FlowBase{
					Digest: "test-digest-123",
				},
				LocalIp:     "192.168.1.100",
				OtherIp:     "8.8.8.8",
				LocalPort:   12345,
				OtherPort:   443,
				FirstSeenAt: now.Unix(),
				LastSeenAt:  now.Unix(),
			},
		}
		processor.Process(testEvent)

		req := httptest.NewRequest(http.MethodGet, "/flows", nil)
		rec := httptest.NewRecorder()

		handler := handleFlows(processor)
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var result FlowsResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if result.Total != 1 {
			t.Errorf("expected total 1, got %d", result.Total)
		}

		if len(result.Flows) != 1 {
			t.Errorf("expected 1 flow, got %d", len(result.Flows))
		}
	})

	t.Run("returns multiple flows", func(t *testing.T) {
		processor := flows.NewFlowProcessor()

		now := time.Now()
		for i := 0; i < 5; i++ {
			testEvent := flows.FlowEvent{
				Type:      "flow",
				Interface: "lan",
				Internal:  true,
				Flow: flows.FlowStart{
					FlowBase: flows.FlowBase{
						Digest: "test-digest-" + string(rune(i)),
					},
					LocalIp:     "192.168.1.100",
					OtherIp:     "8.8.8.8",
					LocalPort:   12345 + i,
					OtherPort:   443,
					FirstSeenAt: now.Unix(),
					LastSeenAt:  now.Unix(),
				},
			}
			processor.Process(testEvent)
		}

		req := httptest.NewRequest(http.MethodGet, "/flows", nil)
		rec := httptest.NewRecorder()

		handler := handleFlows(processor)
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		var result FlowsResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if result.Total != 5 {
			t.Errorf("expected total 5, got %d", result.Total)
		}

		if len(result.Flows) != 5 {
			t.Errorf("expected 5 flows, got %d", len(result.Flows))
		}
	})

	t.Run("pagination with start parameter", func(t *testing.T) {
		processor := flows.NewFlowProcessor()

		now := time.Now()
		for i := 0; i < 10; i++ {
			testEvent := flows.FlowEvent{
				Type:      "flow",
				Interface: "lan",
				Internal:  true,
				Flow: flows.FlowStart{
					FlowBase: flows.FlowBase{
						Digest: "test-digest-" + fmt.Sprintf("%d", i),
					},
					LocalIp:     "192.168.1.100",
					OtherIp:     "8.8.8.8",
					LocalPort:   12345 + i,
					OtherPort:   443,
					FirstSeenAt: now.Unix(),
					LastSeenAt:  now.Unix(),
				},
			}
			processor.Process(testEvent)
		}

		req := httptest.NewRequest(http.MethodGet, "/flows?start=5", nil)
		rec := httptest.NewRecorder()

		handler := handleFlows(processor)
		handler.ServeHTTP(rec, req)

		var result FlowsResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if result.Total != 10 {
			t.Errorf("expected total 10, got %d", result.Total)
		}

		if len(result.Flows) != 5 {
			t.Errorf("expected 5 flows (from index 5 to 10), got %d", len(result.Flows))
		}
	})

	t.Run("pagination with start and end parameters", func(t *testing.T) {
		processor := flows.NewFlowProcessor()

		now := time.Now()
		for i := 0; i < 20; i++ {
			testEvent := flows.FlowEvent{
				Type:      "flow",
				Interface: "lan",
				Internal:  true,
				Flow: flows.FlowStart{
					FlowBase: flows.FlowBase{
						Digest: "test-digest-" + fmt.Sprintf("%d", i),
					},
					LocalIp:     "192.168.1.100",
					OtherIp:     "8.8.8.8",
					LocalPort:   12345 + i,
					OtherPort:   443,
					FirstSeenAt: now.Unix(),
					LastSeenAt:  now.Unix(),
				},
			}
			processor.Process(testEvent)
		}

		req := httptest.NewRequest(http.MethodGet, "/flows?start=5&end=15", nil)
		rec := httptest.NewRecorder()

		handler := handleFlows(processor)
		handler.ServeHTTP(rec, req)

		var result FlowsResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if result.Total != 20 {
			t.Errorf("expected total 20, got %d", result.Total)
		}

		if len(result.Flows) != 10 {
			t.Errorf("expected 10 flows (from index 5 to 15), got %d", len(result.Flows))
		}
	})

	t.Run("pagination handles out of bounds", func(t *testing.T) {
		processor := flows.NewFlowProcessor()

		now := time.Now()
		for i := 0; i < 5; i++ {
			testEvent := flows.FlowEvent{
				Type:      "flow",
				Interface: "lan",
				Internal:  true,
				Flow: flows.FlowStart{
					FlowBase: flows.FlowBase{
						Digest: "test-digest-" + string(rune(i)),
					},
					LocalIp:     "192.168.1.100",
					OtherIp:     "8.8.8.8",
					LocalPort:   12345 + i,
					OtherPort:   443,
					FirstSeenAt: now.Unix(),
					LastSeenAt:  now.Unix(),
				},
			}
			processor.Process(testEvent)
		}

		req := httptest.NewRequest(http.MethodGet, "/flows?start=10&end=20", nil)
		rec := httptest.NewRecorder()

		handler := handleFlows(processor)
		handler.ServeHTTP(rec, req)

		var result FlowsResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
			t.Fatalf("failed to unmarshal response: %v", err)
		}

		if result.Total != 5 {
			t.Errorf("expected total 5, got %d", result.Total)
		}

		// Start > total, so should return empty
		if len(result.Flows) != 0 {
			t.Errorf("expected 0 flows (start beyond bounds), got %d", len(result.Flows))
		}
	})
}
