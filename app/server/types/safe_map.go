package types

import "sync"

type SafeMap[V any] struct {
	items map[string]V
	mu    sync.RWMutex
}

func NewSafeMap[V any]() *SafeMap[V] {
	return &SafeMap[V]{items: make(map[string]V)}
}

func (sm *SafeMap[V]) Get(key string) V {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.items[key]
}

func (sm *SafeMap[V]) Set(key string, value V) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.items[key] = value
}

func (sm *SafeMap[V]) Delete(key string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.items, key)
}

func (sm *SafeMap[V]) Update(key string, fn func(V)) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if item, ok := sm.items[key]; ok {
		fn(item)
		sm.items[key] = item
	}
}

func (sm *SafeMap[V]) Items() map[string]V {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Return a snapshot copy so callers can't mutate internal state without locks.
	items := make(map[string]V, len(sm.items))
	for k, v := range sm.items {
		items[k] = v
	}
	return items
}

func (sm *SafeMap[V]) Keys() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	keys := make([]string, len(sm.items))
	i := 0
	for k := range sm.items {
		keys[i] = k
		i++
	}
	return keys
}

func (sm *SafeMap[V]) Len() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.items)
}
