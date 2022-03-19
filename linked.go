package lru

import (
	"sync"

	"github.com/wzshiming/lru/internal/container/list"
)

// linked is thread-safe list for LRU
type linked[T any] struct {
	mut  sync.RWMutex
	list *list.List[T]
}

func newLinked[T any]() *linked[T] {
	return &linked[T]{
		list: list.New[T](),
	}
}

func (l *linked[T]) Len() int {
	l.mut.RLock()
	defer l.mut.RUnlock()
	return l.list.Len()
}

func (l *linked[T]) Front() *list.Element[T] {
	l.mut.RLock()
	defer l.mut.RUnlock()
	return l.list.Front()
}

func (l *linked[T]) Back() *list.Element[T] {
	l.mut.RLock()
	defer l.mut.RUnlock()
	return l.list.Back()
}

func (l *linked[T]) PushBack(v T) *list.Element[T] {
	l.mut.Lock()
	defer l.mut.Unlock()
	return l.list.PushBack(v)
}

func (l *linked[T]) Remove(e *list.Element[T]) T {
	l.mut.Lock()
	defer l.mut.Unlock()
	return l.list.Remove(e)
}

func (l *linked[T]) MoveToBack(e *list.Element[T]) {
	l.mut.Lock()
	defer l.mut.Unlock()
	l.list.MoveToBack(e)
}

func (l *linked[T]) ForEach(fun func(e *list.Element[T]) bool) {
	l.mut.RLock()
	defer l.mut.RUnlock()
	for element := l.list.Back(); element != nil; element = element.Prev() {
		fun(element)
	}
}
