package feed

import (
	"testing"
	"time"
)

func TestTelegramSummaryCacheGetSet(t *testing.T) {
	cache := newTelegramSummaryCache(2)
	if cache == nil {
		t.Fatalf("expected cache instance")
	}

	now := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	cache.set("key", "value", now.Add(time.Hour), now)

	summary, ok := cache.get("key", now)
	if !ok {
		t.Fatalf("expected cached summary to be present")
	}

	if summary != "value" {
		t.Fatalf("unexpected summary: %q", summary)
	}
}

func TestTelegramSummaryCacheExpiresEntries(t *testing.T) {
	cache := newTelegramSummaryCache(2)
	now := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	cache.set("key", "value", now.Add(time.Minute), now)

	if _, ok := cache.get("key", now.Add(2*time.Minute)); ok {
		t.Fatalf("expected cache entry to expire")
	}

	if len(cache.entries) != 0 {
		t.Fatalf("expected expired cache entry to be removed")
	}
}

func TestTelegramSummaryCacheEvictsLeastRecentlyUsed(t *testing.T) {
	cache := newTelegramSummaryCache(2)
	now := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	expiresAt := now.Add(time.Hour)

	cache.set("a", "summary-a", expiresAt, now)
	cache.set("b", "summary-b", expiresAt, now)

	if _, ok := cache.get("a", now); !ok {
		t.Fatalf("expected entry a to exist before eviction check")
	}

	cache.set("c", "summary-c", expiresAt, now)

	if _, ok := cache.get("a", now); !ok {
		t.Fatalf("expected entry a to remain after evicting least recently used")
	}

	if _, ok := cache.get("b", now); ok {
		t.Fatalf("expected entry b to be evicted")
	}

	if _, ok := cache.get("c", now); !ok {
		t.Fatalf("expected entry c to be cached")
	}
}
