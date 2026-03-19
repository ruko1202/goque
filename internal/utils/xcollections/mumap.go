// Package xcollections provides thread-safe collection types.
package xcollections

import "sync"

// MUMap is a thread-safe map implementation using RWMutex for concurrent access.
type MUMap[K comparable, V any] struct {
	mu    *sync.RWMutex
	store map[K]V
}

// NewMUMap creates and initializes a new thread-safe map.
func NewMUMap[K comparable, V any]() *MUMap[K, V] {
	return &MUMap[K, V]{
		mu:    new(sync.RWMutex),
		store: make(map[K]V),
	}
}

// Add inserts or updates a key-value pair in the map.
func (m *MUMap[K, V]) Add(key K, value V) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.store[key] = value
}

// Get retrieves a value by key. Returns the value and a boolean indicating whether the key exists.
func (m *MUMap[K, V]) Get(key K) (V, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	value, ok := m.store[key]

	return value, ok
}
