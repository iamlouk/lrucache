package lrucache

import (
	"sync"
	"time"
)

// Type of the closure that must be passed to `Get` to
// compute the value in case it is not cached.
//
// returned values are the computed value to be stored in the cache,
// the duration until this value will expire and a size estimate.
type ComputeValue func() (value interface{}, ttl time.Duration, size int)

type cacheEntry struct {
	key   string
	value interface{}

	expiration            time.Time
	size                  int
	waitingForComputation int

	next, prev *cacheEntry
}

type Cache struct {
	mutex                 sync.Mutex
	cond                  *sync.Cond
	maxmemory, usedmemory int
	entries               map[string]*cacheEntry
	head, tail            *cacheEntry
}

// Return a new instance of a LRU In-Memory Cache.
// Read [the README](./README.md) for more information
// on what is going on with `maxmemory`.
func New(maxmemory int) *Cache {
	cache := &Cache{
		maxmemory: maxmemory,
		entries:   map[string]*cacheEntry{},
	}
	cache.cond = sync.NewCond(&cache.mutex)
	return cache
}

// Return the cached value for key `key` or call `computeValue` and
// store its return value in the cache. If called, the closure will be
// called synchronous and __shall not call methods on the same cache__
// or a deadlock might ocure. Read [the README](./README.md) for more
// information on how things work.
func (c *Cache) Get(key string, computeValue ComputeValue) interface{} {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()

	if entry, ok := c.entries[key]; ok {
		// The expiration not being set is what shows us that
		// the computation of that value is still ongoing.
		for entry.expiration.IsZero() {
			entry.waitingForComputation += 1
			c.cond.Wait()
			entry.waitingForComputation -= 1
		}

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
				c.insertFront(entry)
			}
			return entry.value
		}
	}

	entry := &cacheEntry{
		key:                   key,
		waitingForComputation: 1,
	}

	c.entries[key] = entry

	c.mutex.Unlock()
	value, ttl, size := computeValue()
	c.mutex.Lock()

	entry.value = value
	entry.expiration = now.Add(ttl)
	entry.size = size
	entry.waitingForComputation -= 1

	// Only broadcast if other goroutines are actually waiting
	// for a result.
	if entry.waitingForComputation > 0 {
		// TODO: Have more than one condition variable so that there are
		// less unnecessary wakeups.
		c.cond.Broadcast()
	}

	c.usedmemory += size
	c.insertFront(entry)

	// Evict only entries with a size of more than zero.
	// This is the only loop in the implementation outside of the `Keys`
	// method.
	evictionCandidate := c.tail
	for c.usedmemory > c.maxmemory && evictionCandidate != nil {
		nextCandidate := evictionCandidate.prev
		if (evictionCandidate.size > 0 || now.After(evictionCandidate.expiration)) &&
			evictionCandidate.waitingForComputation == 0 {
			c.evictEntry(evictionCandidate)
		}
		evictionCandidate = nextCandidate
	}

	return value
}

// Remove the value at key `key` from the cache.
// Return true if the key was in the cache and false
// otherwise. It is possible that true is returned even
// though the value already expired.
func (c *Cache) Del(key string) bool {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if entry, ok := c.entries[key]; ok {
		c.evictEntry(entry)
		return true
	}
	return false
}

// Call f for every entry in the cache. Some sanity checks
// and eviction of expired keys are done as well.
func (c *Cache) Keys(f func(key string, val interface{})) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

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

func (c *Cache) insertFront(e *cacheEntry) {
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

func (c *Cache) evictEntry(e *cacheEntry) {
	if e.waitingForComputation != 0 {
		panic("cannot evict this entry as other goroutines need the value")
	}

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
