package http

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	maxResponseSize = 512
)

// Client is an interface for making HTTP requests.
// This allows for easier testing and mocking if needed in the future.
type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

// createClient creates an HTTP client with optional insecure TLS configuration.
func createClient(insecure bool) *http.Client {
	if insecure {
		return &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, //nolint:gosec // Explicitly requested by user via https+insecure://
				},
			},
		}
	}

	return http.DefaultClient
}

// Fetch retrieves content from an HTTP(S) URL with optional insecure TLS and basic auth support.
// Supported URL formats:
//   - http://example.com/path
//   - https://example.com/path
//   - https+insecure://example.com/path (skips TLS verification)
//   - https://user:pass@example.com/path (basic auth)
//   - https+insecure://user:pass@example.com/path (both options)
func Fetch(ctx context.Context, urlStr string) (string, error) {
	return fetchWithClient(ctx, urlStr, nil)
}

// parseURL parses and validates the URL, handling the https+insecure:// prefix.
func parseURL(urlStr string) (*url.URL, bool, error) {
	insecure := false
	if strings.HasPrefix(urlStr, "https+insecure://") {
		insecure = true
		urlStr = strings.Replace(urlStr, "https+insecure://", "https://", 1)
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, false, fmt.Errorf("invalid URL: %w", err)
	}

	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return nil, false, fmt.Errorf("unsupported URL scheme: %s", parsedURL.Scheme)
	}

	return parsedURL, insecure, nil
}

// fetchWithClient is the internal implementation that allows injecting a custom HTTP client.
// This is useful for testing but not exposed in the public API.
func fetchWithClient(ctx context.Context, urlStr string, client Client) (string, error) {
	parsedURL, insecure, err := parseURL(urlStr)
	if err != nil {
		return "", err
	}

	// Create HTTP client if not provided
	if client == nil {
		client = createClient(insecure)
	}

	// Create a sanitized copy of the URL without credentials to prevent leaking them in logs
	sanitizedURL := *parsedURL
	sanitizedURL.User = nil

	// Create request with sanitized URL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, sanitizedURL.String(), nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	// Handle Basic Auth from URL - set credentials via header after request creation
	if parsedURL.User != nil {
		username := parsedURL.User.Username()
		password, _ := parsedURL.User.Password()
		req.SetBasicAuth(username, password)
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP request failed with status: %d", resp.StatusCode)
	}

	// Read response with size limit to protect against misconfigured endpoints
	limitedReader := io.LimitReader(resp.Body, maxResponseSize+1)

	data, err := io.ReadAll(limitedReader)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if len(data) > maxResponseSize {
		return "", fmt.Errorf("response body too large: exceeds %d byte limit", maxResponseSize)
	}

	return strings.TrimSpace(string(data)), nil
}
