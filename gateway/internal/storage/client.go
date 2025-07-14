package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/pkg/errors"
)

type Client struct {
	httpClient *http.Client
	baseURL    string
}

func NewClient(storageAddr string) (*Client, error) {
	baseURL := fmt.Sprintf("http://%s", storageAddr)

	if len(baseURL) > 6 && baseURL[len(baseURL)-5:] == ":9090" {
		baseURL = baseURL[:len(baseURL)-5] + ":8081"
	}

	httpClient := &http.Client{
		Timeout: 30 * time.Second,
	}

	return &Client{
		httpClient: httpClient,
		baseURL:    baseURL,
	}, nil
}

func (c *Client) Close() error {
	return nil
}

func (c *Client) UploadChunkStream(ctx context.Context, fileUUID string, chunkIndex int64, reader io.Reader, contentLength int64) error {
	url := fmt.Sprintf("%s/api/chunks/upload?file_uuid=%s&chunk_index=%d", c.baseURL, fileUUID, chunkIndex)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return errors.Wrap(err, "new request with context")
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("Content-Length", fmt.Sprintf("%d", contentLength))

	req.Body = io.NopCloser(reader)
	req.ContentLength = contentLength

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "do request")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return errors.Wrapf(errors.New(string(body)), "upload failed with status %d", resp.StatusCode)
	}

	return nil
}

func (c *Client) DownloadChunkStream(ctx context.Context, fileUUID string, chunkIndex int64, writer io.Writer) error {
	url := fmt.Sprintf("%s/api/chunks/download?file_uuid=%s&chunk_index=%d", c.baseURL, fileUUID, chunkIndex)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return errors.Wrap(err, "new request with context")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "do")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return errors.Wrapf(errors.New(string(body)), "download failed with status %d", resp.StatusCode)
	}

	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		return errors.Wrap(err, "copy response to writer")
	}

	return nil
}
