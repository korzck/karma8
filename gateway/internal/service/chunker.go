package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"mime/multipart"
	"time"

	"gateway/internal/models"
	"gateway/internal/repository"
	"gateway/internal/storage"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

const NUM_OF_CHUNKS = 6

type ChunkerService struct {
	repository     *repository.Repository
	storageManager *storage.StorageManager
}

func NewChunkerService(repository *repository.Repository, storageManager *storage.StorageManager) *ChunkerService {
	return &ChunkerService{
		repository:     repository,
		storageManager: storageManager,
	}
}

type chunkReader struct {
	reader        io.Reader
	contentLength int64
}

func (cr *chunkReader) Read(p []byte) (n int, err error) {
	return cr.reader.Read(p)
}

func (s *ChunkerService) InsertStream(ctx context.Context, file multipart.File, header *multipart.FileHeader) (string, error) {
	chunkSizes := getChunkSizes(header.Size, NUM_OF_CHUNKS)
	fileUUID := uuid.New().String()

	for i := int64(0); i < int64(len(chunkSizes)); i++ {
		limitedReader := io.LimitReader(file, chunkSizes[i])

		reader := &chunkReader{
			reader:        limitedReader,
			contentLength: chunkSizes[i],
		}

		md5Hash := md5.New()
		teeReader := io.TeeReader(reader, md5Hash)

		storageID := s.storageManager.GetStorageID(fileUUID, i)

		err := s.storageManager.UploadChunkStream(ctx, fileUUID, i, teeReader, chunkSizes[i])
		if err != nil {
			return "", errors.Wrap(err, "upload chunk to storage")
		}

		chunkHash := hex.EncodeToString(md5Hash.Sum(nil))

		err = s.repository.InsertChunk(ctx, fileUUID, i, chunkHash, models.ChunkStatusPending, NUM_OF_CHUNKS, storageID)
		if err != nil {
			return "", errors.Wrap(err, "insert chunk")
		}

		err = s.repository.UpdateChunkStatus(ctx, fileUUID, i, models.ChunkStatusSentToStorage, time.Now())
		if err != nil {
			return "", errors.Wrap(err, "update chunk status")
		}
	}

	return fileUUID, nil
}

func getChunkSizes(fileSize int64, numOfChunks int) []int64 {
	if numOfChunks <= 0 {
		return nil
	}

	chunkSizes := make([]int64, numOfChunks)
	baseSize := fileSize / int64(numOfChunks)
	remainder := fileSize % int64(numOfChunks)

	for i := range numOfChunks {
		chunkSizes[i] = baseSize
		if int64(i) < remainder {
			chunkSizes[i]++
		}
	}

	return chunkSizes
}

func (s *ChunkerService) SelectStream(ctx context.Context, fileUUID string, writer io.Writer) error {
	chunks, err := s.repository.GetChunksByUUID(ctx, fileUUID)
	if err != nil {
		return errors.Wrap(err, "get chunks from database")
	}

	if len(chunks) == 0 {
		return errors.New("no chunks found for file")
	}

	fmt.Printf("DEBUG: Found %d chunks for file %s\n", len(chunks), fileUUID)

	chunksIntegrity := make(map[int64]struct{})
	for i, chunk := range chunks {
		fmt.Printf("DEBUG: Chunk %d: index=%d, hash=%s, status=%s, storage_id=%d\n", i, chunk.ChunkIndex, chunk.ChunkHash, chunk.Status, chunk.StorageID)
		if chunk.Status != models.ChunkStatusSentToStorage.String() {
			return errors.New("chunk is not sent to storage")
		}

		chunksIntegrity[chunk.ChunkIndex] = struct{}{}
	}

	for i := range NUM_OF_CHUNKS {
		if _, ok := chunksIntegrity[int64(i)]; !ok {
			return errors.New("file integrity check failed")
		}
	}

	for _, chunk := range chunks {
		fmt.Printf("DEBUG: Downloading chunk %d from storage %d\n", chunk.ChunkIndex, chunk.StorageID)
		err := s.storageManager.DownloadChunkStream(ctx, fileUUID, chunk.ChunkIndex, writer)
		if err != nil {
			return errors.Wrapf(err, "download chunk stream %d", chunk.ChunkIndex)
		}
		fmt.Printf("DEBUG: Successfully downloaded chunk %d\n", chunk.ChunkIndex)
	}

	return nil
}
