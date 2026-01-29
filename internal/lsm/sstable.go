package lsm

import (
	"encoding/binary"
	"io"
	"os"
	"sort"
)

// SSTable (Sorted String Table) is an immutable on-disk file.
// Format:
// [KeyLen(4)][Key][ValLen(4)][Value]...
// We iterate the MemTable (sorted) and write to disk.
type SSTable struct {
	file *os.File
}

func NewSSTable(path string) (*SSTable, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return &SSTable{file: f}, nil
}

// FlushMemTable writes the MemTable to a new SSTable file.
func FlushMemTable(m *MemTable, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	// 1. Get all keys
	// Note: In real LSM, MemTable is already sorted (SkipList).
	// Since we are using a map, we must sort keys now.
	// This is O(N log N) but fine for MVP.
	m.mu.RLock()
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	m.mu.RUnlock()

	sort.Strings(keys)

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, k := range keys {
		val := m.data[k]
		if err := writeEntry(f, k, val); err != nil {
			return err
		}
	}
	return nil
}

func writeEntry(w io.Writer, key string, value []byte) error {
	kBytes := []byte(key)
	kLen := uint32(len(kBytes))
	vLen := uint32(len(value))

	if err := binary.Write(w, binary.LittleEndian, kLen); err != nil {
		return err
	}
	if _, err := w.Write(kBytes); err != nil {
		return err
	}
	if err := binary.Write(w, binary.LittleEndian, vLen); err != nil {
		return err
	}
	if _, err := w.Write(value); err != nil {
		return err
	}
	return nil
}
