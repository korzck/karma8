package service

import (
	"testing"
)

func TestCreateChunkSizes_ValidChunks(t *testing.T) {
	tests := []struct {
		totalSize int64
		n         int
	}{
		{600, 6},
		{601, 6},
		{0, 6},
		{1234, 6},
		{100, 1},
		{100, 2},
		{5, 6},
		{100500, 6},
	}

	for _, tt := range tests {
		chunkSizes := getChunkSizes(tt.totalSize, tt.n)
		if tt.n <= 0 {
			if chunkSizes != nil {
				t.Errorf("Expected nil for n=%d, got %v", tt.n, chunkSizes)
			}
			continue
		}

		if len(chunkSizes) != tt.n {
			t.Errorf("Expected %d chunks, got %d", tt.n, len(chunkSizes))
		}

		sum := int64(0)
		for i, sz := range chunkSizes {
			if sz < 0 {
				t.Errorf("Chunk size at index %d is negative: %d", i, sz)
			}

			sum += sz
		}

		if sum != tt.totalSize {
			t.Errorf("Sum of chunk sizes (%d) does not match totalSize (%d)", sum, tt.totalSize)
		}
	}
}
