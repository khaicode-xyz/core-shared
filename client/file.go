package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"mime/multipart"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// FileClient calls file-service internal APIs.
type FileClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *slog.Logger
}

// PresignDownloadResponse is the response from file-service presign-download-url API.
type PresignDownloadResponse struct {
	URL string `json:"url"`
}

// NewFileClient creates a file-service client.
func NewFileClient(baseURL string, timeout time.Duration, logger *slog.Logger) *FileClient {
	return &FileClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout:   timeout,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
		logger: logger,
	}
}

// GetPresignDownloadURL calls POST /internal/presign-download-url to get a presigned S3 download URL.
func (c *FileClient) GetPresignDownloadURL(ctx context.Context, tenantCode, filePath string) (string, error) {
	url := c.baseURL + "/internal/presign-download-url"

	// Build multipart form body
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("tenantCode", tenantCode); err != nil {
		return "", fmt.Errorf("write tenantCode field: %w", err)
	}
	if err := writer.WriteField("filePath", filePath); err != nil {
		return "", fmt.Errorf("write filePath field: %w", err)
	}
	if err := writer.Close(); err != nil {
		return "", fmt.Errorf("close multipart writer: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	c.logger.Info("requesting presign download URL",
		slog.String("file_path", filePath),
		slog.String("tenant_code", tenantCode),
	)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call file-service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("file-service returned status %d", resp.StatusCode)
	}

	var result PresignDownloadResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if result.URL == "" {
		return "", fmt.Errorf("file-service returned empty presign URL")
	}

	return result.URL, nil
}
