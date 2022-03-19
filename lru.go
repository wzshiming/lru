package lru

import (
	"sync"
	"sync/atomic"

	"github.com/wzshiming/lru/internal/container/list"
	syncmap "github.com/wzshiming/lru/internal/sync"
)

// LRU is a thread-safe fixed size LRU cache.
type LRU[K comparable, V any] struct {
	mut sync.Mutex

	size uint64 // maximum number of items in cache

	items syncmap.Map[K, *list.Element[entries[K, V]]] // map of items in cache

	linked  *linked[entries[K, V]] // linked list of items in cache
	evicted func(K, V)             // callback function when an item is evicted

	evictCh    chan struct{}                     // evict channel
	recentlyCh chan *list.Element[entries[K, V]] // recently used channel

	isClosed uint32
}

// NewLRU returns a new LRU of the given size.
func NewLRU[K comparable, V any](size int, evicted func(K, V)) *LRU[K, V] {
	l := &LRU[K, V]{
		linked:     newLinked[entries[K, V]](),
		size:       uint64(size),
		evicted:    evicted,
		evictCh:    make(chan struct{}, 1),
		recentlyCh: make(chan *list.Element[entries[K, V]], 128),
	}
	go l.channelRecently()
	go l.channelEvict()
	return l
}

func (l *LRU[K, V]) channelRecently() {
	for atomic.LoadUint32(&l.isClosed) == 0 {
		select {
		case item := <-l.recentlyCh:
			l.linked.MoveToBack(item)
		}
	}
}

func (l *LRU[K, V]) channelEvict() {
	for atomic.LoadUint32(&l.isClosed) == 0 {
		select {
		case <-l.evictCh:
			for l.Len() > l.Cap() {
				l.evict()
			}
		}
	}
}

func (l *LRU[K, V]) toRecently(item *list.Element[entries[K, V]]) {
	l.recentlyCh <- item
}

func (l *LRU[K, V]) tryEvict() {
	select {
	case l.evictCh <- struct{}{}:
	default:
	}
}

// Evict evicts the least recently used item from the cache.
func (l *LRU[K, V]) Evict() (key K, value V, evicted bool) {
	item := l.evict()
	if item == nil {
		return
	}

	key, value = item.Value.get()
	return key, value, true
}

func (l *LRU[K, V]) evict() *list.Element[entries[K, V]] {
	node := l.linked.Front()
	if node == nil {
		return nil
	}

	item := l.linked.Remove(node)
	key, value := item.get()
	l.items.Delete(key)
	if l.evicted != nil {
		l.evicted(key, value)
	}
	return node
}

// Resize resizes the cache to the specified size.
func (l *LRU[K, V]) Resize(size int) {
	atomic.StoreUint64(&l.size, uint64(size))
	l.tryEvict()
	return
}

// Len returns the length of the lru cache
func (l *LRU[K, V]) Len() int {
	return l.linked.Len()
}

// Cap returns the capacity of the lru cache
func (l *LRU[K, V]) Cap() int {
	return int(atomic.LoadUint64(&l.size))
}

// Put sets the value for the specified key.
func (l *LRU[K, V]) Put(key K, value V) (prev V, replaced bool) {
	item, ok := l.items.Load(key)
	// key exists in cache already so we update it
	if ok && item != nil {
		l.toRecently(item)
		prev = item.Value.set(value)
		return prev, true
	}

	l.mut.Lock()
	defer l.mut.Unlock()
	item, ok = l.items.Load(key)
	// re-check if key exists in cache after we acquire the lock
	if ok && item != nil {
		l.toRecently(item)
		prev = item.Value.set(value)
		return prev, true
	}

	// key doesn't exist in cache so we add it
	item = l.linked.PushBack(entries[K, V]{key: key, value: value})
	l.items.Store(key, item)

	l.tryEvict()
	return
}

// Get returns a value for key and mark it as most recently used.
func (l *LRU[K, V]) Get(key K) (value V, ok bool) {
	item, ok := l.items.Load(key)
	if !ok || item == nil {
		return
	}

	l.toRecently(item)
	_, value = item.Value.get()
	return value, true
}

// Contains returns true if the key exists.
func (l *LRU[K, V]) Contains(key K) bool {
	_, ok := l.items.Load(key)
	return ok
}

// Peek returns the value for key without marking it as most recently used.
func (l *LRU[K, V]) Peek(key K) (value V, ok bool) {
	item, ok := l.items.Load(key)
	if !ok || item == nil {
		return
	}

	_, value = item.Value.get()
	return value, true
}

// Delete a value for a key
func (l *LRU[K, V]) Delete(key K) (prev V, deleted bool) {
	item, ok := l.items.LoadAndDelete(key)
	if !ok || item == nil {
		return
	}

	l.linked.Remove(item)
	_, prev = item.Value.get()
	return prev, true
}

// ForEach iterates over the cache, calling f for each item.
func (l *LRU[K, V]) ForEach(iter func(key K, value V) bool) {
	l.linked.ForEach(func(item *list.Element[entries[K, V]]) bool {
		key, value := item.Value.get()
		return iter(key, value)
	})
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
	atomic.StoreUint32(&l.isClosed, 1)
}
