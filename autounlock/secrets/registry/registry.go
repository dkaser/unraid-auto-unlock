package registry

import (
	"context"
	"sort"
	"sync"
)

// Fetcher is an interface for fetching secret data from various sources.
type Fetcher interface {
	// Fetch retrieves the secret data from the given path.
	Fetch(ctx context.Context, path string) (string, error)
	// Match checks if this fetcher can handle the given path.
	// Returns true if matched, false otherwise.
	Match(path string) bool
	// Priority returns the priority of this fetcher (lower number = higher priority).
	// Multiple fetchers with the same priority can run in any order.
	Priority() int
}

var (
	registryMu sync.RWMutex
	fetchers   []Fetcher
)

// Register adds a new fetcher to the registry.
// Fetchers are automatically sorted by priority (lower number = higher priority).
// Each package should call this in its init() function to self-register.
func Register(fetcher Fetcher) {
	registryMu.Lock()
	defer registryMu.Unlock()

	fetchers = append(fetchers, fetcher)

	// Sort fetchers by priority (lower number = higher priority)
	sort.Slice(fetchers, func(i, j int) bool {
		return fetchers[i].Priority() < fetchers[j].Priority()
	})
}

// GetFetchers returns a copy of the registered fetchers in priority order.
func GetFetchers() []Fetcher {
	registryMu.RLock()
	defer registryMu.RUnlock()

	// Return a copy to prevent external modification
	result := make([]Fetcher, len(fetchers))
	copy(result, fetchers)

	return result
}
