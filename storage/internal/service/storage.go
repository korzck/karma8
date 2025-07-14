package service

import (
	"context"
	"io"

	"storage/internal/repository"

	"github.com/pkg/errors"
)

type StorageService struct {
	repository *repository.Repository
}

func NewStorageService(repository *repository.Repository) *StorageService {
	return &StorageService{
		repository: repository,
	}
}

func (s *StorageService) GetRepository() *repository.Repository {
	return s.repository
}

func (s *StorageService) UploadChunkStream(ctx context.Context, fileUUID string, chunkIndex int64, reader io.Reader, contentLength int64) error {
	err := s.repository.UploadChunkStream(ctx, fileUUID, chunkIndex, reader, contentLength)
	if err != nil {
		return errors.Wrap(err, "upload chunk stream")
	}

	return nil
}

func (s *StorageService) DownloadChunkStream(ctx context.Context, fileUUID string, chunkIndex int64, writer io.Writer) error {
	err := s.repository.DownloadChunkStream(ctx, fileUUID, chunkIndex, writer)
	if err != nil {
		return errors.Wrap(err, "download chunk stream")
	}

	return nil
}
