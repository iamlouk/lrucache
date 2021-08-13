package lrucache

import "time"

type ComputeValue func() (value interface{}, ttl int, size int)

type Cache interface {
	Get(key string, computeValue ComputeValue) interface{}
}

type cacheEntry struct {
	key string
	value interface{}

	expiration time.Time
	size int

	next, prev *cacheEntry
}

type lruCache struct {
	maxmemory int
	usedmemory int
	entries map[string]*cacheEntry
	head *cacheEntry
}

func (c *lruCache) Get(key string, computeValue ComputeValue) interface{} {
	now := time.Now()

	if entry, ok := c.entries[key]; ok {
		if entry.expiration.After(now) {
			if entry.prev != nil {
				entry.prev.next = entry.next
			}
			if entry.next != nil {
				entry.next.prev = entry.prev
			}

			c.usedmemory -= entry.size
			delete(c.entries, key)
		} else {
			return entry.value
		}
	}

	value, ttl, size := computeValue()
	entry := &cacheEntry{
		key: key,
		value: value,
		expiration: now.Add(time.Duration(ttl) * time.Second),
		size: size,
		next: c.head,
		prev: nil,
	}

	c.usedmemory += size
	c.entries[key] = entry
	if c.head != nil {
		c.head.prev = entry
	}
	c.head = entry

	if c.usedmemory > c.maxmemory {
		c.evict()
	}

	return value
}

// TODO: Try to evict expired values!
func (c *lruCache) evict() {
	for c.usedmemory > c.maxmemory && c.head != nil {
		entry := c.head
		c.head = entry.next
		c.head.prev = nil
		c.usedmemory -= entry.size
		delete(c.entries, entry.key)
	}
}

func New(maxmemory int) Cache {
	return &lruCache{
		maxmemory: maxmemory,
		usedmemory: 0,
		entries: map[string]*cacheEntry{},
		head: nil,
	}
}

