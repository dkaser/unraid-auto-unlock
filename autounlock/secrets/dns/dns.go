package dns

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
