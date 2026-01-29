package lsm

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type LSMStore struct {
	mu       sync.RWMutex
	memTable *MemTable
	wal      *WAL
	sstables []*SSTable // For later
}

func NewLSMStore(dir string) (*LSMStore, error) {
	// Create directory if not exists
	// MVP: Assume exists or simple path

	walPath := filepath.Join(dir, "wal.log")
	wal, err := NewWAL(walPath)
	if err != nil {
		return nil, err
	}

	// Phase 4: Crash Recovery
	// Load existing data from WAL into MemTable
	recoveredData, err := wal.Recover()
	if err != nil {
		return nil, err
	}

	// Create MemTable with recovered data
	memTable := NewMemTable()
	// Hack: MemTable internal data is private, should use Put.
	// But Put writes to WAL again! We just want to hydrate memory.
	// Let's add a "Load" method to MemTable or just loop Put (inefficient but safe if we disable WAL write temporarily)
	// Better: MemTable NewMemTableWithData... or just expose Put without side effects?
	// MemTable is pure memory, so Put has no side effects.
	// Side effects are in Store.Put (which calls WAL.Write).
	// So calling memTable.Put is safe!
	for k, v := range recoveredData {
		memTable.Put(k, v)
	}

	// Load existing SSTables
	matches, err := filepath.Glob(filepath.Join(dir, "*.sst"))
	if err != nil {
		return nil, err
	}

	var sstables []*SSTable
	for _, match := range matches {
		sst, err := NewSSTable(match)
		if err != nil {
			// If one fails, maybe log and continue? For now, fail hard.
			return nil, err
		}
		sstables = append(sstables, sst)
	}

	return &LSMStore{
		memTable: memTable,
		wal:      wal,
		sstables: sstables,
	}, nil
}

func (s *LSMStore) Put(key string, value []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Write to WAL
	if err := s.wal.Write(key, value); err != nil {
		return err
	}
	// 2. Update MemTable
	s.memTable.Put(key, value)

	// 3. Check Flush Threshold
	// MVP: Flush every 100 keys for demo purposes
	if s.memTable.KeyCount() > 100 {
		if err := s.Flush(); err != nil {
			// Log error but don't fail the Put?
			fmt.Printf("Flush failed: %v\n", err)
		}
	}

	return nil
}

func (s *LSMStore) Get(key string) ([]byte, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 1. Check MemTable
	if val, found := s.memTable.Get(key); found {
		return val, true
	}

	// 2. Check SSTables
	// Iterate in reverse order (newest first)
	for i := len(s.sstables) - 1; i >= 0; i-- {
		if val, found := s.sstables[i].Get(key); found {
			return val, true
		}
	}

	return nil, false
}

// Flush persists the current MemTable to disk as an SSTable.
func (s *LSMStore) Flush() error {
	// s.mu.Lock() // Already called by Put if we call from there.
	// But if called externally, we need lock.
	// Put calls Flush with lock held? No, Put holds lock then calls Flush? A deadlock!
	// We should split "Flush" logic (internal) vs "TriggerFlush" (external).
	// Let's assume Flush is internal utility called with lock held.
	// Wait, if Put calls Flush, Put holds s.mu.Lock(). Flush needs s.mu.Lock()?
	// Go deadlocks on re-entrant locks.
	// So Flush must NOT lock, or we use an internal `flushLocked`.

	// Implementation below assumes Caller holds Lock.

	if s.memTable.KeyCount() == 0 {
		return nil
	}

	// 1. Write MemTable to SSTable
	filename := fmt.Sprintf("data_%d.sst", time.Now().UnixNano())
	path := filepath.Join(filepath.Dir(s.wal.path), filename)

	if err := FlushMemTable(s.memTable, path); err != nil {
		return err
	}

	// 2. Open new SSTable
	sst, err := NewSSTable(path)
	if err != nil {
		return err
	}
	s.sstables = append(s.sstables, sst)

	// 3. Clear MemTable
	s.memTable = NewMemTable()

	// 4. Truncate WAL (Since data is now safe in SST)
	// MVP: Check if WAL supports truncate/reset.
	// We need to Close and Reopen/Create WAL.
	if err := s.wal.Close(); err != nil {
		return err
	}

	// Truncate file
	if err := os.Truncate(s.wal.path, 0); err != nil {
		return err
	}

	// Reopen
	wal, err := NewWAL(s.wal.path)
	if err != nil {
		return err
	}
	s.wal = wal

	fmt.Println("[LSM] Flushed MemTable to", filename)
	return nil
}

func (s *LSMStore) Close() error {
	return s.wal.Close()
}
