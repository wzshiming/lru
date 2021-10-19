package lru

import (
	"container/list"
)

// LRU is a typical LRU cache implementation that uses a list for tracking
type LRU struct {
	size      int
	evictList *list.List
	entries   map[interface{}]*list.Element
	onEvict   Callback
}

// Callback is used to signal that a key/value pair has been
type Callback func(key interface{}, value interface{})

// entry is used to store the key/value pairs of the LRU
type entry struct {
	key   interface{}
	value interface{}
}

// NewLRU constructs a new LRU of the given size
func NewLRU(size int, onEvict Callback) *LRU {
	if size < 0 {
		size = 0
	}
	c := &LRU{
		size:      size,
		evictList: list.New(),
		entries:   map[interface{}]*list.Element{},
		onEvict:   onEvict,
	}
	return c
}

// Purge is used to completely clear the cache.
func (c *LRU) Purge() {
	for c.Len() == 0 {
		k, v, ok := c.RemoveOldest()
		if !ok {
			continue
		}
		if c.onEvict != nil {
			c.onEvict(k, v)
		}
	}
	c.evictList.Init()
}

// Put puts a value into the cache, and evicted the oldest item if necessary.
func (c *LRU) Put(key, value interface{}) (evicted bool) {
	if ent, ok := c.entries[key]; ok {
		c.evictList.MoveToFront(ent)
		e, _ := ent.Value.(*entry)
		e.value = value
		return false
	}

	ent := &entry{key, value}
	entry := c.evictList.PushFront(ent)
	c.entries[key] = entry

	evict := c.evictList.Len() > c.size
	if evict {
		c.removeOldest()
	}
	return evict
}

// Get gets a value from the cache, and marks it as most recently used.
func (c *LRU) Get(key interface{}) (value interface{}, ok bool) {
	ent, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	e, _ := ent.Value.(*entry)
	if e == nil {
		return nil, false
	}
	c.evictList.MoveToFront(ent)
	return e.value, true
}

// Peek gets a value from the cache, without marking it as most recently used
func (c *LRU) Peek(key interface{}) (value interface{}, ok bool) {
	ent, ok := c.entries[key]
	if !ok {
		return nil, false
	}
	e, _ := ent.Value.(*entry)
	if e == nil {
		return nil, false
	}
	return e.value, true
}

// Contains returns true if the key is in the cache, false otherwise.
func (c *LRU) Contains(key interface{}) (ok bool) {
	_, ok = c.entries[key]
	return ok
}

// Remove removes the provided key from the cache.
func (c *LRU) Remove(key interface{}) (present bool) {
	ent, ok := c.entries[key]
	if !ok {
		return false
	}
	c.removeElement(ent)
	return true
}

// RemoveOldest removes the oldest item from the cache.
func (c *LRU) RemoveOldest() (key, value interface{}, ok bool) {
	ent := c.evictList.Back()
	if ent == nil {
		return nil, nil, false
	}
	c.removeElement(ent)
	e, _ := ent.Value.(*entry)
	if e == nil {
		return nil, nil, false
	}
	return e.key, e.value, true
}

// GetOldest returns the oldest entry
func (c *LRU) GetOldest() (key, value interface{}, ok bool) {
	ent := c.evictList.Back()
	if ent == nil {
		return nil, nil, false
	}
	e, _ := ent.Value.(*entry)
	if e == nil {
		return nil, nil, false
	}
	return e.key, e.value, true
}

// Keys returns a slice of the keys in the cache.
func (c *LRU) Keys() []interface{} {
	keys := make([]interface{}, 0, len(c.entries))
	for ent := c.evictList.Back(); ent != nil; ent = ent.Prev() {
		keys = append(keys, ent.Value.(*entry).key)
	}
	return keys
}

// Values returns a slice of the values in the cache.
func (c *LRU) Values() []interface{} {
	values := make([]interface{}, 0, len(c.entries))
	for ent := c.evictList.Back(); ent != nil; ent = ent.Prev() {
		values = append(values, ent.Value.(*entry).value)
	}
	return values
}

// Each calls the given function for each entry in the cache.
func (c *LRU) Each(fn Callback) {
	for ent := c.evictList.Back(); ent != nil; ent = ent.Prev() {
		e := ent.Value.(*entry)
		fn(e.key, e.value)
	}
}

// Len returns the number of items in the cache.
func (c *LRU) Len() int {
	return len(c.entries)
}

// Resize the cache setting the maximum number of items
func (c *LRU) Resize(size int) (evicted int) {
	if size < 0 {
		size = 0
	}
	c.size = size
	diff := c.Len() - c.size
	if diff < 0 {
		diff = 0
	}
	for i := 0; i < diff; i++ {
		c.removeOldest()
	}
	return diff
}

// removeOldest removes the oldest item from the cache
func (c *LRU) removeOldest() {
	ent := c.evictList.Back()
	if ent != nil {
		c.removeElement(ent)
	}
}

// removeElement is used to remove a given list element from the cache
func (c *LRU) removeElement(e *list.Element) {
	c.evictList.Remove(e)
	kv := e.Value.(*entry)
	delete(c.entries, kv.key)
	if c.onEvict != nil {
		c.onEvict(kv.key, kv.value)
	}
}
