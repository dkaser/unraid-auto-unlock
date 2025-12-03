package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/bytemare/secret-sharing/keys"
	_ "github.com/rclone/rclone/backend/all"
	"github.com/rclone/rclone/fs"
	"github.com/rs/zerolog/log"
)

func FetchShare(ctx context.Context, path string) (string, error) {
	// Check for DNS protocol
	if after, ok := strings.CutPrefix(path, "dns:"); ok {
		domain := after

		return fetchDNSTXT(domain)
	}

	// Use rclone for everything else
	return fetchWithRclone(ctx, path)
}

func fetchDNSTXT(domain string) (string, error) {
	txts, err := net.LookupTXT(domain)
	if err != nil {
		return "", fmt.Errorf("failed to lookup TXT records for domain %s: %w", domain, err)
	}

	// Return concatenated TXT records
	return strings.Join(txts, ""), nil
}

func fetchWithRclone(ctx context.Context, path string) (string, error) {
	var fsPath, objPath string

	// Handle local file paths
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

func readPathsFromFile(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open paths file: %w", err)
	}
	defer file.Close()

	var paths []string

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		paths = append(paths, line)
	}

	err = scanner.Err()
	if err != nil {
		return nil, fmt.Errorf("error reading paths from file: %w", err)
	}

	return paths, nil
}

func GetShares(
	paths []string,
	state State,
	retryInterval uint16,
	test bool,
) ([]*keys.KeyShare, error) {
	var shares []*keys.KeyShare

	triedPaths := make(map[string]bool)
	seenShares := make(map[string]bool)

	retryDuration := time.Duration(retryInterval) * time.Second

	for i, path := range paths {
		log.Debug().Int("path", i).Str("target", path).Msg("Configured share path")
	}

	for {
		if (!VerifyArrayStatus("Stopped")) && !test {
			return nil, errors.New("array is no longer stopped, aborting share retrieval")
		}

		for pathNum, path := range paths {
			// Skip paths we've already tried
			if triedPaths[path] {
				continue
			}

			shareStr, err := FetchShare(context.Background(), path)
			if err != nil {
				log.Debug().Int("path", pathNum).Stack().Err(err).Msg("Failed to fetch share")

				continue
			}

			// Mark path as tried after successful fetch
			triedPaths[path] = true

			share, err := GetShare(shareStr, state.SigningKey)
			if err != nil {
				log.Debug().Int("path", pathNum).Stack().Err(err).Msg("Failed to get share")

				continue
			}

			// Use share identifier to detect duplicates
			shareID := strconv.FormatUint(uint64(share.Identifier()), 10)
			if seenShares[shareID] {
				log.Debug().Int("path", pathNum).Msg("Duplicate share, ignoring")

				continue
			}

			seenShares[shareID] = true

			log.Info().Int("path", pathNum).Msg("Successfully retrieved share")

			shares = append(shares, share)

			if len(shares) >= int(state.Threshold) && !test {
				return shares, nil
			}
		}

		// Check if all paths have been tried
		if len(triedPaths) >= len(paths) || test {
			break
		}

		// Wait before retrying remaining paths
		log.Warn().
			Int("have", len(shares)).
			Int("need", int(state.Threshold)).
			Dur("wait", retryDuration).
			Msg("Not enough shares retrieved. Waiting before retrying.")
		time.Sleep(retryDuration)
	}

	if len(shares) >= int(state.Threshold) {
		return shares, nil
	}

	return nil, fmt.Errorf(
		"tried all paths, could not retrieve enough valid shares: have %d, need %d",
		len(shares),
		state.Threshold,
	)
}
