package lrucache

type ComputeValue func() (value interface{}, ttl, size int)

type Cache interface {
	Get(key string, computeValue ComputeValue) interface{}
}

type cacheEntry struct {
	value interface{}
	expiration int
	size int
}

type lruCache struct {
	maxmemory int
	usedmemory int
	entries map[string]cacheEntry
}

func (c *lruCache) Get(key string, computeValue ComputeValue) interface{} {
	value, _, _ := computeValue()
	return value
}

func New(maxmemory int) Cache {
	return &lruCache{
		maxmemory: maxmemory,
		usedmemory: 0,
		entries: map[string]cacheEntry{},
	}
}



