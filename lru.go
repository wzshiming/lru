package lru

import (
	"sync"
	"sync/atomic"

	"github.com/wzshiming/lru/internal/container/list"
)

// LRU is a thread-safe fixed size LRU cache.
type LRU[K comparable, V any] struct {
	size uint64 // maximum number of items in cache

	items    map[K]*list.Element[entries[K, V]] // map of items in cache
	itemsMut sync.RWMutex

	linked    *list.List[entries[K, V]] // linked list of items in cache
	linkedMut sync.RWMutex

	evictCh chan struct{}
	evicted func(K, V) // callback function when an item is evicted
}

// NewLRU returns a new LRU of the given size.
func NewLRU[K comparable, V any](size int, evicted func(K, V)) *LRU[K, V] {
	l := &LRU[K, V]{
		linked:  list.New[entries[K, V]](),
		size:    uint64(size),
		items:   map[K]*list.Element[entries[K, V]]{},
		evictCh: make(chan struct{}, 1),
		evicted: evicted,
	}

	go l.channelEvict()

	return l
}

func (l *LRU[K, V]) channelEvict() {
	for range l.evictCh {
		for l.Len() > l.Cap() {
			_, _, _ = l.Evict()
		}
	}
}

func (l *LRU[K, V]) toRecently(item *list.Element[entries[K, V]]) {
	l.linkedMut.Lock()
	defer l.linkedMut.Unlock()
	l.linked.MoveToBack(item)
}

func (l *LRU[K, V]) tryEvict() {
	select {
	case l.evictCh <- struct{}{}:
	default:
	}
}

// Evict evicts the least recently used item from the cache.
func (l *LRU[K, V]) Evict() (key K, value V, evicted bool) {
	l.linkedMut.Lock()
	node := l.linked.Front()
	if node == nil {
		l.linkedMut.Unlock()
		return
	}

	item := l.linked.Remove(node)
	l.linkedMut.Unlock()

	l.itemsMut.Lock()
	key, value = item.get()
	delete(l.items, key)
	l.itemsMut.Unlock()

	if l.evicted != nil {
		l.evicted(key, value)
	}
	return key, value, true
}

// Resize resizes the cache to the specified size.
func (l *LRU[K, V]) Resize(size int) {
	old := atomic.SwapUint64(&l.size, uint64(size))
	if old > uint64(size) {
		l.tryEvict()
	}
}

// Len returns the length of the lru cache
func (l *LRU[K, V]) Len() int {
	l.linkedMut.RLock()
	defer l.linkedMut.RUnlock()
	return l.linked.Len()
}

// Cap returns the capacity of the lru cache
func (l *LRU[K, V]) Cap() int {
	return int(atomic.LoadUint64(&l.size))
}

// Put sets the value for the specified key.
func (l *LRU[K, V]) Put(key K, value V) (prev V, replaced bool) {
	l.itemsMut.Lock()
	defer l.itemsMut.Unlock()
	item, ok := l.items[key]
	// key exists in cache already so we update it
	if ok && item != nil {
		l.toRecently(item)
		prev = item.Value.set(value)
		return prev, true
	}

	// key doesn't exist in cache so we add it
	l.linkedMut.Lock()
	item = l.linked.PushBack(newEntries(key, value))
	l.linkedMut.Unlock()

	l.items[key] = item

	l.tryEvict()
	return
}

// Get returns a value for key and mark it as most recently used.
func (l *LRU[K, V]) Get(key K) (value V, ok bool) {
	l.itemsMut.RLock()
	defer l.itemsMut.RUnlock()

	item, ok := l.items[key]
	if !ok || item == nil {
		return
	}

	l.toRecently(item)
	_, value = item.Value.get()
	return value, true
}

// Contains returns true if the key exists.
func (l *LRU[K, V]) Contains(key K) bool {
	l.itemsMut.RLock()
	defer l.itemsMut.RUnlock()

	_, ok := l.items[key]
	return ok
}

// Peek returns the value for key without marking it as most recently used.
func (l *LRU[K, V]) Peek(key K) (value V, ok bool) {
	l.itemsMut.RLock()
	defer l.itemsMut.RUnlock()

	item, ok := l.items[key]
	if !ok || item == nil {
		return
	}

	_, value = item.Value.get()
	return value, true
}

// Delete a value for a key
func (l *LRU[K, V]) Delete(key K) (prev V, deleted bool) {
	l.itemsMut.RLock()
	defer l.itemsMut.RUnlock()

	item, ok := l.items[key]
	if !ok || item == nil {
		return
	}

	delete(l.items, key)
	_, prev = item.Value.get()

	l.linkedMut.Lock()
	defer l.linkedMut.Unlock()
	l.linked.Remove(item)
	return prev, true
}

// ForEach iterates over the cache, calling f for each item.
func (l *LRU[K, V]) ForEach(iter func(key K, value V) bool) {
	l.linkedMut.RLock()
	defer l.linkedMut.RUnlock()

	for element := l.linked.Back(); element != nil; element = element.Prev() {
		l.itemsMut.RLock()
		key, value := element.Value.get()
		l.itemsMut.RUnlock()
		if !iter(key, value) {
			break
		}
	}
}

// Keys returns a slice of the keys in the cache.
func (l *LRU[K, V]) Keys() []K {
	keys := make([]K, 0, l.Len())
	l.ForEach(func(key K, value V) bool {
		keys = append(keys, key)
		return true
	})
	return keys
}

// Values returns a slice of the values in the cache.
func (l *LRU[K, V]) Values() []V {
	values := make([]V, 0, l.Len())
	l.ForEach(func(key K, value V) bool {
		values = append(values, value)
		return true
	})
	return values
}

// Close closes the cache.
func (l *LRU[K, V]) Close() {
	close(l.evictCh)
	l.linkedMut.Lock()
	l.linked.Init()
	l.linkedMut.Unlock()

	l.itemsMut.Lock()
	l.items = map[K]*list.Element[entries[K, V]]{}
	l.itemsMut.Unlock()
}
