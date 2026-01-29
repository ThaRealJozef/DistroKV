package lsm

// StorageEngine defines the interface for the LSM Tree storage.
// It handles persistent Key-Value storage with high write throughput.
type StorageEngine interface {
	// Put stores a key-value pair.
	Put(key string, value []byte) error

	// Get retrieves a value by key. Returns nil if not found (or specific error).
	Get(key string) ([]byte, error)

	// Delete removes a key.
	Delete(key string) error

	// Close flushes any pending writes and closes file handles.
	Close() error
}
