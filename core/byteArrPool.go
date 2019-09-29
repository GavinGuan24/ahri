package core

import (
	"sync"
)

type byteArrPool struct {
	*sync.Pool
	sizeLimit int
}

func (pool *byteArrPool) Get(args ...int) []byte {
	realSize := 0
	if len(args) > 0 && args[0] >= 0 {
		realSize = args[0]
	} else {
		realSize = pool.sizeLimit
	}
	arr := pool.Pool.Get().([]byte)
	if cap(arr) >= realSize {
		arr = arr[0:realSize]
	} else {
		pool.Pool.Put(arr)
		arr = make([]byte, realSize)
	}
	return arr
}

func (pool *byteArrPool) Put(arr []byte) {
	arr = arr[:cap(arr)]
	pool.Pool.Put(arr)
}

func NewByteArrPool(sizeLimit int) *byteArrPool {
	pool := &byteArrPool{}
	pool.sizeLimit = sizeLimit
	pool.Pool = &sync.Pool{New: func() interface{} { return make([]byte, pool.sizeLimit) }}
	return pool
}
