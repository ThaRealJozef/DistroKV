package lsm

import (
	"sync"
)

// MemTable is an in-memory key-value store.
// In a full LSM, this would be a SkipList or Red-Black Tree.
// For MVP, we use a Go map with a RWMutex.
type MemTable struct {
	mu   sync.RWMutex
	data map[string][]byte
	size int // Estimate of size in bytes
}

func NewMemTable() *MemTable {
	return &MemTable{
		data: make(map[string][]byte),
	}
}

func (m *MemTable) Put(key string, value []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Basic size calc: keysize + valuesize
	// (Note: this is rough approximation)
	if oldStub, ok := m.data[key]; ok {
		m.size -= (len(key) + len(oldStub))
	}
	m.data[key] = value
	m.size += len(key) + len(value)
}

func (m *MemTable) Get(key string) ([]byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.data[key]
	return val, ok
}

func (m *MemTable) Delete(key string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if val, ok := m.data[key]; ok {
		m.size -= (len(key) + len(val))
		delete(m.data, key)
	}
	// In LSM, a delete is actually a "Tombstone" Put.
	// For this in-memory component, we actually delete from the map,
	// but the Writer logic will likely need to handle tombstones
	// when hitting disk.
}

func (m *MemTable) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.size
}

// KeyCount returns the number of keys.
func (m *MemTable) KeyCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data)
}

func (m *MemTable) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string][]byte)
	m.size = 0
}
