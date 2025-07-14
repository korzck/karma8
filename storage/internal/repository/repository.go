package repository

import (
	"context"
	"io"
	"strconv"

	"github.com/minio/minio-go/v7"
	"github.com/pkg/errors"
)

type Repository struct {
	client *minio.Client
	bucket string
}

func NewRepository(client *minio.Client, bucket string) *Repository {
	return &Repository{
		client: client,
		bucket: bucket,
	}
}

func (r *Repository) UploadChunkStream(ctx context.Context, fileUUID string, chunkIndex int64, reader io.Reader, contentLength int64) error {
	objectName := r.getObjectName(fileUUID, chunkIndex)

	readCloser := io.NopCloser(reader)

	_, err := r.client.PutObject(ctx, r.bucket, objectName, readCloser, contentLength, minio.PutObjectOptions{})
	if err != nil {
		return errors.Wrap(err, "put object stream")
	}

	return nil
}

func (r *Repository) DownloadChunkStream(ctx context.Context, fileUUID string, chunkIndex int64, writer io.Writer) error {
	objectName := r.getObjectName(fileUUID, chunkIndex)

	obj, err := r.client.GetObject(ctx, r.bucket, objectName, minio.GetObjectOptions{})
	if err != nil {
		return errors.Wrap(err, "get object")
	}
	defer obj.Close()

	_, err = io.Copy(writer, obj)
	if err != nil {
		return errors.Wrap(err, "copy")
	}

	return nil
}

func (r *Repository) getObjectName(fileUUID string, chunkIndex int64) string {
	return fileUUID + "_chunk_" + strconv.FormatInt(chunkIndex, 10)
}
