package rclone

import (
	"context"
	"fmt"
	"io"
	"strings"

	_ "github.com/rclone/rclone/backend/all" // Import all rclone backends
	"github.com/rclone/rclone/fs"
)

// Fetch retrieves secret data using rclone from various backends.
// Supports local files and remote backends (S3, SFTP, etc.).
// Path format:
//   - Local files: /path/to/file or relative/path/to/file
//   - Remote backends: :backend:bucket/path/to/file
func Fetch(ctx context.Context, path string) (string, error) {
	var fsPath, objPath string

	// Handle local file paths vs remote backends
	switch {
	case !strings.HasPrefix(path, ":"):
		// Local file: split into directory and file
		dir, file := splitLocalPath(path)
		fsPath = dir
		objPath = file
	case strings.HasPrefix(path, ":http"):
		fsPath = path
		objPath = ""
	default:
		// Remote backend: split at last '/'
		idx := strings.LastIndex(path, "/")
		if idx == -1 {
			return "", fmt.Errorf("invalid backend path: %s", path)
		}

		fsPath = path[:idx]
		objPath = path[idx+1:]
	}

	fsys, err := fs.NewFs(ctx, fsPath)
	if err != nil {
		return "", fmt.Errorf("failed to create filesystem: %w", err)
	}

	obj, err := fsys.NewObject(ctx, objPath)
	if err != nil {
		return "", fmt.Errorf("failed to open object: %w", err)
	}

	reader, err := obj.Open(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to open: %w", err)
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return "", fmt.Errorf("failed to read: %w", err)
	}

	return strings.TrimSpace(string(data)), nil
}

// splitLocalPath splits a local file path into directory and file name.
func splitLocalPath(path string) (string, string) {
	idx := strings.LastIndex(path, "/")
	if idx == -1 {
		return ".", path // file in current directory
	}

	return path[:idx], path[idx+1:]
}
