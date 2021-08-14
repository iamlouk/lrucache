# In-Memory LRU Cache for Golang Applications

This library can be embedded into your existing go applications
and play the role *Memcached* or *Redis* might play for others.
It is heavily inspired by [PHP Symfony's Cache Components](https://symfony.com/doc/current/components/cache/adapters/array_cache_adapter.html),
having a similar API. This library can not be used for persistance,
is not properly tested yet and a bit special in a few ways described
below (Especially with regards to the memory usage/`size`).

- Advantages:
    - Anything (`interface{}`) can be stored as value
    - As it lives in the application itself, no serialization or de-serialization is needed
    - As it lives in the application itself, no memory moving/networking is needed
- Disadvantages:
    - You have to provide a size estimate for every value
    - __This size estimate should not change (i.e. values should not mutate)__
    - The cache can only be accessed by one application

```go
// Go look at the godocs and ./cache_test.go for more documentation and examples

maxMemory := 1000
cache := lrucache.New(maxMemory)

bar = cache.Get("foo", func () (value interface{}, ttl time.Duration, size int) {
	return "bar", 10 * time.Second, len("bar")
}).(string)

// bar == "bar"

bar = cache.Get("foo", func () (value interface{}, ttl time.Duration, size int) {
	panic("will not be called")
}).(string)
```

The closure passed to `Get` will be called if the value asked for is not cached or
expired. It should return the following values:

- The value corresponding to that key and to be stored in the cache
- The time to live for that value (how long until it expires and needs to be recomputed)
- A size estimate

When `maxMemory` is reached, cache entries need to be evicted. Theoretically,
it would be possible to use reflection on every value placed in the cache
to get its exact size in bytes. This would be very expansive and slow though.
Also, size can change. Instead of this library calculating the size in bytes, you, the user,
have to provide a size for every value in whatever unit you like (as long as it is the same unit everywhere).

Suggestions on what to use as size: `len(str)` for strings, `len(slice) * size_of_slice_type`, etc.. It is possible
to use `1` as size for every entry, in that case at most `maxMemory` entries will be in the cache at the same time.

