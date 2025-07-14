package models

type ChunkStatus string

func (s ChunkStatus) String() string {
	return string(s)
}

const (
	ChunkStatusPending       ChunkStatus = "pending"
	ChunkStatusSentToStorage ChunkStatus = "sent_to_storage"
)
