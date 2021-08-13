package lrucache

import "time"

type ComputeValue func() (value interface{}, ttl int, size int)

type Cache interface {
	Get(key string, computeValue ComputeValue) interface{}
	Del(key string) bool
}

type cacheEntry struct {
	key string
	value interface{}

	expiration time.Time
	size int

	next, prev *cacheEntry
}

type lruCache struct {
	maxmemory, usedmemory int
	entries map[string]*cacheEntry
	head, tail *cacheEntry
}

func (c *lruCache) Get(key string, computeValue ComputeValue) interface{} {
	now := time.Now()

	if entry, ok := c.entries[key]; ok {
		if entry.expiration.After(now) {
			c.evictEntry(entry)
		} else {
			if e != c.head {
				if entry.prev != nil {
					entry.prev.next = entry.next
				}
				if entry.next != nil {
					entry.next.prev = entry.prev
				}
				if e == c.tail {
					c.tail = e.prev
					if c.tail == nil {
						panic()
					}
				}
				c.insertFront(entry);
			}
			return entry.value
		}
	}

	value, ttl, size := computeValue()
	entry := &cacheEntry{
		key: key,
		value: value,
		expiration: now.Add(time.Duration(ttl) * time.Second),
		size: size
	}

	c.insertFront(entry)
	c.usedmemory += size
	c.entries[key] = entry

	for c.usedmemory > c.maxmemory && c.tail != nil {
		c.evictEntry(c.tail)
	}

	return value
}

func (c *lruCache) Del(key string) bool {
	if entry, ok := c.entries[key]; ok {
		c.evictEntry(entry)
		return true
	}
	return false
}

func (c *lruCache) insertFront(e *cacheEntry) {
	e.next = c.head
	c.head = e

	e.prev = nil
	if e.next != nil {
		e.next.prev = e
	}

	if c.tail == nil {
		c.tail = e
	}
}

func (c *lruCache) evictEntry(e *cacheEntry) {
	if e.prev != nil {
		e.prev.next = e.next
	}

	if e.next != nil {
		e.next.prev = e.prev
	}

	if e == c.head {
		c.head = e.next
	}

	if e == c.tail {
		c.tail = e.prev
	}

	c.usedmemory -= e.size
	delete(c.entries, e.key)
}

func New(maxmemory int) Cache {
	return &lruCache{
		maxmemory: maxmemory,
		usedmemory: 0,
		entries: map[string]*cacheEntry{},
		head: nil,
	}
}

