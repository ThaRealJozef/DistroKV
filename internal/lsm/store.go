package lsm

import (
	"path/filepath"
)

type LSMStore struct {
	memTable *MemTable
	wal      *WAL
	// sstables []*SSTable // For later
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

	return &LSMStore{
		memTable: memTable,
		wal:      wal,
	}, nil
}

func (s *LSMStore) Put(key string, value []byte) error {
	// 1. Write to WAL
	if err := s.wal.Write(key, value); err != nil {
		return err
	}
	// 2. Update MemTable
	s.memTable.Put(key, value)
	return nil
}

func (s *LSMStore) Get(key string) ([]byte, bool) {
	// 1. Check MemTable
	return s.memTable.Get(key)
	// 2. Check SSTables (TODO)
}

func (s *LSMStore) Close() error {
	return s.wal.Close()
}
