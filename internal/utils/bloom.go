package utils

import (
	"hash/fnv"
)

// BloomFilter is a probabilistic data structure.
// False positives are possible, false negatives are not.
type BloomFilter struct {
	bitset []bool
	size   uint32
}

func NewBloomFilter(size uint32) *BloomFilter {
	return &BloomFilter{
		bitset: make([]bool, size),
		size:   size,
	}
}

func (bf *BloomFilter) Add(key string) {
	idx1, idx2 := bf.hash(key)
	bf.bitset[idx1] = true
	bf.bitset[idx2] = true
}

func (bf *BloomFilter) MayContain(key string) bool {
	idx1, idx2 := bf.hash(key)
	return bf.bitset[idx1] && bf.bitset[idx2]
}

// hash returns two hash values for the key
// MVP: Uses FNV-1a with simple offset
func (bf *BloomFilter) hash(key string) (uint32, uint32) {
	h := fnv.New32a()
	h.Write([]byte(key))
	val1 := h.Sum32() % bf.size

	// Double hashing simulation
	val2 := (val1 + 7) % bf.size
	return val1, val2
}
