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

	return &LSMStore{
		memTable: NewMemTable(),
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
