package http

import (
	"context"
	"net/http"
	"net/http/httptest"
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
