package lrucache

import (
	"time"
	"sync"
)

type ComputeValue func() (value interface{}, ttl time.Duration, size int)

// Because more implementations might follow, this is a interface.
// See README.md for more general information.
type Cache interface {
	// Get and set values
	Get(key string, computeValue ComputeValue) interface{}

	// Delete a value, returning true if it was in the cache
	Del(key string) bool

	// Call f on every key in the cache,
	// evict all expired keys and do some sanity checks.
	Keys(f func(key string, val interface{}))
}

type cacheEntry struct {
	key string
	value interface{}

	expiration time.Time
	size int

	next, prev *cacheEntry
}

type lruCache struct {
	mutex sync.Mutex
	maxmemory, usedmemory int
	entries map[string]*cacheEntry
	head, tail *cacheEntry
}

func (c *lruCache) Get(key string, computeValue ComputeValue) interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()

	if entry, ok := c.entries[key]; ok {
		if now.After(entry.expiration) {
			c.evictEntry(entry)
		} else {
			if entry != c.head {
				if entry.prev != nil {
					entry.prev.next = entry.next
				}
				if entry.next != nil {
					entry.next.prev = entry.prev
				}
				if entry == c.tail {
					c.tail = entry.prev
					if c.tail == nil {
						panic("Hä?")
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
		expiration: now.Add(ttl),
		size: size,
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
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if entry, ok := c.entries[key]; ok {
		c.evictEntry(entry)
		return true
	}
	return false
}

func (c *lruCache) Keys(f func(key string, val interface{})) {
	now := time.Now()

	size := 0
	for key, e := range c.entries {
		if key != e.key {
			panic("key mismatch")
		}

		if now.After(e.expiration) {
			c.evictEntry(e)
			continue
		}

		if e.prev != nil {
			if e.prev.next != e {
				panic("list corrupted")
			}
		}

		if e.next != nil {
			if e.next.prev != e {
				panic("list corrupted")
			}
		}

		size += e.size
		f(key, e.value)
	}

	if size != c.usedmemory {
		panic("size calculations failed")
	}

	if c.head != nil {
		if c.tail == nil || c.head.prev != nil {
			panic("head/tail corrupted")
		}
	}

	if c.tail != nil {
		if c.head == nil || c.tail.next != nil {
			panic("head/tail corrupted")
		}
	}
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
		entries: map[string]*cacheEntry{},
	}
}

