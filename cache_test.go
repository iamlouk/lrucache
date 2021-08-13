package lrucache

import "testing"

func TestColdCache(t *testing.T) {
	cache := New(123)

	value := cache.Get("foo", func()(interface{}, int, int) {
		return "bar", 0, 0
	})

	if value != "bar" {
		t.Error("cache returned wrong value")
	}
}

