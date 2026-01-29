package lsm

import "encoding/binary"

// Iterator helper for Compaction
type SSTableIterator struct {
	sst *SSTable
}

func NewSSTableIterator(sst *SSTable) (*SSTableIterator, error) {
	_, err := sst.file.Seek(0, 0)
	return &SSTableIterator{sst: sst}, err
}

func (it *SSTableIterator) Next() (string, []byte, bool) {
	// Read KeyLen
	var kLen uint32
	err := binary.Read(it.sst.file, binary.LittleEndian, &kLen)
	if err != nil {
		return "", nil, false
	}

	// Read Key
	kBytes := make([]byte, kLen)
	if _, err := it.sst.file.Read(kBytes); err != nil {
		return "", nil, false
	}

	// Read ValLen
	var vLen uint32
	if err := binary.Read(it.sst.file, binary.LittleEndian, &vLen); err != nil {
		return "", nil, false
	}

	// Read Value
	vBytes := make([]byte, vLen)
	if _, err := it.sst.file.Read(vBytes); err != nil {
		return "", nil, false
	}

	return string(kBytes), vBytes, true
}
