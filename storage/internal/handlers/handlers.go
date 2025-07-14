package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"storage/internal/models"
	"storage/internal/service"
)

type StorageHandler struct {
	storageService *service.StorageService
}

func NewStorageHandler(storageService *service.StorageService) *StorageHandler {
	return &StorageHandler{
		storageService: storageService,
	}
}

func (h *StorageHandler) UploadChunk(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	fileUUID := r.URL.Query().Get("file_uuid")
	if fileUUID == "" {
		http.Error(w, "Missing file_uuid parameter", http.StatusBadRequest)
		return
	}

	chunkIndexStr := r.URL.Query().Get("chunk_index")
	if chunkIndexStr == "" {
		http.Error(w, "Missing chunk_index parameter", http.StatusBadRequest)
		return
	}

	chunkIndex, err := strconv.ParseInt(chunkIndexStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid chunk_index parameter", http.StatusBadRequest)
		return
	}

	contentLength := r.ContentLength
	if contentLength < 0 {
		if contentLengthStr := r.Header.Get("Content-Length"); contentLengthStr != "" {
			if parsedLength, err := strconv.ParseInt(contentLengthStr, 10, 64); err == nil {
				contentLength = parsedLength
			}
		}

		if contentLength < 0 {
			http.Error(w, "Content-Length header is required for streaming upload", http.StatusBadRequest)
			return
		}
	}

	err = h.storageService.UploadChunkStream(r.Context(), fileUUID, chunkIndex, r.Body, contentLength)
	if err != nil {
		http.Error(w, "Error uploading chunk: "+err.Error(), http.StatusInternalServerError)
		return
	}

	response := models.UploadResponse{
		FileUUID:   fileUUID,
		ChunkIndex: chunkIndex,
	}

	responseData, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Error encoding response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(responseData)
}

func (h *StorageHandler) DownloadChunk(w http.ResponseWriter, r *http.Request) {
	fileUUID := r.URL.Query().Get("file_uuid")
	if fileUUID == "" {
		http.Error(w, "Missing file_uuid parameter", http.StatusBadRequest)
		return
	}

	chunkIndexStr := r.URL.Query().Get("chunk_index")
	if chunkIndexStr == "" {
		http.Error(w, "Missing chunk_index parameter", http.StatusBadRequest)
		return
	}

	chunkIndex, err := strconv.ParseInt(chunkIndexStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid chunk_index parameter", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", "attachment; filename=chunk_"+strconv.FormatInt(chunkIndex, 10))
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	err = h.storageService.DownloadChunkStream(r.Context(), fileUUID, chunkIndex, w)
	if err != nil {
		http.Error(w, "Error downloading chunk: "+err.Error(), http.StatusInternalServerError)
		return
	}
}
