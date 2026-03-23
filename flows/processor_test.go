package flows

import (
	"math/rand/v2"
	"strings"
	"sync"
	"testing"
	"time"
)

func randomDigest(t *testing.T) string {
	t.Helper()
	const length = 16
	const hexChars = "0123456789abcdef"
	var builder strings.Builder
	builder.Grow(length)

	for i := 0; i < length; i++ {
		builder.WriteByte(hexChars[rand.IntN(16)])
	}

	return builder.String()
}

func createFlowBase(t *testing.T) FlowBase {
	t.Helper()
	return FlowBase{
		Digest: randomDigest(t),
	}
}

// createFlowCompleteEvent creates a FlowComplete event with a unique digest.
func createFlowCompleteEvent(t *testing.T) FlowEvent {
	t.Helper()
	return FlowEvent{
		Type: FlowTypeDpiComplete,
		Flow: FlowComplete{
			FlowBase: createFlowBase(t),
		},
	}
}

func createFlowStatsEvent(t *testing.T) FlowEvent {
	t.Helper()
	minRate := 1000.0
	maxRate := 10000.0
	return FlowEvent{
		Type: FlowTypeStats,
		Flow: FlowStats{
			FlowBase: createFlowBase(t),
			Stats: Stats{
				LocalBytes:   rand.Int64N(1000),
				LocalPackets: rand.IntN(1000),
				LocalRate:    minRate + rand.Float64()*(maxRate-minRate),
				OtherBytes:   rand.Int64N(1000),
				OtherPackets: rand.IntN(1000),
				OtherRate:    minRate + rand.Float64()*(maxRate-minRate),
				TotalBytes:   rand.Int64N(1000),
				TotalPackets: rand.IntN(1000),
			},
			LastSeenAt: rand.Int64N(time.Now().Unix()),
		},
	}
}

func TestFlowsProcessor(t *testing.T) {
	t.Run("runs safely concurrently", func(t *testing.T) {
		wantedCount := 1000
		flowProcessor := NewFlowProcessor()

		var wg sync.WaitGroup
		wg.Add(wantedCount)

		for i := 0; i < wantedCount; i++ {
			go func() {
				defer wg.Done()
				flowProcessor.Process(createFlowCompleteEvent(t))
			}()
		}

		wg.Wait()

		events := flowProcessor.GetEvents()
		if len(events) != wantedCount {
			t.Errorf("Expected %d events, got %d", wantedCount, len(events))
		}
	})

	t.Run("save a flow complete correctly", func(t *testing.T) {
		flow := createFlowCompleteEvent(t)
		flowProcessor := NewFlowProcessor()
		flowProcessor.Process(flow)
		events := flowProcessor.GetEvents()
		if len(events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(events))
		}
		if _, ok := flow.Flow.(FlowComplete); !ok {
			t.Errorf("Expected flow to be of type FlowComplete")
		}
		if _, ok := events[flow.Flow.(FlowComplete).Digest]; !ok {
			t.Errorf("Expected event with digest %s to be present", flow.Flow.(FlowComplete).Digest)
		}
	})

	t.Run("handles flows correctly", func(t *testing.T) {
		flowComplete1 := createFlowCompleteEvent(t)
		flowComplete2 := createFlowCompleteEvent(t)
		flowPurge := FlowEvent{
			Type: FlowTypePurge,
			Flow: FlowPurge{
				FlowBase: flowComplete1.Flow.(FlowComplete).FlowBase,
			},
		}

		flowEvents := []struct {
			FlowEvent FlowEvent
			count     int
		}{
			{flowComplete1, 1},
			{flowComplete2, 2},
			{flowPurge, 1},
		}

		flowProcessor := NewFlowProcessor()

		for _, fe := range flowEvents {
			flowProcessor.Process(fe.FlowEvent)
			events := flowProcessor.GetEvents()
			if len(events) != fe.count {
				t.Errorf("Expected %d events, got %d", fe.count, len(events))
			}
		}
	})

	t.Run("handles stats correctly", func(t *testing.T) {
		completedFlow := createFlowCompleteEvent(t)
		tmpFlow := createFlowStatsEvent(t)
		completedFlowUpdate := FlowEvent{
			Type: FlowTypeStats,
			Flow: FlowStats{
				FlowBase:   completedFlow.Flow.(FlowComplete).FlowBase,
				Stats:      tmpFlow.Flow.(FlowStats).Stats,
				LastSeenAt: tmpFlow.Flow.(FlowStats).LastSeenAt,
			},
		}

		flowProcessor := NewFlowProcessor()
		flowProcessor.Process(completedFlow)
		flowProcessor.Process(completedFlowUpdate)
		events := flowProcessor.GetEvents()
		if len(events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(events))
		}
		storedFlow := events[completedFlow.Flow.(FlowComplete).Digest]
		assertEqual(
			t,
			storedFlow.Flow.(FlowComplete).LastSeenAt,
			completedFlowUpdate.Flow.(FlowStats).LastSeenAt,
			"LastSeenAt",
		)
		assertEqual(
			t,
			storedFlow.Flow.(FlowComplete).LocalBytes,
			completedFlowUpdate.Flow.(FlowStats).LocalBytes,
			"LocalBytes",
		)
		assertEqual(
			t,
			storedFlow.Flow.(FlowComplete).LocalPackets,
			completedFlowUpdate.Flow.(FlowStats).LocalPackets,
			"LocalPackets",
		)
		assertEqual(
			t,
			storedFlow.Flow.(FlowComplete).LocalRate,
			completedFlowUpdate.Flow.(FlowStats).LocalRate,
			"LocalRate",
		)
		assertEqual(
			t,
			storedFlow.Flow.(FlowComplete).OtherBytes,
			completedFlowUpdate.Flow.(FlowStats).OtherBytes,
			"OtherBytes",
		)
		assertEqual(
			t,
			storedFlow.Flow.(FlowComplete).OtherPackets,
			completedFlowUpdate.Flow.(FlowStats).OtherPackets,
			"OtherPackets",
		)
		assertEqual(
			t,
			storedFlow.Flow.(FlowComplete).OtherRate,
			completedFlowUpdate.Flow.(FlowStats).OtherRate,
			"OtherRate",
		)
	})

	t.Run("prunes flows correctly", func(t *testing.T) {
		flowComplete1 := createFlowCompleteEvent(t)
		flowComplete2 := createFlowCompleteEvent(t)
		pruneFlow1 := FlowEvent{
			Type: FlowTypePurge,
			Flow: FlowPurge{
				FlowBase: flowComplete1.Flow.(FlowComplete).FlowBase,
			},
		}
		pruneFlow2 := FlowEvent{
			Type: FlowTypePurge,
			Flow: FlowPurge{
				FlowBase: flowComplete2.Flow.(FlowComplete).FlowBase,
			},
		}
		flowProcessor := NewFlowProcessor()
		flowProcessor.Process(flowComplete1)
		flowProcessor.Process(flowComplete2)
		flowProcessor.Process(pruneFlow1)
		events := flowProcessor.GetEvents()
		if len(events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(events))
		}
		flowProcessor.Process(pruneFlow2)
		events = flowProcessor.GetEvents()
		if len(events) != 0 {
			t.Errorf("Expected 0 events, got %d", len(events))
		}
		// Prune again to ensure no panic on the unknown flow
		flowProcessor.Process(pruneFlow1)
		events = flowProcessor.GetEvents()
		if len(events) != 0 {
			t.Errorf("Expected 0 events, got %d", len(events))
		}
	})

	t.Run("stats for unknown flow", func(t *testing.T) {
		unknownFlowStats := createFlowStatsEvent(t)
		flowProcessor := NewFlowProcessor()
		flowProcessor.Process(unknownFlowStats)
		events := flowProcessor.GetEvents()
		if len(events) != 0 {
			t.Errorf("Expected 0 events, got %d", len(events))
		}
	})

	t.Run("remove older flows", func(t *testing.T) {
		lastSeenFlowCompleted := []time.Time{
			time.Now().Add(-599 * time.Second),
			time.Now().Add(-1 * time.Hour),
			time.Now().Add(-11 * time.Minute),
			time.Now().Add(-3 * time.Minute),
			time.Now().Add(10 * time.Second),
			time.Now().Add(1 * time.Second),
			time.Now(),
		}

		processor := NewFlowProcessor()

		for _, ts := range lastSeenFlowCompleted {
			processor.Process(FlowEvent{
				Type: FlowTypeDpiComplete,
				Flow: FlowComplete{
					FlowBase: FlowBase{
						Digest: randomDigest(t),
					},
					LastSeenAt: ts.UnixMilli(),
				},
			})
		}

		processor.PurgeFlowsOlderThan(10 * time.Minute)
		events := processor.GetEvents()
		if len(events) != 5 {
			t.Errorf("Expected 5 events, got %d", len(events))
		}
	})

	t.Run("concurrent access is safe", func(t *testing.T) {
		flowProcessor := NewFlowProcessor()
		var wg sync.WaitGroup
		stop := make(chan struct{})

		// Helper to create event without using t (which is not thread-safe for concurrent use)
		createEventSafe := func() FlowEvent {
			const length = 16
			const hexChars = "0123456789abcdef"
			var builder strings.Builder
			builder.Grow(length)
			for i := 0; i < length; i++ {
				builder.WriteByte(hexChars[rand.IntN(16)])
			}
			return FlowEvent{
				Type: FlowTypeDpiComplete,
				Flow: FlowComplete{
					FlowBase: FlowBase{
						Digest: builder.String(),
					},
				},
			}
		}

		// Populate initial data
		for i := 0; i < 100; i++ {
			flowProcessor.Process(createEventSafe())
		}

		// Writer: modifies the map concurrently
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					flowProcessor.Process(createEventSafe())
					// Yield slightly to allow interleaving
					time.Sleep(time.Microsecond)
				}
			}
		}()

		// Reader: iterates over the map returned by GetEvents
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					events := flowProcessor.GetEvents()
					// If GetEvents returns the internal map reference, iterating here
					// while the writer modifies it will cause a panic:
					// "fatal error: concurrent map iteration and map write"
					for range events {
					}
				}
			}
		}()

		// Let the race condition manifest
		time.Sleep(100 * time.Millisecond)
		close(stop)
		wg.Wait()
	})
}
