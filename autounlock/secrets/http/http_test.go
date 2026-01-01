package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestFetch_Success tests successful HTTP GET requests.
func TestFetch_Success(t *testing.T) {
	testCases := []struct {
		name         string
		responseBody string
		want         string
	}{
		{
			name:         "simple text",
			responseBody: "test-content",
			want:         "test-content",
		},
		{
			name:         "text with whitespace",
			responseBody: "  test-content  \n",
			want:         "test-content",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(tc.responseBody))
				}),
			)
			defer server.Close()

			ctx := context.Background()

			got, err := Fetch(ctx, server.URL)
			if err != nil {
				t.Fatalf("Fetch failed: %v", err)
			}

			if got != tc.want {
				t.Errorf("Fetch() = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestFetch_BasicAuth tests HTTP basic authentication.
func TestFetch_BasicAuth(t *testing.T) {
	expectedUser := "testuser"
	expectedPass := "testpass"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		if user != expectedUser || pass != expectedPass {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("authenticated"))
	}))
	defer server.Close()

	// Build URL with auth credentials
	urlWithAuth := "http://" + expectedUser + ":" + expectedPass + "@" + server.Listener.Addr().
		String()

	ctx := context.Background()

	got, err := Fetch(ctx, urlWithAuth)
	if err != nil {
		t.Fatalf("Fetch with basic auth failed: %v", err)
	}

	if got != "authenticated" {
		t.Errorf("Fetch() = %q, want %q", got, "authenticated")
	}
}

// TestFetch_InsecureTLS tests HTTPS with insecure TLS verification.
func TestFetch_InsecureTLS(t *testing.T) {
	// Create HTTPS server with self-signed certificate
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("secure-content"))
	}))
	defer server.Close()

	// Standard HTTPS request should fail with self-signed cert
	ctx := context.Background()

	_, err := Fetch(ctx, server.URL)
	if err == nil {
		t.Error("Expected error with self-signed certificate, got none")
	}

	// Request with https+insecure:// should succeed
	insecureURL := "https+insecure://" + server.Listener.Addr().String()

	got, err := Fetch(ctx, insecureURL)
	if err != nil {
		t.Fatalf("Fetch with insecure flag failed: %v", err)
	}

	if got != "secure-content" {
		t.Errorf("Fetch() = %q, want %q", got, "secure-content")
	}
}

// TestFetch_HTTPStatusErrors tests various HTTP error status codes.
func TestFetch_HTTPStatusErrors(t *testing.T) {
	testCases := []struct {
		name       string
		statusCode int
	}{
		{"404 Not Found", http.StatusNotFound},
		{"403 Forbidden", http.StatusForbidden},
		{"500 Internal Server Error", http.StatusInternalServerError},
		{"401 Unauthorized", http.StatusUnauthorized},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(tc.statusCode)
				}),
			)
			defer server.Close()

			ctx := context.Background()

			_, err := Fetch(ctx, server.URL)
			if err == nil {
				t.Errorf("Expected error for status code %d, got none", tc.statusCode)
			}
		})
	}
}

// TestFetch_InvalidURLs tests error handling for invalid URLs.
func TestFetch_InvalidURLs(t *testing.T) {
	testCases := []struct {
		name string
		url  string
	}{
		{"malformed URL", "ht!tp://invalid"},
		{"unsupported scheme", "ftp://example.com/file.txt"},
		{"empty URL", ""},
		{"invalid characters", "https://example.com/<invalid>"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()

			_, err := Fetch(ctx, tc.url)
			if err == nil {
				t.Errorf("Expected error for URL %q, got none", tc.url)
			}
		})
	}
}

// TestFetch_SubdirectoryPath tests that URLs with subdirectories work correctly.
func TestFetch_SubdirectoryPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the full path is preserved
		if r.URL.Path != "/share/subdir/file.txt" {
			t.Errorf("Expected path /share/subdir/file.txt, got %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)

			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("subdirectory-content"))
	}))
	defer server.Close()

	ctx := context.Background()

	got, err := Fetch(ctx, server.URL+"/share/subdir/file.txt")
	if err != nil {
		t.Fatalf("Fetch with subdirectory path failed: %v", err)
	}

	if got != "subdirectory-content" {
		t.Errorf("Fetch() = %q, want %q", got, "subdirectory-content")
	}
}

// TestFetch_CombinedFeatures tests URL with both insecure flag and basic auth.
func TestFetch_CombinedFeatures(t *testing.T) {
	expectedUser := "user"
	expectedPass := "pass"

	// Create HTTPS server with self-signed certificate
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok || user != expectedUser || pass != expectedPass {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("secure-authenticated"))
	}))
	defer server.Close()

	// Build URL with insecure flag and auth credentials
	urlWithAuth := "https+insecure://" + expectedUser + ":" + expectedPass + "@" + server.Listener.Addr().
		String()

	ctx := context.Background()

	got, err := Fetch(ctx, urlWithAuth)
	if err != nil {
		t.Fatalf("Fetch with combined features failed: %v", err)
	}

	if got != "secure-authenticated" {
		t.Errorf("Fetch() = %q, want %q", got, "secure-authenticated")
	}
}

// TestFetch_URLEncodedCredentials tests that special characters in credentials are handled.
func TestFetch_URLEncodedCredentials(t *testing.T) {
	expectedUser := "user@example.com"
	expectedPass := "p@ss:word!"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok {
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		if user != expectedUser || pass != expectedPass {
			t.Errorf("Auth mismatch: got user=%q pass=%q, want user=%q pass=%q",
				user, pass, expectedUser, expectedPass)
			w.WriteHeader(http.StatusUnauthorized)

			return
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte("encoded-auth"))
	}))
	defer server.Close()

	// URL encode the credentials
	encodedUser := "user%40example.com"
	encodedPass := "p%40ss%3Aword%21" //nolint:gosec // Test credential, not real
	urlWithAuth := "http://" + encodedUser + ":" + encodedPass + "@" + server.Listener.Addr().
		String()

	ctx := context.Background()

	got, err := Fetch(ctx, urlWithAuth)
	if err != nil {
		t.Fatalf("Fetch with encoded credentials failed: %v", err)
	}

	if got != "encoded-auth" {
		t.Errorf("Fetch() = %q, want %q", got, "encoded-auth")
	}
}

// TestFetch_ResponseTooLarge tests that responses exceeding the size limit are rejected.
func TestFetch_ResponseTooLarge(t *testing.T) {
	largeBody := strings.Repeat("A", maxResponseSize+1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(largeBody))
	}))
	defer server.Close()

	ctx := context.Background()

	_, err := Fetch(ctx, server.URL)
	if err == nil {
		t.Error("Expected error for response body too large, got none")
	}

	if !strings.Contains(err.Error(), "response body too large") {
		t.Errorf("Expected error about response body too large, got: %v", err)
	}
}

// TestFetch_ResponseAtLimit tests that responses at the size limit are accepted.
func TestFetch_ResponseAtLimit(t *testing.T) {
	bodyAtLimit := strings.Repeat("A", maxResponseSize)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(bodyAtLimit))
	}))
	defer server.Close()

	ctx := context.Background()

	got, err := Fetch(ctx, server.URL)
	if err != nil {
		t.Fatalf("Fetch failed for response at limit: %v", err)
	}

	if got != bodyAtLimit {
		t.Errorf("Fetch() returned %d bytes, want %d bytes", len(got), len(bodyAtLimit))
	}
}

// TestParseURL_HTTP tests parseURL with http URLs.
func TestParseURL_HTTP(t *testing.T) {
	parsedURL, insecure, err := parseURL("http://example.com/path")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if parsedURL.Scheme != "http" {
		t.Errorf("Scheme = %q, want %q", parsedURL.Scheme, "http")
	}

	if insecure {
		t.Errorf("Insecure = %v, want false", insecure)
	}
}

// TestParseURL_HTTPS tests parseURL with https URLs.
func TestParseURL_HTTPS(t *testing.T) {
	parsedURL, insecure, err := parseURL("https://example.com/path")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if parsedURL.Scheme != "https" {
		t.Errorf("Scheme = %q, want %q", parsedURL.Scheme, "https")
	}

	if insecure {
		t.Errorf("Insecure = %v, want false", insecure)
	}
}

// TestParseURL_HTTPSInsecure tests parseURL with https+insecure URLs.
func TestParseURL_HTTPSInsecure(t *testing.T) {
	parsedURL, insecure, err := parseURL("https+insecure://example.com/path")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if parsedURL.Scheme != "https" {
		t.Errorf("Scheme = %q, want %q", parsedURL.Scheme, "https")
	}

	if !insecure {
		t.Errorf("Insecure = %v, want true", insecure)
	}
}

// TestParseURL_UnsupportedScheme tests parseURL with unsupported URL schemes.
func TestParseURL_UnsupportedScheme(t *testing.T) {
	_, _, err := parseURL("ftp://example.com/file")
	if err == nil {
		t.Error("Expected error for unsupported scheme, got none")
	}

	if !strings.Contains(err.Error(), "unsupported URL scheme") {
		t.Errorf("Expected error about unsupported scheme, got: %v", err)
	}
}

// TestParseURL_Invalid tests parseURL with invalid URLs.
func TestParseURL_Invalid(t *testing.T) {
	_, _, err := parseURL("ht!tp://invalid")
	if err == nil {
		t.Error("Expected error for invalid URL, got none")
	}

	if !strings.Contains(err.Error(), "invalid URL") {
		t.Errorf("Expected error about invalid URL, got: %v", err)
	}
}

// TestParseURL_WithAuth tests parseURL with URLs containing authentication.
func TestParseURL_WithAuth(t *testing.T) {
	parsedURL, insecure, err := parseURL("https://user:pass@example.com/path")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if parsedURL.Scheme != "https" {
		t.Errorf("Scheme = %q, want %q", parsedURL.Scheme, "https")
	}

	if insecure {
		t.Errorf("Insecure = %v, want false", insecure)
	}

	if parsedURL.User == nil {
		t.Error("Expected user info in URL, got none")
	}
}

// TestParseURL_InsecureWithAuth tests parseURL with insecure URLs containing authentication.
func TestParseURL_InsecureWithAuth(t *testing.T) {
	parsedURL, insecure, err := parseURL("https+insecure://user:pass@example.com/path")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if parsedURL.Scheme != "https" {
		t.Errorf("Scheme = %q, want %q", parsedURL.Scheme, "https")
	}

	if !insecure {
		t.Errorf("Insecure = %v, want true", insecure)
	}

	if parsedURL.User == nil {
		t.Error("Expected user info in URL, got none")
	}
}
