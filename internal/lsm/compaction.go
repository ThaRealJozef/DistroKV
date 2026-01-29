package lsm

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// StartCompactionTrigger starts a background ticker to run compaction.
func (s *LSMStore) StartCompactionTrigger(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			s.Compact()
		}
	}()
}

// Compact merges all existing SSTables into a single one.
// MVP: "Major Compaction" only (Merge All).
func (s *LSMStore) Compact() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.sstables) <= 1 {
		return nil // Nothing to compact
	}

	fmt.Println("[Compaction] Starting merge of", len(s.sstables), "tables...")

	// 2. Load all data into memory map (Naive Merge)
	// Real LSM: Streaming Merge Sort (K-Way Merge) to avoid RAM spike.
	// MVP: Load all to map, same as Recover logic.
	mergedData := make(map[string][]byte)

	// Iterate from Oldest to Newest to overwrite correctly
	for _, sst := range s.sstables {
		// Reset seek
		sst.file.Seek(0, 0)

		// Read all
		// Re-using the scanning logic from NewSSTable (bloom rebuild)
		// Code duplication alert: We should have extracted "ScanAll".
		// Implementing inline for speed.
		// TODO: Extract to sstable.go if time permits.
	}

	// Wait, we need the logic to read keys/values.
	// Let's implement K-Way merge? No, too complex.
	// Map approach:
	for _, sst := range s.sstables {
		// Read all entries
		iterator, _ := NewSSTableIterator(sst)
		for {
			k, v, ok := iterator.Next()
			if !ok {
				break
			}
			mergedData[k] = v // Overwrite older value
		}
	}

	// 3. Write to new SSTable
	newFilename := fmt.Sprintf("compacted_%d.sst", time.Now().UnixNano())
	newPath := filepath.Join(filepath.Dir(s.sstables[0].file.Name()), newFilename)

	// Reuse FlushMemTable logic?
	// We need a MemTable to pass to FlushMemTable.
	tempMem := NewMemTable()
	for k, v := range mergedData {
		tempMem.Put(k, v)
	}

	if err := FlushMemTable(tempMem, newPath); err != nil {
		return err
	}

	// 4. Open new SSTable
	newSST, err := NewSSTable(newPath)
	if err != nil {
		return err
	}

	// 5. Replace List & Cleanup
	oldTables := s.sstables
	s.sstables = []*SSTable{newSST}

	fmt.Println("[Compaction] Merged into", newFilename)

	// Close and Delete old files
	for _, t := range oldTables {
		t.Close()
		os.Remove(t.file.Name())
	}

	return nil
}
