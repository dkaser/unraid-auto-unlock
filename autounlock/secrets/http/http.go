package http

/*
	autounlock - Unraid Auto Unlock
	Copyright (C) 2025-2026 Derek Kaser

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/dkaser/unraid-auto-unlock/autounlock/secrets/registry"
)

const (
	maxResponseSize = 512
	PriorityHTTP    = 20
)

func init() {
	registry.Register(&Fetcher{})
}

type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

type Fetcher struct {
	// Client can be optionally set for testing. If nil, a default client is created.
	Client Client
}

func (f *Fetcher) Match(path string) bool {
	return strings.HasPrefix(path, "http")
}

func (f *Fetcher) Priority() int {
	return PriorityHTTP
}

// Fetch retrieves content from an HTTP(S) URL with optional insecure TLS and basic auth support.
// Supported URL formats:
//   - http://example.com/path
//   - https://example.com/path
//   - https+insecure://example.com/path (skips TLS verification)
//   - https://user:pass@example.com/path (basic auth)
//   - https+insecure://user:pass@example.com/path (both options)
func (f *Fetcher) Fetch(ctx context.Context, urlStr string) (string, error) {
	return f.fetchWithClient(ctx, urlStr)
}

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
func (f *Fetcher) fetchWithClient(ctx context.Context, urlStr string) (string, error) {
	parsedURL, insecure, err := parseURL(urlStr)
	if err != nil {
		return "", err
	}

	// Use the configured client or create a new one
	client := f.Client
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
