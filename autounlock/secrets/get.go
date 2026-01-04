package secrets

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
	"bufio"
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bytemare/secret-sharing/keys"
	"github.com/rs/zerolog/log"

	_ "github.com/dkaser/unraid-auto-unlock/autounlock/secrets/awssecrets" // Register AWS fetchers
	_ "github.com/dkaser/unraid-auto-unlock/autounlock/secrets/dns"        // Register DNS fetcher
	_ "github.com/dkaser/unraid-auto-unlock/autounlock/secrets/http"       // Register HTTP fetcher
	_ "github.com/dkaser/unraid-auto-unlock/autounlock/secrets/rclone"     // Register Rclone fetcher
	"github.com/dkaser/unraid-auto-unlock/autounlock/secrets/registry"
	"github.com/dkaser/unraid-auto-unlock/autounlock/state"
)

type RetrievedShare struct {
	Share   *keys.KeyShare
	ShareID string
}

// FetchShare fetches a share from the specified path using the registry.
// Tries each registered fetcher in priority order until one matches.
func FetchShare(ctx context.Context, path string) (string, error) {
	registeredFetchers := registry.GetFetchers()

	// Try each registered fetcher in priority order.
	for _, fetcher := range registeredFetchers {
		if matched := fetcher.Match(path); matched {
			result, err := fetcher.Fetch(ctx, path)
			if err != nil {
				return "", fmt.Errorf("failed to fetch resource: %w", err)
			}

			return result, nil
		}
	}

	return "", fmt.Errorf("no fetcher available for path: %s", path)
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
