package reverse_dns

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type lookupAddress func(context.Context, string) ([]string, error)

type cacheEntry struct {
	ip        string
	name      string
	hit       atomic.Int64
	expiresAt time.Time
}

type Resolver struct {
	mu         sync.RWMutex
	lookup     lookupAddress
	cacheTTL   time.Duration
	maxEntries int
	entries    map[string]*cacheEntry
	hits       atomic.Int64
	misses     atomic.Int64
}

func NewResolver(lookup lookupAddress, cacheTTL time.Duration, maxEntries int) *Resolver {
	return &Resolver{
		entries:    make(map[string]*cacheEntry),
		lookup:     lookup,
		cacheTTL:   cacheTTL,
		maxEntries: maxEntries,
	}
}

func (r *Resolver) Lookup(ctx context.Context, ip string) string {
	r.mu.RLock()
	// Try cache hit with read lock (lock-free for atomic operations)
	if entry, ok := r.entries[ip]; ok && time.Now().Before(entry.expiresAt) {
		name := entry.name
		entry.hit.Add(1)
		r.hits.Add(1)
		r.mu.RUnlock()
		return name
	}
	r.misses.Add(1)
	r.mu.RUnlock()

	// Cache miss or expired
	timeoutCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	name := ip
	if names, err := r.lookup(timeoutCtx, ip); err == nil && len(names) > 0 {
		name = strings.TrimSuffix(names[0], ".")
	}

	// Add to cache (requires write lock)
	r.mu.Lock()
	// Double-check: another goroutine may have added it while we were looking up
	if entry, ok := r.entries[ip]; ok && time.Now().Before(entry.expiresAt) {
		entry.hit.Add(1)
		r.mu.Unlock()
		return entry.name
	}

	// If cache is full, prune expired entries first, then evict if still full
	if len(r.entries) >= r.maxEntries {
		r.pruneExpired()
		if len(r.entries) >= r.maxEntries {
			r.evictOne()
		}
	}

	entry := &cacheEntry{
		ip:        ip,
		name:      name,
		expiresAt: time.Now().Add(r.cacheTTL),
	}
	entry.hit.Store(1)
	r.entries[ip] = entry

	r.mu.Unlock()
	return name
}

// PruneExpired removes entries that have exceeded cacheTTL.
// Must be called while r.mu is held (either read or write lock).
func (r *Resolver) PruneExpired() {
	now := time.Now()
	for ip, entry := range r.entries {
		if now.After(entry.expiresAt) {
			delete(r.entries, ip)
		}
	}
}

// pruneExpired is an internal helper that prunes expired entries.
// Must be called while r.mu is held (write lock).
func (r *Resolver) pruneExpired() {
	r.PruneExpired()
}

// evictOne removes the entry with the lowest hit count (tie-break: soonest to expire).
// Must be called while r.mu is held.
func (r *Resolver) evictOne() {
	var victimIP string
	var victimEntry *cacheEntry
	first := true

	for ip, entry := range r.entries {
		if first {
			victimIP = ip
			victimEntry = entry
			first = false
			continue
		}

		// Lower hit count wins
		if entry.hit.Load() < victimEntry.hit.Load() {
			victimIP = ip
			victimEntry = entry
			continue
		}

		// Same hit count: earlier expiration wins (tie-break)
		if entry.hit.Load() == victimEntry.hit.Load() &&
			entry.expiresAt.Before(victimEntry.expiresAt) {
			victimIP = ip
			victimEntry = entry
		}
	}

	if victimIP != "" {
		delete(r.entries, victimIP)
	}
}

// CacheStats holds cache statistics snapshot.
type CacheStats struct {
	Size     int
	Hits     int64
	Misses   int64
	MissRate float64
}

// Stats returns a snapshot of current cache statistics.
func (r *Resolver) Stats() CacheStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	hits := r.hits.Load()
	misses := r.misses.Load()
	missRate := 0.0
	if hits > 0 {
		missRate = float64(misses) / float64(hits)
	}

	return CacheStats{
		Size:     len(r.entries),
		Hits:     hits,
		Misses:   misses,
		MissRate: missRate,
	}
}
