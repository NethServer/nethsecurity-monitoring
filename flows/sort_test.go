package flows

import (
	"testing"

	"github.com/go-playground/assert/v2"
)

// digestOf extracts the digest from any concrete flow type.
func digestOf(t *testing.T, ev FlowEvent) string {
	t.Helper()
	switch f := ev.Flow.(type) {
	case FlowStart:
		return f.Digest
	case FlowComplete:
		return f.Digest
	case FlowStats:
		return f.Digest
	case FlowPurge:
		return f.Digest
	default:
		t.Fatalf("unexpected flow type: %T", ev.Flow)
		return ""
	}
}

func assertOrder(t *testing.T, events []FlowEvent, want []string) {
	t.Helper()
	assert.Equal(t, len(want), len(events))
	for i, ev := range events {
		assert.Equal(t, want[i], digestOf(t, ev))
	}
}

// makeComplete builds a FlowEvent wrapping a FlowComplete.
func makeComplete(
	digest string,
	localOrigin bool,
	localRate, otherRate float64,
	firstSeenAt, lastSeenAt int64,
) FlowEvent {
	return FlowEvent{
		Type: FlowTypeDpiComplete,
		Flow: FlowComplete{
			FlowStart: FlowStart{
				FlowBase:    FlowBase{Digest: digest},
				LocalOrigin: localOrigin,
				FirstSeenAt: firstSeenAt,
				LastSeenAt:  lastSeenAt,
			},
			Stats: Stats{LocalRate: localRate, OtherRate: otherRate},
		},
	}
}

// makeStart builds a FlowEvent wrapping a FlowStart (no rate stats).
func makeStart(digest string, firstSeenAt, lastSeenAt int64) FlowEvent {
	return FlowEvent{
		Type: FlowTypeBegin,
		Flow: FlowStart{
			FlowBase:    FlowBase{Digest: digest},
			FirstSeenAt: firstSeenAt,
			LastSeenAt:  lastSeenAt,
		},
	}
}

// makeStats builds a FlowEvent wrapping a FlowStats (has last_seen_at, but no duration/rate).
func makeStats(digest string, lastSeenAt int64) FlowEvent {
	return FlowEvent{
		Type: FlowTypeStats,
		Flow: FlowStats{
			FlowBase:   FlowBase{Digest: digest},
			LastSeenAt: lastSeenAt,
		},
	}
}

func TestSortEvents(t *testing.T) {
	t.Run("empty slice is returned unchanged", func(t *testing.T) {
		result := SortEvents([]FlowEvent{}, SortByDownloadRate, false)
		assert.Equal(t, 0, len(result))
	})

	t.Run("download_rate ascending", func(t *testing.T) {
		// f-001: LocalOrigin=false → download=LocalRate=100
		// f-002: LocalOrigin=true  → download=OtherRate=50
		// f-003: LocalOrigin=false → download=LocalRate=200
		// f-004: FlowStart (no rate) → sinks to bottom
		events := []FlowEvent{
			makeComplete("f-001", false, 100, 10, 0, 0),
			makeComplete("f-002", true, 300, 50, 0, 0),
			makeComplete("f-003", false, 200, 10, 0, 0),
			makeStart("f-004", 0, 0),
		}
		result := SortEvents(events, SortByDownloadRate, false)
		// ascending download rates: 50 < 100 < 200, then FlowStart at bottom
		assertOrder(t, result, []string{"f-002", "f-001", "f-003", "f-004"})
	})

	t.Run("download_rate descending", func(t *testing.T) {
		events := []FlowEvent{
			makeComplete("f-001", false, 100, 10, 0, 0),
			makeComplete("f-002", true, 300, 50, 0, 0),
			makeComplete("f-003", false, 200, 10, 0, 0),
			makeStart("f-004", 0, 0),
		}
		result := SortEvents(events, SortByDownloadRate, true)
		// descending download rates: 200 > 100 > 50, then FlowStart at bottom
		assertOrder(t, result, []string{"f-003", "f-001", "f-002", "f-004"})
	})

	t.Run("download_rate all metric-less sorted by digest ascending", func(t *testing.T) {
		events := []FlowEvent{
			makeStart("f-003", 0, 0),
			makeStart("f-001", 0, 0),
			makeStart("f-002", 0, 0),
		}
		result := SortEvents(events, SortByDownloadRate, false)
		assertOrder(t, result, []string{"f-001", "f-002", "f-003"})
	})

	t.Run("upload_rate ascending", func(t *testing.T) {
		// f-001: LocalOrigin=false → upload=OtherRate=5
		// f-002: LocalOrigin=true  → upload=LocalRate=30
		// f-003: LocalOrigin=false → upload=OtherRate=20
		// f-004: FlowStart (no rate) → sinks to bottom
		events := []FlowEvent{
			makeComplete("f-001", false, 100, 5, 0, 0),
			makeComplete("f-002", true, 30, 999, 0, 0),
			makeComplete("f-003", false, 100, 20, 0, 0),
			makeStart("f-004", 0, 0),
		}
		result := SortEvents(events, SortByUploadRate, false)
		// ascending upload rates: 5 < 20 < 30, then FlowStart at bottom
		assertOrder(t, result, []string{"f-001", "f-003", "f-002", "f-004"})
	})

	t.Run("upload_rate descending", func(t *testing.T) {
		events := []FlowEvent{
			makeComplete("f-001", false, 100, 5, 0, 0),
			makeComplete("f-002", true, 30, 999, 0, 0),
			makeComplete("f-003", false, 100, 20, 0, 0),
			makeStart("f-004", 0, 0),
		}
		result := SortEvents(events, SortByUploadRate, true)
		// descending upload rates: 30 > 20 > 5, then FlowStart at bottom
		assertOrder(t, result, []string{"f-002", "f-003", "f-001", "f-004"})
	})

	t.Run("last_seen_at ascending", func(t *testing.T) {
		// FlowStart, FlowComplete, and FlowStats all carry LastSeenAt
		events := []FlowEvent{
			makeComplete("f-001", false, 0, 0, 0, 300),
			makeStart("f-002", 0, 100),
			makeStats("f-003", 200),
		}
		result := SortEvents(events, SortByLastSeenAt, false)
		assertOrder(t, result, []string{"f-002", "f-003", "f-001"})
	})

	t.Run("last_seen_at descending", func(t *testing.T) {
		events := []FlowEvent{
			makeComplete("f-001", false, 0, 0, 0, 300),
			makeStart("f-002", 0, 100),
			makeStats("f-003", 200),
		}
		result := SortEvents(events, SortByLastSeenAt, true)
		assertOrder(t, result, []string{"f-001", "f-003", "f-002"})
	})

	t.Run("duration ascending", func(t *testing.T) {
		// duration = lastSeenAt - firstSeenAt
		// f-001: 300-100=200  (FlowComplete)
		// f-002: 500-100=400  (FlowStart)
		// f-003: 400-300=100  (FlowComplete)
		// f-004: FlowStats → no duration, sinks to bottom
		events := []FlowEvent{
			makeComplete("f-001", false, 0, 0, 100, 300),
			makeStart("f-002", 100, 500),
			makeComplete("f-003", false, 0, 0, 300, 400),
			makeStats("f-004", 999),
		}
		result := SortEvents(events, SortByDuration, false)
		// durations: 100 < 200 < 400, FlowStats at bottom
		assertOrder(t, result, []string{"f-003", "f-001", "f-002", "f-004"})
	})

	t.Run("duration descending", func(t *testing.T) {
		events := []FlowEvent{
			makeComplete("f-001", false, 0, 0, 100, 300),
			makeStart("f-002", 100, 500),
			makeComplete("f-003", false, 0, 0, 300, 400),
			makeStats("f-004", 999),
		}
		result := SortEvents(events, SortByDuration, true)
		// durations: 400 > 200 > 100, FlowStats at bottom
		assertOrder(t, result, []string{"f-002", "f-001", "f-003", "f-004"})
	})

	t.Run("tiebreak by digest ascending regardless of desc flag", func(t *testing.T) {
		// Three flows with identical download rates; tiebreak must be ascending digest.
		events := []FlowEvent{
			makeComplete("f-003", false, 100, 0, 0, 0),
			makeComplete("f-001", false, 100, 0, 0, 0),
			makeComplete("f-002", false, 100, 0, 0, 0),
		}

		asc := SortEvents(append([]FlowEvent{}, events...), SortByDownloadRate, false)
		assertOrder(t, asc, []string{"f-001", "f-002", "f-003"})

		desc_ := SortEvents(append([]FlowEvent{}, events...), SortByDownloadRate, true)
		assertOrder(t, desc_, []string{"f-001", "f-002", "f-003"})
	})

	t.Run("metric-less tiebreak by digest ascending regardless of desc flag", func(t *testing.T) {
		events := []FlowEvent{
			makeStart("f-003", 0, 0),
			makeStart("f-001", 0, 0),
			makeStart("f-002", 0, 0),
		}

		asc := SortEvents(append([]FlowEvent{}, events...), SortByDownloadRate, false)
		assertOrder(t, asc, []string{"f-001", "f-002", "f-003"})

		desc_ := SortEvents(append([]FlowEvent{}, events...), SortByDownloadRate, true)
		assertOrder(t, desc_, []string{"f-001", "f-002", "f-003"})
	})
}
