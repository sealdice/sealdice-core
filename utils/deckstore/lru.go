package deckstore

import (
	"container/list"
	"strings"
	"sync"
)

// lruEntry 是 LRU 链表中的一个元素。
type lruEntry[V any] struct {
	key   string
	value V
	size  int64
}

// byteLRU 是一个按总字节数逐出的最近最少使用缓存，线程安全。
// 值类型由调用方通过泛型参数决定；每条的字节大小由调用方在 put 时显式给出。
type byteLRU[V any] struct {
	mu       sync.Mutex
	maxBytes int64
	curBytes int64
	ll       *list.List // 头 = 最近使用
	idx      map[string]*list.Element
}

func newByteLRU[V any](maxBytes int64) *byteLRU[V] {
	return &byteLRU[V]{
		maxBytes: maxBytes,
		ll:       list.New(),
		idx:      make(map[string]*list.Element),
	}
}

func (c *byteLRU[V]) get(key string) (V, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.idx[key]
	if !ok {
		var zero V
		return zero, false
	}
	c.ll.MoveToFront(el)
	return el.Value.(*lruEntry[V]).value, true
}

func (c *byteLRU[V]) put(key string, value V, size int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.idx[key]; ok {
		e := el.Value.(*lruEntry[V])
		c.curBytes += size - e.size
		e.value = value
		e.size = size
		c.ll.MoveToFront(el)
	} else {
		c.idx[key] = c.ll.PushFront(&lruEntry[V]{key: key, value: value, size: size})
		c.curBytes += size
	}
	for c.curBytes > c.maxBytes {
		c.evictOldestLocked()
	}
}

func (c *byteLRU[V]) remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.removeLocked(key)
}

func (c *byteLRU[V]) removePrefix(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for key := range c.idx {
		if strings.HasPrefix(key, prefix) {
			c.removeLocked(key)
		}
	}
}

func (c *byteLRU[V]) removeLocked(key string) {
	el, ok := c.idx[key]
	if !ok {
		return
	}
	e := el.Value.(*lruEntry[V])
	c.ll.Remove(el)
	delete(c.idx, key)
	c.curBytes -= e.size
}

func (c *byteLRU[V]) evictOldestLocked() {
	el := c.ll.Back()
	if el == nil {
		return
	}
	e := el.Value.(*lruEntry[V])
	c.ll.Remove(el)
	delete(c.idx, e.key)
	c.curBytes -= e.size
}
