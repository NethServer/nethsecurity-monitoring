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

func createFlowStart(t *testing.T) FlowStart {
	t.Helper()
	return FlowStart{
		FlowBase: createFlowBase(t),
	}
}

// createFlowStartEvent creates a FlowStart event with a unique digest.
func createFlowStartEvent(t *testing.T) FlowEvent {
	t.Helper()
	return FlowEvent{
		Type: FlowTypeBegin,
		Flow: createFlowStart(t),
	}
}

// createFlowCompleteEvent creates a FlowComplete event with a unique digest.
func createFlowCompleteEvent(t *testing.T) FlowEvent {
	t.Helper()
	flowStart := createFlowStart(t)
	return FlowEvent{
		Type: FlowTypeDpiComplete,
		Flow: FlowComplete{
			FlowStart: flowStart,
		},
	}
}

func createFlowStatsEvent(t *testing.T) FlowEvent {
	t.Helper()
	minRate := 1000.0
	maxRate := 10000.0
	flowStart := createFlowStart(t)
	return FlowEvent{
		Type: FlowTypeStats,
		Flow: FlowStats{
			FlowBase: flowStart.FlowBase,
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
				event := FlowEvent{
					Type: FlowTypeBegin,
					Flow: createFlowStart(t),
				}
				flowProcessor.Process(event)
			}()
		}

		wg.Wait()

		events := flowProcessor.GetEvents()
		if len(events) != wantedCount {
			t.Errorf("Expected %d events, got %d", wantedCount, len(events))
		}
	})

	t.Run("save a flow start correctly", func(t *testing.T) {
		flow := createFlowStartEvent(t)
		flowProcessor := NewFlowProcessor()
		flowProcessor.Process(flow)
		events := flowProcessor.GetEvents()
		if len(events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(events))
		}
		if _, ok := flow.Flow.(FlowStart); !ok {
			t.Errorf("Expected flow to be of type FlowStart")
		}
		if _, ok := events[flow.Flow.(FlowStart).Digest]; !ok {
			t.Errorf("Expected event with digest %s to be present", flow.Flow.(FlowStart).Digest)
		}
	})

	t.Run("handles flows correctly", func(t *testing.T) {
		startedFlow := createFlowStartEvent(t)
		completedFlow := FlowEvent{
			Type: FlowTypeDpiComplete,
			Flow: FlowComplete{
				FlowStart: startedFlow.Flow.(FlowStart),
			},
		}
		flowPurge := FlowEvent{
			Type: FlowTypePurge,
			Flow: FlowPurge{
				FlowBase: startedFlow.Flow.(FlowStart).FlowBase,
			},
		}

		flowEvents := []struct {
			FlowEvent FlowEvent
			count     int
		}{
			{startedFlow, 1},
			{completedFlow, 1},
			{createFlowStartEvent(t), 2},
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
		startedFlow := createFlowStartEvent(t)
		tmpFlow := createFlowStatsEvent(t)
		startedFlowUpdate := FlowEvent{
			Type: FlowTypeStats,
			Flow: FlowStats{
				FlowBase:   startedFlow.Flow.(FlowStart).FlowBase,
				LastSeenAt: tmpFlow.Flow.(FlowStats).LastSeenAt,
			},
		}
		completedFlow := createFlowCompleteEvent(t)
		completedFlowUpdate := FlowEvent{
			Type: FlowTypeStats,
			Flow: FlowStats{
				FlowBase:   completedFlow.Flow.(FlowComplete).FlowBase,
				Stats:      tmpFlow.Flow.(FlowStats).Stats,
				LastSeenAt: tmpFlow.Flow.(FlowStats).LastSeenAt,
			},
		}

		flowProcessor := NewFlowProcessor()
		flowProcessor.Process(startedFlow)
		flowProcessor.Process(startedFlowUpdate)
		events := flowProcessor.GetEvents()
		if len(events) != 1 {
			t.Errorf("Expected 1 event, got %d", len(events))
		}
		storedFlow, ok := events[startedFlow.Flow.(FlowStart).Digest]
		if !ok {
			t.Errorf(
				"Expected event with digest %s to be present",
				startedFlow.Flow.(FlowStart).Digest,
			)
		}
		assertEqual(
			t,
			storedFlow.Flow.(FlowStart).LastSeenAt,
			startedFlowUpdate.Flow.(FlowStats).LastSeenAt,
			"LastSeenAt",
		)

		flowProcessor.Process(completedFlow)
		flowProcessor.Process(completedFlowUpdate)
		events = flowProcessor.GetEvents()
		if len(events) != 2 {
			t.Errorf("Expected 2 events, got %d", len(events))
		}
		storedFlow = events[completedFlow.Flow.(FlowComplete).Digest]
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
		startedFlow := createFlowStartEvent(t)
		completedFlow := createFlowCompleteEvent(t)
		pruneCompletedFlow := FlowEvent{
			Type: FlowTypePurge,
			Flow: FlowPurge{
				FlowBase: completedFlow.Flow.(FlowComplete).FlowBase,
			},
		}
		pruneStartedFlow := FlowEvent{
			Type: FlowTypePurge,
			Flow: FlowPurge{
				FlowBase: startedFlow.Flow.(FlowStart).FlowBase,
			},
		}
		flowProcessor := NewFlowProcessor()
		flowProcessor.Process(startedFlow)
		flowProcessor.Process(completedFlow)
		flowProcessor.Process(pruneCompletedFlow)
		events := flowProcessor.GetEvents()
		if len(events) != 1 {
			t.Errorf("Expected 1 events, got %d", len(events))
		}
		flowProcessor.Process(pruneStartedFlow)
		events = flowProcessor.GetEvents()
		if len(events) != 0 {
			t.Errorf("Expected 0 events, got %d", len(events))
		}
		// Prune again to ensure no panic on the unknown flow
		flowProcessor.Process(pruneCompletedFlow)
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

		lastSeenFlowStarted := []time.Time{
			time.Now().Add(-2 * time.Hour),
			time.Now().Add(-15 * time.Minute),
			time.Now().Add(-5 * time.Minute),
			time.Now().Add(-1 * time.Minute),
			time.Now(),
		}

		processor := NewFlowProcessor()

		for _, ts := range lastSeenFlowCompleted {
			processor.Process(FlowEvent{
				Type: FlowTypeDpiComplete,
				Flow: FlowComplete{
					FlowStart: FlowStart{
						FlowBase: FlowBase{
							Digest: randomDigest(t),
						},
						LastSeenAt: ts.UnixMilli(),
					},
				},
			})
		}

		for _, ts := range lastSeenFlowStarted {
			processor.Process(FlowEvent{
				Type: FlowTypeBegin,
				Flow: FlowStart{
					FlowBase: FlowBase{
						Digest: randomDigest(t),
					},
					LastSeenAt: ts.UnixMilli(),
				},
			})
		}

		processor.PurgeFlowsOlderThan(10 * time.Minute)
		events := processor.GetEvents()
		if len(events) != 8 {
			t.Errorf("Expected 8 events, got %d", len(events))
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
				Type: FlowTypeBegin,
				Flow: FlowStart{
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
