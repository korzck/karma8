package repository

import (
	"context"
	"time"

	"gateway/internal/models"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type Chunk struct {
	UUID        string    `db:"uuid"`
	ChunkIndex  int64     `db:"chunk_index"`
	ChunkHash   string    `db:"chunk_hash"`
	Status      string    `db:"status"`
	UpdatedAt   time.Time `db:"updated_at"`
	NumOfChunks int64     `db:"num_of_chunks"`
	StorageID   int       `db:"storage_id"`
}

type Repository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{
		db: db,
	}
}

func (r *Repository) InsertChunk(
	ctx context.Context,
	uuid string,
	chunkIndex int64,
	chunkHash string,
	status models.ChunkStatus,
	numOfChunks int64,
	storageID int,
) error {
	_, err := r.db.ExecContext(ctx, `
		insert into chunks (uuid, chunk_index, chunk_hash, status, num_of_chunks, storage_id) values ($1, $2, $3, $4, $5, $6)
		on conflict (uuid, chunk_index) do update set
			chunk_hash = excluded.chunk_hash,
			status = excluded.status,
			storage_id = excluded.storage_id,
			updated_at = excluded.updated_at
	`, uuid, chunkIndex, chunkHash, status, numOfChunks, storageID)
	if err != nil {
		return errors.Wrap(err, "exec context")
	}

	return nil
}

func (r *Repository) UpdateChunkStatus(
	ctx context.Context,
	uuid string,
	chunkIndex int64,
	status models.ChunkStatus,
	updatedAt time.Time,
) error {
	_, err := r.db.ExecContext(ctx, `
		update chunks set status = $1, updated_at = $2 where uuid = $3 and chunk_index = $4
	`, status, updatedAt, uuid, chunkIndex)
	if err != nil {
		return errors.Wrap(err, "exec context")
	}

	return nil
}

func (r *Repository) GetChunksByUUID(ctx context.Context, uuid string) ([]Chunk, error) {
	var chunks []Chunk
	err := r.db.SelectContext(ctx, &chunks, `
		select * from chunks where uuid = $1 order by chunk_index
	`, uuid)
	if err != nil {
		return nil, errors.Wrap(err, "select context")
	}
	return chunks, nil
}
