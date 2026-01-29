package lsm

import (
	"encoding/binary"
	"os"
	"sync"
)

// WAL (Write Ahead Log) handles append-only persistence.
type WAL struct {
	mu   sync.Mutex
	file *os.File
	path string
}

func NewWAL(path string) (*WAL, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	return &WAL{
		file: f,
		path: path,
	}, nil
}

// Write appends a key-value pair to the log.
// Format: [KeyLen(4)][ValLen(4)][Key][Value]
func (w *WAL) Write(key string, value []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	keyBytes := []byte(key)
	kLen := int32(len(keyBytes))
	vLen := int32(len(value))

	// Pre-allocate buffer for metadata
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint32(buf[0:4], uint32(kLen))
	binary.LittleEndian.PutUint32(buf[4:8], uint32(vLen))

	// Write metadata
	if _, err := w.file.Write(buf); err != nil {
		return err
	}
	// Write Key
	if _, err := w.file.Write(keyBytes); err != nil {
		return err
	}
	// Write Value
	if _, err := w.file.Write(value); err != nil {
		return err
	}

	// Ensure data is on disk
	return w.file.Sync()
}

func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.file.Close()
}
