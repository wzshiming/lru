package lru

import (
	"fmt"
	"reflect"
	"testing"
	"time"
)

func testStep(t *testing.T, lru *LRU[string, string], c, l int, keys []string, values []string) {
	time.Sleep(time.Millisecond)
	if l != lru.Len() {
		t.Errorf("got len = %d, want %d", lru.Len(), l)
	}
	if c != lru.Cap() {
		t.Errorf("got cap = %d, want %d", lru.Cap(), c)
	}
	if k := lru.Keys(); !reflect.DeepEqual(keys, k) {
		t.Errorf("got keys = %v, want %v", k, keys)
	}
	if v := lru.Values(); !reflect.DeepEqual(values, v) {
		t.Errorf("got values = %v, want %v", v, values)
	}
	for i, k := range keys {
		if !lru.Contains(k) {
			t.Errorf("key %q not found", k)
		}
		if v, ok := lru.Peek(k); !ok {
			t.Errorf("key %q not found", k)
		} else if v != values[i] {
			t.Errorf("key %q = %q, want %q", k, v, values[i])
		}
	}
}

func TestLRU(t *testing.T) {
	lru := NewLRU[string, string](4, func(k string, v string) {
		t.Logf("evict key: %s, value: %s", k, v)
	})
	defer lru.Close()
	testStep(t, lru, 4, 0,
		[]string{},
		[]string{},
	)

	_, _, evicted := lru.Evict()
	if evicted {
		t.Errorf("evicted = %v, want %v", evicted, false)
	}

	for i := 0; i < 5; i++ {
		lru.Put(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i))
	}
	testStep(t, lru, 4, 4,
		[]string{"key4", "key3", "key2", "key1"},
		[]string{"value4", "value3", "value2", "value1"},
	)

	lru.Put("key2", "new-value2")
	testStep(t, lru, 4, 4,
		[]string{"key2", "key4", "key3", "key1"},
		[]string{"new-value2", "value4", "value3", "value1"},
	)

	if !lru.Contains("key4") {
		t.Errorf("key4 should be in lru")
	}
	if lru.Contains("key5") {
		t.Errorf("key5 should not be in lru")
	}
	_, okpkey5 := lru.Peek("key5")
	if okpkey5 {
		t.Errorf("key5 should not be in lru")
	}

	_, okpkey5 = lru.Get("key5")
	if okpkey5 {
		t.Errorf("key5 should not be in lru")
	}

	_, okpkey5 = lru.Delete("key5")
	if okpkey5 {
		t.Errorf("key5 should not be in lru")
	}

	pkey3, okpkey3 := lru.Peek("key3")
	testStep(t, lru, 4, 4,
		[]string{"key2", "key4", "key3", "key1"},
		[]string{"new-value2", "value4", "value3", "value1"},
	)
	if !okpkey3 {
		t.Errorf("key3 should be in lru")
	}
	if pkey3 != "value3" {
		t.Errorf("got peek value = %s, want %s", pkey3, "value3")
	}

	gkey3, okgkey3 := lru.Get("key3")
	testStep(t, lru, 4, 4,
		[]string{"key3", "key2", "key4", "key1"},
		[]string{"value3", "new-value2", "value4", "value1"},
	)
	if !okgkey3 {
		t.Errorf("key3 should be in lru")
	}
	if gkey3 != "value3" {
		t.Errorf("got get value = %s, want %s", gkey3, "value3")
	}

	lru.Delete("key3")
	testStep(t, lru, 4, 3,
		[]string{"key2", "key4", "key1"},
		[]string{"new-value2", "value4", "value1"},
	)

	evictK, evictV, evicted := lru.Evict()
	if !evicted {
		t.Errorf("evict should have evicted")
	}
	if evictK != "key1" {
		t.Errorf("got evict key = %s, want %s", evictK, "key1")
	}
	if evictV != "value1" {
		t.Errorf("got evict value = %s, want %s", evictV, "value1")
	}
	testStep(t, lru, 4, 2,
		[]string{"key2", "key4"},
		[]string{"new-value2", "value4"},
	)

	for i := 0; i < 2; i++ {
		lru.Put(fmt.Sprintf("again-key%d", i), fmt.Sprintf("again-value%d", i))
	}
	testStep(t, lru, 4, 4,
		[]string{"again-key1", "again-key0", "key2", "key4"},
		[]string{"again-value1", "again-value0", "new-value2", "value4"},
	)

	lru.Resize(2)
	testStep(t, lru, 2, 2,
		[]string{"again-key1", "again-key0"},
		[]string{"again-value1", "again-value0"},
	)

}
