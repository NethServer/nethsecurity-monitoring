package stats

import (
	"context"
	"net"
	"sort"
	"strings"
	"sync"
	"time"
)

type reverseDNSResolver func(context.Context, string) ([]string, error)

type reverseDNSCacheEntry struct {
	ip       string
	name     string
	requests int
	addedAt  time.Time
}

type ReverseDNSCache struct {
	mu         sync.Mutex
	entries    map[string]*reverseDNSCacheEntry
	resolver   reverseDNSResolver
	ttl        time.Duration
	timeout    time.Duration
	maxRecords int
}

func NewReverseDNSCache() *ReverseDNSCache {
	return NewReverseDNSCacheWithResolver(func(ctx context.Context, ip string) ([]string, error) {
		return net.DefaultResolver.LookupAddr(ctx, ip)
	})
}

func NewReverseDNSCacheWithResolver(resolver reverseDNSResolver) *ReverseDNSCache {
	return &ReverseDNSCache{
		entries:    make(map[string]*reverseDNSCacheEntry),
		resolver:   resolver,
		ttl:        10 * time.Minute,
		timeout:    time.Second,
		maxRecords: 10000,
	}
}

func (c *ReverseDNSCache) Resolve(ctx context.Context, ip string) string {
	if c == nil || ip == "" {
		return ip
	}

	now := time.Now()

	c.mu.Lock()
	if entry, ok := c.entries[ip]; ok {
		if now.Sub(entry.addedAt) <= c.ttl {
			entry.requests++
			name := entry.name
			c.mu.Unlock()
			return name
		}
		delete(c.entries, ip)
	}
	c.mu.Unlock()

	name := ip
	lookupCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	if names, err := c.resolver(lookupCtx, ip); err == nil && len(names) > 0 {
		name = strings.TrimSuffix(names[0], ".")
	}

	c.mu.Lock()
	if entry, ok := c.entries[ip]; ok {
		if now.Sub(entry.addedAt) <= c.ttl {
			entry.requests++
			name = entry.name
			c.mu.Unlock()
			return name
		}
	}

	c.entries[ip] = &reverseDNSCacheEntry{
		ip:       ip,
		name:     name,
		requests: 1,
		addedAt:  now,
	}
	c.pruneLocked(now)
	c.mu.Unlock()

	return name
}

func (c *ReverseDNSCache) Prune(now time.Time) {
	if c == nil {
		return
	}

	c.mu.Lock()
	c.pruneLocked(now)
	c.mu.Unlock()
}

func (c *ReverseDNSCache) pruneLocked(now time.Time) {
	for ip, entry := range c.entries {
		if now.Sub(entry.addedAt) > c.ttl {
			delete(c.entries, ip)
		}
	}

	if len(c.entries) <= c.maxRecords {
		return
	}

	entries := make([]*reverseDNSCacheEntry, 0, len(c.entries))
	for _, entry := range c.entries {
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].requests != entries[j].requests {
			return entries[i].requests < entries[j].requests
		}
		if !entries[i].addedAt.Equal(entries[j].addedAt) {
			return entries[i].addedAt.Before(entries[j].addedAt)
		}
		return entries[i].ip < entries[j].ip
	})

	for len(c.entries) > c.maxRecords && len(entries) > 0 {
		entry := entries[0]
		entries = entries[1:]
		delete(c.entries, entry.ip)
	}
}
