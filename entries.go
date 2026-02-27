package lru

// entries is thread-safe entry for LRU
type entries[K comparable, V any] struct {
	key   K
	value V
}

func newEntries[K comparable, V any](key K, value V) entries[K, V] {
	return entries[K, V]{
		key:   key,
		value: value,
	}
}

func (e *entries[K, V]) get() (K, V) {
	return e.key, e.value
}

func (e *entries[K, V]) set(value V) V {
	prev := e.value
	e.value = value
	return prev
}
