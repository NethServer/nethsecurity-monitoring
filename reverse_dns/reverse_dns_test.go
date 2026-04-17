package reverse_dns

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestReverseDns(t *testing.T) {
	t.Run("basic resolution", func(t *testing.T) {
		resolutions := map[string]string{
			"0.0.0.0": "example.com",
			"1.1.1.1": "google.com",
		}
		mockLookup := func(ctx context.Context, ip string) ([]string, error) {
			return []string{resolutions[ip]}, nil
		}
		resolver := NewResolver(mockLookup, 10*time.Minute, 1000)

		for ip, expected := range resolutions {
			name := resolver.Lookup(context.Background(), ip)
			if name != expected {
				t.Fatalf("got %s, expected %s", name, expected)
			}
		}
	})

	t.Run("on fail, returns ip", func(t *testing.T) {
		mockLookup := func(ctx context.Context, ip string) ([]string, error) {
			return nil, errors.New("failed")
		}
		resolver := NewResolver(mockLookup, 10*time.Minute, 1000)
		expected := "0.0.0.0"
		name := resolver.Lookup(context.Background(), expected)
		if name != expected {
			t.Fatalf("got %s, expected %s", name, expected)
		}
	})

	t.Run("cache hit", func(t *testing.T) {
		counter := 0
		nameExpected := "example.com"
		mockLookup := func(ctx context.Context, ip string) ([]string, error) {
			counter++
			return []string{nameExpected}, nil
		}
		resolver := NewResolver(mockLookup, 10*time.Minute, 1000)
		for range 10 {
			name := resolver.Lookup(context.Background(), "0.0.0.0")
			if name != nameExpected {
				t.Fatalf("got %s, expected %s", name, nameExpected)
			}

		}
		if counter != 1 {
			t.Fatalf("got %d, expected %d", counter, 1)
		}
	})

	t.Run("hit count increments", func(t *testing.T) {
		mockLookup := func(ctx context.Context, ip string) ([]string, error) {
			return []string{"example.com"}, nil
		}
		resolver := NewResolver(mockLookup, 10*time.Minute, 1000)

		resolver.Lookup(context.Background(), "0.0.0.0")
		resolver.Lookup(context.Background(), "0.0.0.0")

		resolver.mu.RLock()
		entry := resolver.entries["0.0.0.0"]
		resolver.mu.RUnlock()

		if entry.hit.Load() != 2 {
			t.Fatalf("got %d hits, expected %d", entry.hit.Load(), 2)
		}
		if entry.expiresAt.Before(time.Now()) {
			t.Fatalf("expected expiresAt to be in the future")
		}
	})

	t.Run("concurrent lookup", func(t *testing.T) {
		var counter int32
		mockLookup := func(ctx context.Context, ip string) ([]string, error) {
			atomic.AddInt32(&counter, 1)
			return []string{"example.com"}, nil
		}
		resolver := NewResolver(mockLookup, 10*time.Minute, 1000)

		const workers = 16
		start := make(chan struct{})
		results := make(chan string, workers)
		var wg sync.WaitGroup

		for range workers {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				results <- resolver.Lookup(context.Background(), "0.0.0.0")
			}()
		}

		close(start)
		wg.Wait()
		close(results)

		for name := range results {
			if name != "example.com" {
				t.Fatalf("got %s, expected %s", name, "example.com")
			}
		}

		if got := atomic.LoadInt32(&counter); got != 1 {
			t.Fatalf("got %d lookup calls, expected %d", got, 1)
		}
	})

	t.Run("on cache hit, expired entries are pruned", func(t *testing.T) {
		callCount := 0
		mockLookup := func(ctx context.Context, ip string) ([]string, error) {
			callCount++
			return []string{"resolved-" + ip}, nil
		}
		// Use a negative TTL so entries are immediately expired
		// Use maxEntries=2 to force pruning when adding a third entry
		resolver := NewResolver(mockLookup, -1*time.Second, 2)
		resolver.Lookup(context.Background(), "1.1.1.1")
		if callCount != 1 {
			t.Fatalf("expected 1 lookup, got %d", callCount)
		}
		resolver.Lookup(context.Background(), "2.2.2.2")
		if callCount != 2 {
			t.Fatalf("expected 2 lookups, got %d", callCount)
		}

		// At this point, cache has 2 entries (both expired)
		// Now try to look up a new IP when cache is at capacity - this should trigger pruning of expired entries
		resolver.Lookup(context.Background(), "3.3.3.3")

		// Check that expired entries were pruned during the cache miss
		resolver.mu.RLock()
		_, existsA := resolver.entries["1.1.1.1"]
		_, existsB := resolver.entries["2.2.2.2"]
		resolver.mu.RUnlock()

		if existsA {
			t.Fatalf("expected expired entry A to be pruned, but it exists")
		}
		if existsB {
			t.Fatalf("expected expired entry B to be pruned, but it exists")
		}
	})

	t.Run("on cache hit, non-expired entries are kept", func(t *testing.T) {
		mockLookup := func(ctx context.Context, ip string) ([]string, error) {
			return []string{"example.com"}, nil
		}
		// Use a far-future TTL so entries never expire
		resolver := NewResolver(mockLookup, 1*time.Hour, 1000)
		resolver.Lookup(context.Background(), "1.1.1.1")
		resolver.Lookup(context.Background(), "2.2.2.2")
		resolver.Lookup(context.Background(), "1.1.1.1")

		// Entry B should still exist
		resolver.mu.RLock()
		_, exists := resolver.entries["2.2.2.2"]
		resolver.mu.RUnlock()

		if !exists {
			t.Fatalf("expected non-expired entry to be kept, but it was pruned")
		}
	})

	t.Run("cache is bounded to maxEntries", func(t *testing.T) {
		mockLookup := func(ctx context.Context, ip string) ([]string, error) {
			return []string{"example.com"}, nil
		}
		resolver := NewResolver(mockLookup, 1*time.Hour, 3)

		// Add 3 entries (fill the cache)
		resolver.Lookup(context.Background(), "1.1.1.1")
		resolver.Lookup(context.Background(), "2.2.2.2")
		resolver.Lookup(context.Background(), "3.3.3.3")

		resolver.mu.RLock()
		if len(resolver.entries) != 3 {
			t.Fatalf("expected 3 entries, got %d", len(resolver.entries))
		}
		resolver.mu.RUnlock()

		// Add one more entry - should trigger eviction
		resolver.Lookup(context.Background(), "4.4.4.4")

		resolver.mu.RLock()
		if len(resolver.entries) != 3 {
			t.Fatalf("expected 3 entries after adding to full cache, got %d", len(resolver.entries))
		}
		resolver.mu.RUnlock()
	})

	t.Run("evicts entry with lowest hit count", func(t *testing.T) {
		mockLookup := func(ctx context.Context, ip string) ([]string, error) {
			return []string{"example.com"}, nil
		}
		resolver := NewResolver(mockLookup, 1*time.Hour, 3)

		// Add 3 entries
		resolver.Lookup(context.Background(), "1.1.1.1") // hit count: 1
		resolver.Lookup(context.Background(), "2.2.2.2") // hit count: 1
		resolver.Lookup(context.Background(), "3.3.3.3") // hit count: 1

		// Increase hit count for entries 2 and 3
		resolver.Lookup(context.Background(), "2.2.2.2") // hit count: 2
		resolver.Lookup(context.Background(), "3.3.3.3") // hit count: 2

		// Add one more entry - should evict 1.1.1.1 (lowest hit count)
		resolver.Lookup(context.Background(), "4.4.4.4")

		resolver.mu.RLock()
		_, exists1 := resolver.entries["1.1.1.1"]
		_, exists2 := resolver.entries["2.2.2.2"]
		_, exists3 := resolver.entries["3.3.3.3"]
		_, exists4 := resolver.entries["4.4.4.4"]
		resolver.mu.RUnlock()

		if exists1 {
			t.Fatalf("expected entry 1.1.1.1 (lowest hit) to be evicted, but it exists")
		}
		if !exists2 || !exists3 || !exists4 {
			t.Fatalf("expected entries 2, 3, 4 to exist")
		}
	})

	t.Run("on hit tie, evicts soonest-to-expire entry", func(t *testing.T) {
		mockLookup := func(ctx context.Context, ip string) ([]string, error) {
			return []string{"example.com"}, nil
		}
		resolver := NewResolver(mockLookup, 1*time.Hour, 3)

		// Add 3 entries with same hit count but different TTLs
		// Entry 1: expires in 1 hour
		resolver.Lookup(context.Background(), "1.1.1.1")

		// Entry 2: expires in 1 hour
		resolver.Lookup(context.Background(), "2.2.2.2")

		// Entry 3: use a shorter TTL by creating it with a resolver that expires sooner
		resolver.Lookup(context.Background(), "3.3.3.3")

		// Manually adjust entry 3 to have a nearer expiration time
		resolver.mu.Lock()
		entry3 := resolver.entries["3.3.3.3"]
		entry3.expiresAt = time.Now().Add(10 * time.Minute) // Expires sooner
		resolver.mu.Unlock()

		// Add one more entry - should evict 3.3.3.3 (soonest to expire)
		resolver.Lookup(context.Background(), "4.4.4.4")

		resolver.mu.RLock()
		_, exists1 := resolver.entries["1.1.1.1"]
		_, exists2 := resolver.entries["2.2.2.2"]
		_, exists3 := resolver.entries["3.3.3.3"]
		_, exists4 := resolver.entries["4.4.4.4"]
		resolver.mu.RUnlock()

		if exists3 {
			t.Fatalf("expected entry 3.3.3.3 (soonest to expire) to be evicted, but it exists")
		}
		if !exists1 || !exists2 || !exists4 {
			t.Fatalf("expected entries 1, 2, 4 to exist")
		}
	})
}
