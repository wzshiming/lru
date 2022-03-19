package lru

import (
	"sync"
)

// entries is thread-safe entry for LRU
type entries[K comparable, V any] struct {
	mut   sync.RWMutex
	key   K
	value V
}

func (e *entries[K, V]) get() (K, V) {
	e.mut.RLock()
	defer e.mut.RUnlock()
	return e.key, e.value
}

func (e *entries[K, V]) set(value V) V {
	e.mut.Lock()
	defer e.mut.Unlock()
	v := e.value
	e.value = value
	return v
}
