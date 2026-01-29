package lsm

import (
	"distrokv/internal/utils"
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
	file  *os.File
	bloom *utils.BloomFilter
}

func NewSSTable(path string) (*SSTable, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	// Rebuild Bloom Filter (Trade-off: Slow Open, Fast Get)
	// Scan the whole file to populate filter.
	// In production, we'd read a "meta block" at the end of file.
	bf := utils.NewBloomFilter(1000) // Fixed size for MVP or dynamic?

	// Reset to 0
	f.Seek(0, 0)
	for {
		var kLen uint32
		if err := binary.Read(f, binary.LittleEndian, &kLen); err != nil {
			break
		}
		kBytes := make([]byte, kLen)
		f.Read(kBytes)
		bf.Add(string(kBytes))

		var vLen uint32
		binary.Read(f, binary.LittleEndian, &vLen)
		f.Seek(int64(vLen), 1) // Skip value
	}

	return &SSTable{file: f, bloom: bf}, nil
}

// Get searches for a key in the SSTable file.
// Returns value and true if found, nil and false otherwise.
// Note: MVP uses linear scan. Real LSM uses Sparse Index + Binary Search.
func (t *SSTable) Get(key string) ([]byte, bool) {
	// Optimization 1: Bloom Filter check
	if t.bloom != nil && !t.bloom.MayContain(key) {
		return nil, false
	}

	// Reset file pointer to beginning
	_, err := t.file.Seek(0, 0)
	if err != nil {
		return nil, false
	}

	for {
		// Read KeyLen
		var kLen uint32
		err := binary.Read(t.file, binary.LittleEndian, &kLen)
		if err != nil {
			break // EOF or error
		}

		// Read Key
		kBytes := make([]byte, kLen)
		if _, err := t.file.Read(kBytes); err != nil {
			break
		}
		currKey := string(kBytes)

		// Read ValLen
		var vLen uint32
		if err := binary.Read(t.file, binary.LittleEndian, &vLen); err != nil {
			break
		}

		// Read Value
		vBytes := make([]byte, vLen)
		if _, err := t.file.Read(vBytes); err != nil {
			break
		}

		// Compare Key
		if currKey == key {
			return vBytes, true
		}

		// Optimization: Since SSTable is sorted, if currKey > key, we can stop early (not found)
		if currKey > key {
			return nil, false
		}
	}
	return nil, false
}

func (t *SSTable) Close() error {
	return t.file.Close()
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

	// Create and populate Bloom Filter
	bf := utils.NewBloomFilter(uint32(len(keys) * 10)) // ~10 bits per key
	for _, k := range keys {
		bf.Add(k)
	}

	// TODO: Save Bloom Filter to disk (header or separate file)
	// For MVP: We will re-generate it on load? No, that requires reading the whole file!
	// We MUST save it.
	// Let's write it at the END of the file or beginning.
	// Simple: Write serialization of bitset at the start? No, size unknown.
	// Let's just create it in memory for FlushMemTable return (if we kept it open), but NewSSTable reads from disk.

	// Decision: Bloom Filter persistence is tricky for 5 min task.
	// Alternative: Just keep it in memory for "Current Run" and re-build on startup?
	// Re-building on startup requires scanning every SSTable (slow startup, fast runs).
	// Let's do that for MVP "Performance Trade-off".
	// "We accepted slower startup to keep file format simple".

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
