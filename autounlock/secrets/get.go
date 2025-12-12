package secrets

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bytemare/secret-sharing/keys"
	"github.com/dkaser/unraid-auto-unlock/autounlock/state"
	_ "github.com/rclone/rclone/backend/all" // Import all rclone backends
	"github.com/rclone/rclone/fs"
	"github.com/rs/zerolog/log"
)

type RetrievedShare struct {
	Share   *keys.KeyShare
	ShareID string
}

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

// ReadPathsFromFile reads share paths from a configuration file.
func (s *Service) ReadPathsFromFile(filename string) ([]string, error) {
	file, err := s.fs.Open(filename)
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

func (s *Service) tryGetShare(
	path string,
	pathNum int,
	signingKey []byte,
	serverTimeout time.Duration,
) (RetrievedShare, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), serverTimeout)
	defer cancel()

	shareStr, err := FetchShare(ctx, path)
	if err != nil {
		log.Debug().Int("path", pathNum).Stack().Err(err).Msg("Failed to fetch share")

		return RetrievedShare{}, false, err
	}

	share, err := s.GetShare(shareStr, signingKey)
	if err != nil {
		log.Debug().Int("path", pathNum).Stack().Err(err).Msg("Failed to get share")

		return RetrievedShare{}, true, err
	}

	// Use share identifier to detect duplicates
	shareID := strconv.FormatUint(uint64(share.Identifier()), 10)

	log.Info().Int("path", pathNum).Msg("Successfully retrieved share")

	return RetrievedShare{
		Share:   share,
		ShareID: shareID,
	}, true, nil
}

//nolint:cyclop,funlen // Complexity and length inherent to share collection with retry logic
func (s *Service) collectShares(
	paths []string,
	appState state.State,
	retryDuration time.Duration,
	serverTimeout time.Duration,
	test bool,
	unraidSvc unraidVerifier,
) ([]*keys.KeyShare, error) {
	var (
		shares     []*keys.KeyShare
		mutex      sync.Mutex
		triedPaths = make(map[string]bool)
		seenShares = make(map[string]bool)
	)

	for {
		if shouldAbort(unraidSvc, test) {
			return nil, errors.New("array is no longer stopped, aborting share retrieval")
		}

		var waitGroup sync.WaitGroup

		for pathNum, path := range paths {
			// Skip paths we've already tried
			mutex.Lock()

			alreadyTried := triedPaths[path]

			mutex.Unlock()

			if alreadyTried {
				continue
			}

			waitGroup.Go(func() {
				retrievedShare, fetchSucceeded, err := s.tryGetShare(
					path,
					pathNum,
					appState.SigningKey,
					serverTimeout,
				)

				mutex.Lock()
				defer mutex.Unlock()

				// Only mark as tried if fetch succeeded (don't retry corrupt shares)
				if fetchSucceeded {
					triedPaths[path] = true
				}

				if err != nil {
					return
				}

				// Check for duplicate shares
				if seenShares[retrievedShare.ShareID] {
					log.Debug().Int("path", pathNum).Msg("Duplicate share, ignoring")

					return
				}

				shares = append(shares, retrievedShare.Share)
				seenShares[retrievedShare.ShareID] = true
			})
		}

		waitGroup.Wait()

		if len(shares) >= int(appState.Threshold) && !test {
			return shares, nil
		}

		// Check if all paths have been tried
		if len(triedPaths) >= len(paths) || test {
			break
		}

		// Wait before retrying remaining paths
		log.Warn().
			Int("have", len(shares)).
			Int("need", int(appState.Threshold)).
			Dur("wait", retryDuration).
			Msg("Not enough shares retrieved. Waiting before retrying.")
		time.Sleep(retryDuration)
	}

	return shares, nil
}

// GetShares retrieves shares from configured paths.
func (s *Service) GetShares(
	paths []string,
	appState state.State,
	retryInterval uint16,
	serverTimeout uint16,
	test bool,
	unraidSvc unraidVerifier,
) ([]*keys.KeyShare, error) {
	retryDuration := time.Duration(retryInterval) * time.Second
	serverTimeoutDuration := time.Duration(serverTimeout) * time.Second

	logSharePaths(paths)

	shares, err := s.collectShares(
		paths,
		appState,
		retryDuration,
		serverTimeoutDuration,
		test,
		unraidSvc,
	)
	if err != nil {
		return nil, err
	}

	if len(shares) >= int(appState.Threshold) {
		return shares, nil
	}

	return nil, fmt.Errorf(
		"tried all paths, could not retrieve enough valid shares: have %d, need %d",
		len(shares),
		appState.Threshold,
	)
}

func logSharePaths(paths []string) {
	for i, path := range paths {
		log.Debug().Int("path", i).Str("target", path).Msg("Configured share path")
	}
}

type unraidVerifier interface {
	VerifyArrayStatus(status string) bool
}

func shouldAbort(unraidSvc unraidVerifier, test bool) bool {
	if test || unraidSvc == nil {
		return false
	}

	return unraidSvc.VerifyArrayStatus("Started")
}
