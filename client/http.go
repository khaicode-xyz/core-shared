// Package client provides base HTTP and gRPC clients for service-to-service
// communication. All clients are pre-instrumented with OpenTelemetry.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// HTTPClient is a thin wrapper around http.Client with:
//   - base URL joining
//   - OpenTelemetry HTTP instrumentation
//   - JSON encode/decode helpers
//   - structured logging on non-2xx responses
//
// Use it to build service clients (e.g. FileClient). For one-off calls, use
// net/http directly.
type HTTPClient struct {
	baseURL string
	http    *http.Client
	logger  *slog.Logger
	headers map[string]string
}

// HTTPOption configures an HTTPClient at construction time.
type HTTPOption func(*HTTPClient)

// WithHTTPHeader sets a default header applied to every request.
func WithHTTPHeader(key, value string) HTTPOption {
	return func(c *HTTPClient) {
		if c.headers == nil {
			c.headers = map[string]string{}
		}
		c.headers[key] = value
	}
}

// WithHTTPTransport overrides the underlying http.RoundTripper. The replacement
// is still wrapped with otelhttp so spans are preserved.
func WithHTTPTransport(rt http.RoundTripper) HTTPOption {
	return func(c *HTTPClient) {
		c.http.Transport = otelhttp.NewTransport(rt)
	}
}

// NewHTTPClient creates a base HTTP client for service-to-service calls.
// baseURL may be empty if callers always pass absolute URLs.
func NewHTTPClient(baseURL string, timeout time.Duration, logger *slog.Logger, opts ...HTTPOption) *HTTPClient {
	c := &HTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		http: &http.Client{
			Timeout:   timeout,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
		logger: logger,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Do issues an HTTP request with the given method, resolved path, and raw body.
// If path starts with "http", it's treated as absolute; otherwise it's joined
// to baseURL. Response body is returned to the caller for decoding + close.
func (c *HTTPClient) Do(ctx context.Context, method, path string, body io.Reader, contentType string) (*http.Response, error) {
	url := c.resolve(path)

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w", method, url, err)
	}
	return resp, nil
}

// GetJSON performs a GET and decodes the JSON response into out.
// Non-2xx responses return an error carrying the status code.
func (c *HTTPClient) GetJSON(ctx context.Context, path string, out any) error {
	resp, err := c.Do(ctx, http.MethodGet, path, nil, "")
	if err != nil {
		return err
	}
	return c.decodeJSON(resp, out)
}

// PostJSON marshals body, POSTs as application/json, and decodes the response
// into out. Pass out = nil to discard the response body.
func (c *HTTPClient) PostJSON(ctx context.Context, path string, body, out any) error {
	buf, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal body: %w", err)
	}
	resp, err := c.Do(ctx, http.MethodPost, path, bytes.NewReader(buf), "application/json")
	if err != nil {
		return err
	}
	return c.decodeJSON(resp, out)
}

func (c *HTTPClient) resolve(path string) string {
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	if c.baseURL == "" {
		return path
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return c.baseURL + path
}

func (c *HTTPClient) decodeJSON(resp *http.Response, out any) error {
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		c.logger.Warn("http non-2xx response",
			slog.String("url", resp.Request.URL.String()),
			slog.Int("status", resp.StatusCode),
			slog.String("body", string(body)),
		)
		return fmt.Errorf("http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}
