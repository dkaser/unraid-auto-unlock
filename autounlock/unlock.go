package main

import (
	"errors"
	"fmt"

	"github.com/dkaser/unraid-auto-unlock/autounlock/encryption"
	"github.com/dkaser/unraid-auto-unlock/autounlock/secrets"
	"github.com/dkaser/unraid-auto-unlock/autounlock/state"
	"github.com/dkaser/unraid-auto-unlock/autounlock/unraid"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

func Unlock(fs afero.Fs) error {
	if !unraid.VerifyArrayStatus(fs, "Stopped") && !args.Test {
		return errors.New("array is not stopped, cannot unlock")
	}

	state, err := state.ReadStateFromFile(fs, args.State)
	if err != nil {
		return fmt.Errorf("failed to read state from file: %w", err)
	}

	secret, err := retrieveSecret(fs, state)
	if err != nil {
		return fmt.Errorf("failed to retrieve secret: %w", err)
	}

	err = encryption.DecryptFile(
		fs,
		args.EncryptedFile,
		args.KeyFile,
		secret,
		state.VerificationKey,
	)
	if err != nil {
		return fmt.Errorf("failed to decrypt file: %w", err)
	}

	defer RemoveKeyfile(fs)

	log.Info().
		Str("encryptedfile", args.EncryptedFile).
		Str("keyfile", args.KeyFile).
		Msg("Decrypted file")

	if args.Test {
		err := unraid.TestKeyfile(args.KeyFile)
		if err != nil {
			return fmt.Errorf("keyfile test failed: %w", err)
		}

		log.Info().Msg("Keyfile test succeeded")

		return nil
	}

	err = unraid.StartArray(fs)
	if err != nil {
		return fmt.Errorf("failed to start array: %w", err)
	}

	err = unraid.WaitForArrayStarted(fs)
	if err != nil {
		return fmt.Errorf("failed to verify array started: %w", err)
	}

	return nil
}

func retrieveSecret(fs afero.Fs, state state.State) ([]byte, error) {
	sharePaths, err := secrets.ReadPathsFromFile(fs, args.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to read paths from config file: %w", err)
	}

	shares, err := secrets.GetShares(
		fs,
		sharePaths,
		state,
		args.RetryDelay,
		args.ServerTimeout,
		args.Test,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get shares: %w", err)
	}

	secret, err := secrets.CombineSecret(shares)
	if err != nil {
		return nil, fmt.Errorf("failed to combine secret: %w", err)
	}

	return secret, nil
}
