package rclone

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
	"io"
	"strings"

	_ "github.com/rclone/rclone/backend/all" // Import all rclone backends
	"github.com/rclone/rclone/fs"

	"github.com/dkaser/unraid-auto-unlock/autounlock/secrets/registry"
)

const (
	// PriorityRclone is the priority for rclone fetcher (lowest priority, catch-all default).
	PriorityRclone = 100
)

func init() {
	registry.Register(&Fetcher{})
}

// Fetcher implements the secret fetching interface for rclone-based file retrieval.
// Supports local files and remote backends (S3, SFTP, etc.).
type Fetcher struct{}

// Match always returns true for rclone, as it's the catch-all default.
// All paths that don't match other fetchers will be handled by rclone.
func (f *Fetcher) Match(_ string) bool {
	return true
}

// Priority returns 100 for rclone (lowest priority, catch-all default).
func (f *Fetcher) Priority() int {
	return PriorityRclone
}

// Fetch retrieves secret data using rclone from various backends.
// Supports local files and remote backends (S3, SFTP, etc.).
// Path format:
//   - Local files: /path/to/file or relative/path/to/file
//   - Remote backends: :backend:bucket/path/to/file
func (f *Fetcher) Fetch(ctx context.Context, path string) (string, error) {
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
