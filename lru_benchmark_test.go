package lru

import (
	"testing"
)

func BenchmarkLRU(b *testing.B) {
	lru := NewLRU[int, int](64, nil)
	for i := 0; i < b.N; i++ {
		lru.Put(i, i)
	}
	for i := 0; i < b.N; i++ {
		lru.Get(i)
	}
}

func BenchmarkParallelLRU(b *testing.B) {
	lru := NewLRU[int, int](64, nil)
	b.RunParallel(func(pb *testing.PB) {
		for i := 0; pb.Next(); i++ {
			lru.Put(i, i)
		}
	})

	b.RunParallel(func(pb *testing.PB) {
		for i := 0; pb.Next(); i++ {
			lru.Get(i)
		}
	})
}
