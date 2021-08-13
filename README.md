# In-Memory LRU Cache for Golang Applications

__*TODO: Everything*__

I want this to become a library that can be embedded into
a go application like a web server where it can do what
*Memcached* or *Redis* might do elsewhere. __It assumes that
values in the cache do not change in size (i.e. do not mutate)__.

- Advantages:
    - As it lives in the application itself, no serialization or de-serialization is needed
    - As it lives in the application itself, no networking (not even on the same host) is needed to access the data
- Disadvantages:
    - You have to provide a size estimate for every value, and that estimate should not change
    - It will probably be shitty and simply for me to toy around with go

```go
// I have implemented nothing yet, but this is what I hope
// the API will look like:

maxMemory := 1000
cache := lrucache.New(maxMemory)

bar = cache.Get("foo", func () (value interface{}, ttl int, size int) {
	return "bar", 10, len("bar")
}).(string)

```

My basic idea for this is that the cache should take strings
as keys and store any go value (`interface{}`). The closure passed
to `Get` will be called if the value asked for is not cached, and should
return the following values:

- The go value corresponding to that key and to be stored in the cache
- The time to live for that value (in seconds)
- A size estimate (more on that later)

If the *ttl* has passed, even if a value is in the cache, the closure passed
to `Get` will be called to recompute a new value.

When `maxMemory` is reached, cache items need to be evicted. Theoretically,
it would be possible to use reflection on every value placed in the cache
to get its exact size in bytes. This would be very expansive/slow though, and i am
scared of reflection since having used it in java. Also, size can change. Instead
of this library calculating the size in bytes, you, the user, have to provide a size
for every value in whatever unit you like (as long as it is the same unit everywhere).

