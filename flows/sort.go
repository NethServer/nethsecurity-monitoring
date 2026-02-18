package flows

import "sort"

type SortBy string

const (
	SortByDuration     SortBy = "duration"
	SortByLastSeenAt   SortBy = "last_seen_at"
	SortByDownloadRate SortBy = "download_rate"
	SortByUploadRate   SortBy = "upload_rate"
)

// flowSortKey extracts a numeric sort key from a FlowEvent for the given SortBy field.
// Returns (value, true) when the flow carries the requested metric, or (0, false) when it
// does not (e.g. a bare FlowStart has no rate statistics).
func flowSortKey(ev FlowEvent, by SortBy) (float64, bool) {
	switch by {
	case SortByDownloadRate:
		fc, ok := ev.Flow.(FlowComplete)
		if !ok {
			return 0, false
		}
		if fc.LocalOrigin {
			// local initiated the connection → local is the uploader, other side is the downloader
			return fc.OtherRate, true
		}
		// other initiated → local is the downloader
		return fc.LocalRate, true

	case SortByUploadRate:
		fc, ok := ev.Flow.(FlowComplete)
		if !ok {
			return 0, false
		}
		if fc.LocalOrigin {
			return fc.LocalRate, true
		}
		return fc.OtherRate, true

	case SortByLastSeenAt:
		switch f := ev.Flow.(type) {
		case FlowComplete:
			return float64(f.LastSeenAt), true
		case FlowStart:
			return float64(f.LastSeenAt), true
		case FlowStats:
			return float64(f.LastSeenAt), true
		}
		return 0, false

	case SortByDuration:
		switch f := ev.Flow.(type) {
		case FlowComplete:
			return float64(f.LastSeenAt - f.FirstSeenAt), true
		case FlowStart:
			return float64(f.LastSeenAt - f.FirstSeenAt), true
		}
		return 0, false
	}

	return 0, false
}

// flowDigest returns the digest string from any concrete flow type.
func flowDigest(ev FlowEvent) string {
	switch f := ev.Flow.(type) {
	case FlowComplete:
		return f.Digest
	case FlowStart:
		return f.Digest
	case FlowStats:
		return f.Digest
	case FlowPurge:
		return f.Digest
	}
	return ""
}

// SortEvents sorts events by the given SortBy field, with desc controlling direction.
// Flows that do not carry the requested metric are always placed at the bottom in ascending
// digest order, regardless of the desc flag.
func SortEvents(events []FlowEvent, sortBy SortBy, desc bool) []FlowEvent {
	sort.SliceStable(events, func(i, j int) bool {
		vi, iHas := flowSortKey(events[i], sortBy)
		vj, jHas := flowSortKey(events[j], sortBy)

		switch {
		case iHas && jHas:
			if vi != vj {
				if desc {
					return vi > vj
				}
				return vi < vj
			}
			// equal values: tiebreak ascending by digest
			return flowDigest(events[i]) < flowDigest(events[j])

		case iHas:
			return true // i has value, j does not → i sorts first

		case jHas:
			return false // j has value, i does not → j sorts first

		default:
			// neither has value → ascending digest, ignoring desc
			return flowDigest(events[i]) < flowDigest(events[j])
		}
	})

	return events
}
