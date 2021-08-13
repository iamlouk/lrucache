package lrucache

type ComputeValue func() (value interface{}, ttl, size int)

type Cache interface {
	Get(key string, computeValue ComputeValue) interface{}
}

func New(maxmemory int) Cache {
	panic("unimplemented")
}



