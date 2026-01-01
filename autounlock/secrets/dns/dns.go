package dns

import (
	"context"
	"fmt"
	"net"
	"strings"
)

// Fetch retrieves secret data from DNS TXT records.
// The path should be a domain name (without the "dns:" prefix).
func Fetch(ctx context.Context, domain string) (string, error) {
	// Use a custom resolver that respects context
	resolver := &net.Resolver{}

	txts, err := resolver.LookupTXT(ctx, domain)
	if err != nil {
		return "", fmt.Errorf("failed to lookup TXT records for domain %s: %w", domain, err)
	}

	// Return concatenated TXT records
	return strings.Join(txts, ""), nil
}
