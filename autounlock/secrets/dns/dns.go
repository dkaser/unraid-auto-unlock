package dns

import (
	"context"
	"fmt"
	"net"
	"strings"

	"github.com/dkaser/unraid-auto-unlock/autounlock/secrets/registry"
)

const (
	// PriorityDNS is the priority for DNS fetcher (checked early, explicit prefix).
	PriorityDNS = 10
)

func init() {
	registry.Register(&Fetcher{})
}

type Fetcher struct {
	Resolver *net.Resolver
}

func (f *Fetcher) Match(path string) bool {
	return strings.HasPrefix(path, "dns:")
}

func (f *Fetcher) Priority() int {
	return PriorityDNS
}

func (f *Fetcher) Fetch(ctx context.Context, domain string) (string, error) {
	// Use the configured resolver or create a default one
	resolver := f.Resolver
	if resolver == nil {
		resolver = &net.Resolver{}
	}

	domain = strings.TrimPrefix(domain, "dns:")

	txts, err := resolver.LookupTXT(ctx, domain)
	if err != nil {
		return "", fmt.Errorf("failed to lookup TXT records for domain %s: %w", domain, err)
	}

	// Return concatenated TXT records
	return strings.Join(txts, ""), nil
}
