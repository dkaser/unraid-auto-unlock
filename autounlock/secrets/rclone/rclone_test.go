package rclone

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFetch_LocalFile tests fetching a local file using rclone.
func TestFetch_LocalFile(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "test-secret-content"

	err := os.WriteFile(testFile, []byte(testContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()

	got, err := Fetch(ctx, testFile)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if got != testContent {
		t.Errorf("Fetch() = %q, want %q", got, testContent)
	}
}

// TestFetch_LocalFileWithWhitespace tests fetching a local file with whitespace trimming.
func TestFetch_LocalFileWithWhitespace(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := "  test-content  \n\t"

	err := os.WriteFile(testFile, []byte(testContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	ctx := context.Background()

	got, err := Fetch(ctx, testFile)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	want := "test-content"
	if got != want {
		t.Errorf("Fetch() = %q, want %q", got, want)
	}
}

// TestFetch_RelativePath tests fetching a file using a relative path.
func TestFetch_RelativePath(t *testing.T) {
	// Create a temporary directory and file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "relative.txt")
	testContent := "relative-path-content"

	err := os.WriteFile(testFile, []byte(testContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Change to the temp directory and ensure we restore it
	t.Chdir(tmpDir)

	ctx := context.Background()

	got, err := Fetch(ctx, "relative.txt")
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	if got != testContent {
		t.Errorf("Fetch() = %q, want %q", got, testContent)
	}
}

// TestFetch_NonExistentFile tests fetching a non-existent file.
func TestFetch_NonExistentFile(t *testing.T) {
	ctx := context.Background()

	_, err := Fetch(ctx, "/nonexistent/path/to/file.txt")
	if err == nil {
		t.Error("Expected error for non-existent file, got none")
	}
}

// TestFetch_InvalidBackendPath tests fetching with an invalid backend path.
func TestFetch_InvalidBackendPath(t *testing.T) {
	ctx := context.Background()

	// Backend path without a slash should fail
	_, err := Fetch(ctx, ":s3:invalid")
	if err == nil {
		t.Error("Expected error for invalid backend path, got none")
	}

	if !strings.Contains(err.Error(), "invalid backend path") {
		t.Errorf("Expected error about invalid backend path, got: %v", err)
	}
}

// TestSplitLocalPath tests the splitLocalPath function.
func TestSplitLocalPath(t *testing.T) {
	testCases := []struct {
		name     string
		path     string
		wantDir  string
		wantFile string
	}{
		{
			name:     "absolute path",
			path:     "/path/to/file.txt",
			wantDir:  "/path/to",
			wantFile: "file.txt",
		},
		{
			name:     "relative path",
			path:     "relative/path/file.txt",
			wantDir:  "relative/path",
			wantFile: "file.txt",
		},
		{
			name:     "file in current directory",
			path:     "file.txt",
			wantDir:  ".",
			wantFile: "file.txt",
		},
		{
			name:     "single directory",
			path:     "dir/file.txt",
			wantDir:  "dir",
			wantFile: "file.txt",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gotDir, gotFile := splitLocalPath(tc.path)

			if gotDir != tc.wantDir {
				t.Errorf("Directory = %q, want %q", gotDir, tc.wantDir)
			}

			if gotFile != tc.wantFile {
				t.Errorf("File = %q, want %q", gotFile, tc.wantFile)
			}
		})
	}
}
