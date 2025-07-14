package storage

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"sync"
)

type StorageManager struct {
	clients    []*Client
	numStorage int
	mu         sync.RWMutex
}

func NewStorageManager(storageAddrs []string) (*StorageManager, error) {
	if len(storageAddrs) == 0 {
		return nil, fmt.Errorf("at least one storage address is required")
	}

	clients := make([]*Client, len(storageAddrs))
	for i, addr := range storageAddrs {
		client, err := NewClient(addr)
		if err != nil {
			return nil, fmt.Errorf("failed to create client for %s: %w", addr, err)
		}
		clients[i] = client
	}

	return &StorageManager{
		clients:    clients,
		numStorage: len(clients),
	}, nil
}

func (sm *StorageManager) Close() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var lastErr error
	for _, client := range sm.clients {
		if err := client.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

func (sm *StorageManager) GetStorageIDForChunk(fileUUID string, chunkIndex int64) int {
	hash := hashString(fmt.Sprintf("%s_%d", fileUUID, chunkIndex))
	return int(hash % uint32(sm.numStorage))
}

func (sm *StorageManager) GetClientForChunk(fileUUID string, chunkIndex int64) *Client {
	storageID := sm.GetStorageIDForChunk(fileUUID, chunkIndex)
	return sm.clients[storageID]
}

func (sm *StorageManager) GetRandomClient() *Client {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.clients[rand.Intn(sm.numStorage)]
}

func (sm *StorageManager) UploadChunkStream(ctx context.Context, fileUUID string, chunkIndex int64, reader io.Reader, contentLength int64) error {
	client := sm.GetClientForChunk(fileUUID, chunkIndex)
	return client.UploadChunkStream(ctx, fileUUID, chunkIndex, reader, contentLength)
}

func (sm *StorageManager) DownloadChunkStream(ctx context.Context, fileUUID string, chunkIndex int64, writer io.Writer) error {
	client := sm.GetClientForChunk(fileUUID, chunkIndex)
	return client.DownloadChunkStream(ctx, fileUUID, chunkIndex, writer)
}

func (sm *StorageManager) GetStorageID(fileUUID string, chunkIndex int64) int {
	return sm.GetStorageIDForChunk(fileUUID, chunkIndex) + 1 // Convert to 1-based indexing
}

func (sm *StorageManager) GetNumStorage() int {
	return sm.numStorage
}

func (sm *StorageManager) HealthCheck() map[int]bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	health := make(map[int]bool)
	for i := range sm.clients {
		health[i+1] = true // 1-based indexing
	}

	return health
}

func hashString(s string) uint32 {
	var hash uint32 = 5381
	for _, c := range s {
		hash = ((hash << 5) + hash) + uint32(c)
	}
	return hash
}
