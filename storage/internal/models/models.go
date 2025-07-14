package models

type UploadResponse struct {
	FileUUID   string `json:"file_uuid"`
	ChunkIndex int64  `json:"chunk_index"`
}
