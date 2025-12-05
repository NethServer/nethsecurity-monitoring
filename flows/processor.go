package flows

import (
	"log/slog"
	"sync"
	"time"
)

type FlowProcessor struct {
	eventMap map[string]FlowEvent
	mu       sync.RWMutex
}

func NewFlowProcessor() *FlowProcessor {
	return &FlowProcessor{
		eventMap: make(map[string]FlowEvent),
	}
}

func (fp *FlowProcessor) Process(event FlowEvent) {
	fp.mu.Lock()
	defer fp.mu.Unlock()
	switch f := event.Flow.(type) {
	case FlowStart:
		slog.Debug("Flow start", "digest", f.Digest)
		fp.eventMap[f.Digest] = event
	case FlowComplete:
		slog.Debug("Flow complete", "digest", f.Digest)
		fp.eventMap[f.Digest] = event
	case FlowStats:
		slog.Debug("Flow stats received", "type", event.Type, "digest", f.Digest)
		flow, ok := fp.eventMap[f.Digest]
		if !ok {
			slog.Debug("Flow stats received for unknown flow", "digest", f.Digest)
			return
		}
		switch toUpdateFlow := flow.Flow.(type) {
		case FlowStart:
			toUpdateFlow.LastSeenAt = f.LastSeenAt
			flow.Flow = toUpdateFlow
			fp.eventMap[f.Digest] = flow
		case FlowComplete:
			toUpdateFlow.LastSeenAt = f.LastSeenAt
			toUpdateFlow.LocalBytes += f.LocalBytes
			toUpdateFlow.LocalPackets += f.LocalPackets
			toUpdateFlow.LocalRate = f.LocalRate
			toUpdateFlow.OtherBytes += f.OtherBytes
			toUpdateFlow.OtherPackets += f.OtherPackets
			toUpdateFlow.OtherRate = f.OtherRate
			toUpdateFlow.TotalPackets = f.TotalPackets
			toUpdateFlow.TotalBytes = f.TotalBytes
			flow.Flow = toUpdateFlow
			fp.eventMap[f.Digest] = flow
		}
	}
}

func (fp *FlowProcessor) GetEvents() map[string]FlowEvent {
	fp.mu.RLock()
	defer fp.mu.RUnlock()
	eventsCopy := make(map[string]FlowEvent)
	for k, v := range fp.eventMap {
		eventsCopy[k] = v
	}
	return eventsCopy
}

func (fp *FlowProcessor) PurgeFlowsOlderThan(olderThan time.Duration) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	digests := make([]string, 0)
	cutoff := time.Now().Add(-olderThan)
	for v, f := range fp.eventMap {
		switch f.Flow.(type) {
		case FlowStart:
			if time.UnixMilli(f.Flow.(FlowStart).LastSeenAt).Before(cutoff) {
				digests = append(digests, v)
			}
		case FlowComplete:
			if time.UnixMilli(f.Flow.(FlowComplete).LastSeenAt).Before(cutoff) {
				digests = append(digests, v)
			}
		}
	}
	slog.Debug("Purging flows", "count", len(digests))
	for _, d := range digests {
		delete(fp.eventMap, d)
	}
}
