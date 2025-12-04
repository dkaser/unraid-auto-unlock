package main

import (
	"errors"
	"fmt"

	"github.com/dkaser/unraid-auto-unlock/autounlock/encryption"
	"github.com/dkaser/unraid-auto-unlock/autounlock/secrets"
	"github.com/dkaser/unraid-auto-unlock/autounlock/state"
	"github.com/dkaser/unraid-auto-unlock/autounlock/unraid"
	"github.com/rs/zerolog/log"
)

func Unlock() error {
	if !unraid.VerifyArrayStatus("Stopped") && !args.Test {
		return errors.New("array is not stopped, cannot unlock")
	}

	state, err := state.ReadStateFromFile(args.State)
	if err != nil {
		return fmt.Errorf("failed to read state from file: %w", err)
	}

	secret, err := retrieveSecret(state)
	if err != nil {
		return fmt.Errorf("failed to retrieve secret: %w", err)
	}

	err = encryption.DecryptFile(args.EncryptedFile, args.KeyFile, secret, state.VerificationKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt file: %w", err)
	}

	defer RemoveKeyfile()

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

	err = unraid.StartArray()
	if err != nil {
		return fmt.Errorf("failed to start array: %w", err)
	}

	err = unraid.WaitForArrayStarted()
	if err != nil {
		return fmt.Errorf("failed to verify array started: %w", err)
	}

	return nil
}

func retrieveSecret(state state.State) ([]byte, error) {
	sharePaths, err := secrets.ReadPathsFromFile(args.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to read paths from config file: %w", err)
	}

	shares, err := secrets.GetShares(
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
