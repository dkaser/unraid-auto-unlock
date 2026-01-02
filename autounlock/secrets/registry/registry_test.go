package registry

import (
	"context"
	"sync"
	"testing"
)

// mockFetcher is a test implementation of the Fetcher interface.
type mockFetcher struct {
	name     string
	priority int
	matches  bool
}

func (m *mockFetcher) Fetch(_ context.Context, _ string) (string, error) {
	return m.name, nil
}

func (m *mockFetcher) Match(_ string) bool {
	return m.matches
}

func (m *mockFetcher) Priority() int {
	return m.priority
}

// TestRegister_SingleFetcher tests registering a single fetcher.
func TestRegister_SingleFetcher(t *testing.T) {
	// Save and restore original state
	originalFetchers := fetchers

	defer func() { fetchers = originalFetchers }()

	// Clear registry
	fetchers = nil

	mock := &mockFetcher{name: "test", priority: 10, matches: true}
	Register(mock)

	got := GetFetchers()
	if len(got) != 1 {
		t.Fatalf("Expected 1 fetcher, got %d", len(got))
	}

	if got[0] != mock {
		t.Error("Fetcher not registered correctly")
	}
}

// TestRegister_MultipleFetchers tests registering multiple fetchers.
func TestRegister_MultipleFetchers(t *testing.T) {
	// Save and restore original state
	originalFetchers := fetchers

	defer func() { fetchers = originalFetchers }()

	// Clear registry
	fetchers = nil

	mock1 := &mockFetcher{name: "first", priority: 10, matches: true}
	mock2 := &mockFetcher{name: "second", priority: 20, matches: true}
	mock3 := &mockFetcher{name: "third", priority: 5, matches: true}

	Register(mock1)
	Register(mock2)
	Register(mock3)

	got := GetFetchers()
	if len(got) != 3 {
		t.Fatalf("Expected 3 fetchers, got %d", len(got))
	}
}

// TestRegister_SortsByPriority tests that fetchers are sorted by priority.
func TestRegister_SortsByPriority(t *testing.T) {
	// Save and restore original state
	originalFetchers := fetchers

	defer func() { fetchers = originalFetchers }()

	// Clear registry
	fetchers = nil

	// Register in random order
	mock1 := &mockFetcher{name: "low-priority", priority: 100, matches: true}
	mock2 := &mockFetcher{name: "high-priority", priority: 10, matches: true}
	mock3 := &mockFetcher{name: "medium-priority", priority: 50, matches: true}

	Register(mock1)
	Register(mock2)
	Register(mock3)

	got := GetFetchers()
	if len(got) != 3 {
		t.Fatalf("Expected 3 fetchers, got %d", len(got))
	}

	// Should be sorted by priority (lower number first)
	if got[0].Priority() != 10 {
		t.Errorf("First fetcher priority = %d, want 10", got[0].Priority())
	}

	if got[1].Priority() != 50 {
		t.Errorf("Second fetcher priority = %d, want 50", got[1].Priority())
	}

	if got[2].Priority() != 100 {
		t.Errorf("Third fetcher priority = %d, want 100", got[2].Priority())
	}
}

// TestRegister_SamePriority tests multiple fetchers with the same priority.
func TestRegister_SamePriority(t *testing.T) {
	// Save and restore original state
	originalFetchers := fetchers

	defer func() { fetchers = originalFetchers }()

	// Clear registry
	fetchers = nil

	mock1 := &mockFetcher{name: "first", priority: 10, matches: true}
	mock2 := &mockFetcher{name: "second", priority: 10, matches: true}
	mock3 := &mockFetcher{name: "third", priority: 5, matches: true}

	Register(mock1)
	Register(mock2)
	Register(mock3)

	got := GetFetchers()
	if len(got) != 3 {
		t.Fatalf("Expected 3 fetchers, got %d", len(got))
	}

	// First should have priority 5
	if got[0].Priority() != 5 {
		t.Errorf("First fetcher priority = %d, want 5", got[0].Priority())
	}

	// Second and third should both have priority 10 (order doesn't matter between them)
	if got[1].Priority() != 10 || got[2].Priority() != 10 {
		t.Error("Expected remaining fetchers to have priority 10")
	}
}

// TestGetFetchers_ReturnsACopy tests that GetFetchers returns a copy, not the original slice.
func TestGetFetchers_ReturnsACopy(t *testing.T) {
	// Save and restore original state
	originalFetchers := fetchers

	defer func() { fetchers = originalFetchers }()

	// Clear registry
	fetchers = nil

	mock := &mockFetcher{name: "test", priority: 10, matches: true}
	Register(mock)

	// Get fetchers twice
	got1 := GetFetchers()
	got2 := GetFetchers()

	// Should be different slices
	if &got1[0] == &got2[0] {
		t.Error("GetFetchers should return a copy, not the same slice")
	}

	// But contain the same fetchers
	if got1[0] != got2[0] {
		t.Error("GetFetchers should contain the same fetcher instances")
	}
}

// TestGetFetchers_ModifyingReturnedSlice tests that modifying the returned slice doesn't affect the registry.
func TestGetFetchers_ModifyingReturnedSlice(t *testing.T) {
	// Save and restore original state
	originalFetchers := fetchers

	defer func() { fetchers = originalFetchers }()

	// Clear registry
	fetchers = nil

	mock := &mockFetcher{name: "test", priority: 10, matches: true}
	Register(mock)

	// Get fetchers and try to modify the slice
	got := GetFetchers()
	originalLen := len(got)

	// Try to append to the returned slice (should not affect registry)
	_ = append(got, &mockFetcher{name: "hacker", priority: 1, matches: true})

	// Get fetchers again
	got2 := GetFetchers()

	if len(got2) != originalLen {
		t.Errorf(
			"Registry was modified by external slice modification: got %d, want %d",
			len(got2),
			originalLen,
		)
	}
}

// TestRegister_ConcurrentAccess tests thread-safety of Register.
func TestRegister_ConcurrentAccess(t *testing.T) {
	// Save and restore original state
	originalFetchers := fetchers

	defer func() { fetchers = originalFetchers }()

	// Clear registry
	fetchers = nil

	const numGoroutines = 10

	const numRegistrations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrently register fetchers
	for i := range numGoroutines {
		go func(id int) {
			defer wg.Done()

			for j := range numRegistrations {
				mock := &mockFetcher{
					name:     "concurrent",
					priority: id*numRegistrations + j,
					matches:  true,
				}
				Register(mock)
			}
		}(i)
	}

	wg.Wait()

	got := GetFetchers()
	expected := numGoroutines * numRegistrations

	if len(got) != expected {
		t.Errorf("Expected %d fetchers after concurrent registration, got %d", expected, len(got))
	}

	// Verify they're sorted
	for i := 1; i < len(got); i++ {
		if got[i].Priority() < got[i-1].Priority() {
			t.Error("Fetchers not properly sorted after concurrent registration")

			break
		}
	}
}

// TestGetFetchers_ConcurrentAccess tests thread-safety of GetFetchers.
func TestGetFetchers_ConcurrentAccess(t *testing.T) {
	// Save and restore original state
	originalFetchers := fetchers

	defer func() { fetchers = originalFetchers }()

	// Clear registry and add some fetchers
	fetchers = nil

	for i := range 10 {
		Register(&mockFetcher{name: "test", priority: i, matches: true})
	}

	const numGoroutines = 50

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Concurrently read from registry
	for range numGoroutines {
		go func() {
			defer wg.Done()

			for range 100 {
				got := GetFetchers()
				if len(got) != 10 {
					t.Errorf("Expected 10 fetchers, got %d", len(got))

					return
				}
			}
		}()
	}

	wg.Wait()
}

// TestRegister_AndGetFetchers_ConcurrentMixed tests concurrent Register and GetFetchers.
func TestRegister_AndGetFetchers_ConcurrentMixed(t *testing.T) {
	// Save and restore original state
	originalFetchers := fetchers

	defer func() { fetchers = originalFetchers }()

	// Clear registry
	fetchers = nil

	const numGoroutines = 20

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Half the goroutines register, half read
	for i := range numGoroutines {
		if i%2 == 0 {
			// Register
			go func(id int) {
				defer wg.Done()

				for j := range 50 {
					Register(&mockFetcher{
						name:     "mixed",
						priority: id*50 + j,
						matches:  true,
					})
				}
			}(i)
		} else {
			// Read
			go func() {
				defer wg.Done()

				for range 50 {
					got := GetFetchers()
					// Just verify we don't panic or get corrupted data
					if got == nil {
						t.Error("GetFetchers returned nil")

						return
					}
				}
			}()
		}
	}

	wg.Wait()

	// Verify final state
	got := GetFetchers()
	if len(got) == 0 {
		t.Error("Expected fetchers to be registered")
	}

	// Verify they're sorted
	for i := 1; i < len(got); i++ {
		if got[i].Priority() < got[i-1].Priority() {
			t.Error("Fetchers not properly sorted after mixed concurrent access")

			break
		}
	}
}

// TestGetFetchers_EmptyRegistry tests getting fetchers from an empty registry.
func TestGetFetchers_EmptyRegistry(t *testing.T) {
	// Save and restore original state
	originalFetchers := fetchers

	defer func() { fetchers = originalFetchers }()

	// Clear registry
	fetchers = nil

	got := GetFetchers()
	if got == nil {
		t.Error("GetFetchers should not return nil")
	}

	if len(got) != 0 {
		t.Errorf("Expected empty slice, got %d fetchers", len(got))
	}
}
