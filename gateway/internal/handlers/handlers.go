package handlers

import (
	"fmt"
	"net/http"

	jsoniter "github.com/json-iterator/go"

	"gateway/internal/service"
)

type GatewayHandler struct {
	chunkerService *service.ChunkerService
}

func NewGatewayHandler(chunkerService *service.ChunkerService) *GatewayHandler {
	return &GatewayHandler{
		chunkerService: chunkerService,
	}
}

func (s *GatewayHandler) UploadFile(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error getting file from form: "+err.Error(), http.StatusBadRequest)
		return
	}

	defer file.Close()

	fileUUID, err := s.chunkerService.InsertStream(r.Context(), file, header)
	if err != nil {
		http.Error(w, "Error loading file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	err = jsoniter.NewEncoder(w).Encode(map[string]string{"file_uuid": fileUUID})
	if err != nil {
		http.Error(w, "Error encoding response: "+err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *GatewayHandler) GetFile(w http.ResponseWriter, r *http.Request) {
	fileUUID := r.URL.Query().Get("file_uuid")
	if fileUUID == "" {
		http.Error(w, "Missing file_uuid parameter", http.StatusBadRequest)
		return
	}

	fmt.Printf("DEBUG: Getting file with UUID: %s\n", fileUUID)

	err := s.chunkerService.SelectStream(r.Context(), fileUUID, w)
	if err != nil {
		fmt.Printf("DEBUG: Error in SelectStream: %v\n", err)
		http.Error(w, "Error downloading file: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")
}
