package main

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
	"errors"
	"fmt"
	"time"

	"github.com/dkaser/unraid-auto-unlock/autounlock/constants"
	"github.com/dkaser/unraid-auto-unlock/autounlock/state"
	"github.com/rs/zerolog/log"
)

//nolint:cyclop,funlen // Unlock decrypts the keyfile and starts the array.
func (a *AutoUnlock) Unlock() error {
	if !a.args.Unlock.Test {
		started := a.unraid.VerifyArrayStatus("Started")
		if started {
			return errors.New("array is already started, aborting unlock")
		}

		err := a.unraid.WaitForArrayStatus("Stopped", constants.ArrayStatusTimeout)
		if err != nil {
			return fmt.Errorf("failed to verify array stopped: %w", err)
		}
	}

	state, err := a.state.ReadStateFromFile(a.args.State)
	if err != nil {
		return fmt.Errorf("failed to read state from file: %w", err)
	}

	secret, err := a.retrieveSecret(state)
	if err != nil {
		return fmt.Errorf("failed to retrieve secret: %w", err)
	}

	err = a.encryption.DecryptFile(
		a.args.EncryptedFile,
		a.args.KeyFile,
		secret,
		state.Nonce,
	)
	if err != nil {
		return fmt.Errorf("failed to decrypt file: %w", err)
	}

	defer a.RemoveKeyfile()

	log.Info().
		Str("encryptedfile", a.args.EncryptedFile).
		Str("keyfile", a.args.KeyFile).
		Msg("Decrypted file")

	if a.args.Unlock.Test {
		err := a.unraid.TestKeyfile(a.args.KeyFile)
		if err != nil {
			return fmt.Errorf("keyfile test failed: %w", err)
		}

		log.Info().Msg("Keyfile test succeeded")

		return nil
	}

	err = a.unraid.StartArray()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to start array, retrying in 30 seconds")
		time.Sleep(constants.StartRetryDelay)

		// Check if array is already started before retrying
		started := a.unraid.VerifyArrayStatus("Started")
		if started {
			log.Info().Msg("Array started successfully on its own")

			return nil
		}

		err = a.unraid.StartArray()
		if err != nil {
			return fmt.Errorf("failed to start array: %w", err)
		}
	}

	err = a.unraid.WaitForArrayStatus("Started", constants.ArrayTimeout)
	if err != nil {
		return fmt.Errorf("failed to verify array started: %w", err)
	}

	return nil
}

func (a *AutoUnlock) retrieveSecret(appState state.State) ([]byte, error) {
	sharePaths, err := a.secrets.ReadPathsFromFile(a.args.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to read paths from config file: %w", err)
	}

	shares, err := a.secrets.GetShares(
		sharePaths,
		appState,
		a.args.Unlock.RetryDelay,
		a.args.Unlock.ServerTimeout,
		a.args.Unlock.Test,
		a.unraid,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get shares: %w", err)
	}

	secret, err := a.secrets.CombineSecret(shares)
	if err != nil {
		return nil, fmt.Errorf("failed to combine secret: %w", err)
	}

	return secret, nil
}
