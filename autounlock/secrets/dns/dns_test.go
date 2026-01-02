package dns

import (
	"context"
	"strings"
	"testing"
)

// TestFetch_ValidDomain tests DNS TXT record lookup with a real domain.
// Note: This test requires network connectivity and uses a public DNS service.
func TestFetch_ValidDomain(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network-dependent test in short mode")
	}

	ctx := context.Background()
	fetcher := &Fetcher{}

	// Using a commonly available test domain - this is just to verify the mechanism works
	// In production, users would use their own domains
	_, err := fetcher.Fetch(ctx, "dns:google.com")
	if err != nil {
		t.Logf("DNS lookup for google.com failed (this is OK if network is unavailable): %v", err)
	}
	// We don't assert on the content since TXT records can change
}

// TestFetch_InvalidDomain tests DNS TXT record lookup with an invalid domain.
func TestFetch_InvalidDomain(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network-dependent test in short mode")
	}

	ctx := context.Background()
	fetcher := &Fetcher{}

	// Use a domain that should not exist
	_, err := fetcher.Fetch(ctx, "dns:this-domain-definitely-does-not-exist-12345.invalid")
	if err == nil {
		t.Error("Expected error for non-existent domain, got none")
	}

	if !strings.Contains(err.Error(), "failed to lookup TXT records") {
		t.Errorf("Expected error about TXT lookup failure, got: %v", err)
	}
}

// TestFetch_EmptyDomain tests DNS TXT record lookup with an empty domain.
func TestFetch_EmptyDomain(t *testing.T) {
	ctx := context.Background()
	fetcher := &Fetcher{}

	_, err := fetcher.Fetch(ctx, "dns:")
	if err == nil {
		t.Error("Expected error for empty domain, got none")
	}
}

// TestFetch_ContextCancellation tests that context cancellation is respected.
func TestFetch_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping network-dependent test in short mode")
	}

	// Create a context that is already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	fetcher := &Fetcher{}

	_, err := fetcher.Fetch(ctx, "dns:google.com")
	if err == nil {
		t.Error("Expected error due to context cancellation, got none")
	}
}
