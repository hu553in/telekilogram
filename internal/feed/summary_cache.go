package feed

import (
	"container/list"
	"sync"
	"time"
)

const telegramSummaryCacheMaxEntries = 1024

type telegramSummaryCache struct {
	mu         sync.Mutex
	entries    map[string]*list.Element
	order      *list.List
	maxEntries int
}

type telegramSummaryCacheEntry struct {
	key       string
	summary   string
	expiresAt time.Time
}

func newTelegramSummaryCache(maxEntries int) *telegramSummaryCache {
	if maxEntries <= 0 {
		return nil
	}

	return &telegramSummaryCache{
		entries:    make(map[string]*list.Element, maxEntries),
		order:      list.New(),
		maxEntries: maxEntries,
	}
}

func (c *telegramSummaryCache) get(key string, now time.Time) (string, bool) {
	if c == nil || key == "" {
		return "", false
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	elem, ok := c.entries[key]
	if !ok {
		return "", false
	}

	entry, ok := elem.Value.(*telegramSummaryCacheEntry)
	if !ok {
		return "", false
	}

	if now.After(entry.expiresAt) {
		c.removeElement(elem)

		return "", false
	}

	c.order.MoveToFront(elem)

	return entry.summary, true
}

func (c *telegramSummaryCache) set(
	key string,
	summary string,
	expiresAt time.Time,
	now time.Time,
) {
	if c == nil || key == "" || summary == "" || expiresAt.IsZero() {
		return
	}

	if !expiresAt.After(now) {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.entries[key]; ok {
		entry, castOk := elem.Value.(*telegramSummaryCacheEntry)
		if !castOk {
			return
		}

		entry.summary = summary
		entry.expiresAt = expiresAt
		c.order.MoveToFront(elem)

		return
	}

	elem := c.order.PushFront(&telegramSummaryCacheEntry{
		key:       key,
		summary:   summary,
		expiresAt: expiresAt,
	})
	c.entries[key] = elem

	c.evictExpiredLocked(now)
	c.enforceSizeLimitLocked()
}

func (c *telegramSummaryCache) evictExpiredLocked(now time.Time) {
	for elem := c.order.Back(); elem != nil; {
		prev := elem.Prev()
		entry, ok := elem.Value.(*telegramSummaryCacheEntry)
		if !ok {
			continue
		}

		if now.After(entry.expiresAt) {
			c.removeElement(elem)
		}
		elem = prev
	}
}

func (c *telegramSummaryCache) enforceSizeLimitLocked() {
	for len(c.entries) > c.maxEntries {
		elem := c.order.Back()
		if elem == nil {
			return
		}
		c.removeElement(elem)
	}
}

func (c *telegramSummaryCache) removeElement(elem *list.Element) {
	entry, ok := elem.Value.(*telegramSummaryCacheEntry)
	if !ok {
		return
	}

	delete(c.entries, entry.key)
	c.order.Remove(elem)
}
