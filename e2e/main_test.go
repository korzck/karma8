package main

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type UploadResponse struct {
	FileUUID string `json:"file_uuid"`
}

func TestFileUploadAndDownload(t *testing.T) {
	gatewayURL := "http://localhost:8080"
	if os.Getenv("GATEWAY_URL") != "" {
		gatewayURL = os.Getenv("GATEWAY_URL")
	}

	client := &http.Client{Timeout: 30 * time.Second}

	waitForGateway(t, client, gatewayURL)

	t.Run("SmallTextFile", func(t *testing.T) {
		originalContent := "Hello, this is a test file for e2e testing!"
		fileName := "test.txt"

		fileUUID := uploadFile(t, client, gatewayURL, fileName, []byte(originalContent))
		t.Logf("Uploaded file with UUID: %s", fileUUID)

		downloadedContent := downloadFile(t, client, gatewayURL, fileUUID)
		assert.Equal(t, originalContent, string(downloadedContent), "Downloaded content should match original")
	})

	t.Run("LargeBinaryFile", func(t *testing.T) {
		fileSize := 1024 * 1024 * 100
		originalContent := make([]byte, fileSize)
		_, err := rand.Read(originalContent)
		require.NoError(t, err, "Failed to generate random content")

		fileName := "large_binary.bin"
		fileUUID := uploadFile(t, client, gatewayURL, fileName, originalContent)
		t.Logf("Uploaded large binary file with UUID: %s", fileUUID)

		downloadedContent := downloadFile(t, client, gatewayURL, fileUUID)
		assert.Equal(t, len(originalContent), len(downloadedContent), "Downloaded file size should match original")
		assert.Equal(t, originalContent, downloadedContent, "Downloaded content should match original")
	})

	t.Run("MultipleFiles", func(t *testing.T) {
		content1 := "First file content"
		content2 := "Second file content"

		fileUUID1 := uploadFile(t, client, gatewayURL, "file1.txt", []byte(content1))
		fileUUID2 := uploadFile(t, client, gatewayURL, "file2.txt", []byte(content2))

		assert.NotEqual(t, fileUUID1, fileUUID2, "Different files should have different UUIDs")

		downloaded1 := downloadFile(t, client, gatewayURL, fileUUID1)
		downloaded2 := downloadFile(t, client, gatewayURL, fileUUID2)

		assert.Equal(t, content1, string(downloaded1), "First file content should match")
		assert.Equal(t, content2, string(downloaded2), "Second file content should match")
	})

	t.Run("DownloadNonExistentFile", func(t *testing.T) {
		nonExistentUUID := "non-existent-uuid-12345"
		resp, err := client.Get(fmt.Sprintf("%s/api/files/get?file_uuid=%s", gatewayURL, nonExistentUUID))
		require.NoError(t, err, "Request should not fail")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusInternalServerError, resp.StatusCode, "Should return 500 for non-existent file")
	})
}

func TestGatewayHealth(t *testing.T) {
	gatewayURL := "http://localhost:8080"
	if os.Getenv("GATEWAY_URL") != "" {
		gatewayURL = os.Getenv("GATEWAY_URL")
	}

	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(gatewayURL + "/health")
	require.NoError(t, err, "Health check request should not fail")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Gateway health check should return 200")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read health response")

	assert.Contains(t, string(body), "ok", "Health response should contain 'ok'")
}

func waitForGateway(t *testing.T, client *http.Client, gatewayURL string) {
	maxRetries := 30
	for i := 0; i < maxRetries; i++ {
		resp, err := client.Get(gatewayURL + "/health")
		if err == nil && resp.StatusCode == http.StatusOK {
			log.Printf("Gateway is ready at %s", gatewayURL)
			return
		}
		log.Printf("Waiting for gateway to be ready... (attempt %d/%d)", i+1, maxRetries)
		time.Sleep(2 * time.Second)
	}
	t.Fatalf("Gateway did not become ready within expected time")
}

func uploadFile(t *testing.T, client *http.Client, gatewayURL, fileName string, content []byte) string {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("file", fileName)
	require.NoError(t, err, "Failed to create form file")

	_, err = part.Write(content)
	require.NoError(t, err, "Failed to write file content")

	err = writer.Close()
	require.NoError(t, err, "Failed to close writer")

	req, err := http.NewRequest("POST", gatewayURL+"/api/files/upload", &buf)
	require.NoError(t, err, "Failed to create request")

	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := client.Do(req)
	require.NoError(t, err, "Failed to send upload request")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Upload should succeed")

	var uploadResp UploadResponse
	err = json.NewDecoder(resp.Body).Decode(&uploadResp)
	require.NoError(t, err, "Failed to decode upload response")

	assert.NotEmpty(t, uploadResp.FileUUID, "File UUID should not be empty")

	return uploadResp.FileUUID
}

func downloadFile(t *testing.T, client *http.Client, gatewayURL, fileUUID string) []byte {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/api/files/get?file_uuid=%s", gatewayURL, fileUUID), nil)
	require.NoError(t, err, "Failed to create download request")

	resp, err := client.Do(req)
	require.NoError(t, err, "Failed to send download request")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Download should succeed")

	content, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read download response")

	return content
}
