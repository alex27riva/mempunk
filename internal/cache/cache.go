// Package cache provides a thread-safe, generic bounded LRU cache.
package cache

import (
	"container/list"
	"sync"
)

type entry[V any] struct {
	key string
	val V
}

// Cache is a thread-safe generic bounded LRU cache keyed by string.
// A capacity <= 0 disables caching: Get always misses, Put is a no-op.
type Cache[V any] struct {
	cap   int
	mu    sync.Mutex
	ll    *list.List
	items map[string]*list.Element
}

// New creates a Cache with the given capacity.
func New[V any](capacity int) *Cache[V] {
	return &Cache[V]{
		cap:   capacity,
		ll:    list.New(),
		items: make(map[string]*list.Element),
	}
}

// Get returns the cached value for key and true, or the zero value and false.
func (c *Cache[V]) Get(key string) (V, bool) {
	if c.cap <= 0 {
		var zero V
		return zero, false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		var zero V
		return zero, false
	}
	c.ll.MoveToFront(el)
	return el.Value.(*entry[V]).val, true
}

// Put stores val under key, evicting the least-recently-used entry if full.
func (c *Cache[V]) Put(key string, val V) {
	if c.cap <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.ll.MoveToFront(el)
		el.Value.(*entry[V]).val = val
		return
	}
	el := c.ll.PushFront(&entry[V]{key: key, val: val})
	c.items[key] = el
	if c.ll.Len() > c.cap {
		c.evict()
	}
}

// evict removes the least-recently-used entry. Must be called with mu held.
func (c *Cache[V]) evict() {
	el := c.ll.Back()
	if el == nil {
		return
	}
	c.ll.Remove(el)
	delete(c.items, el.Value.(*entry[V]).key)
}
