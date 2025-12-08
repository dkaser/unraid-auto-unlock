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

func (a *AutoUnlock) Unlock() error {
	if !unraid.VerifyArrayStatus(a.fs, "Stopped") && !a.args.Unlock.Test {
		return errors.New("array is not stopped, cannot unlock")
	}

	state, err := state.ReadStateFromFile(a.fs, a.args.State)
	if err != nil {
		return fmt.Errorf("failed to read state from file: %w", err)
	}

	secret, err := a.retrieveSecret(state)
	if err != nil {
		return fmt.Errorf("failed to retrieve secret: %w", err)
	}

	err = encryption.DecryptFile(
		a.fs,
		a.args.EncryptedFile,
		a.args.KeyFile,
		secret,
		state.VerificationKey,
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
		err := unraid.TestKeyfile(a.args.KeyFile)
		if err != nil {
			return fmt.Errorf("keyfile test failed: %w", err)
		}

		log.Info().Msg("Keyfile test succeeded")

		return nil
	}

	err = unraid.StartArray(a.fs)
	if err != nil {
		return fmt.Errorf("failed to start array: %w", err)
	}

	err = unraid.WaitForArrayStarted(a.fs)
	if err != nil {
		return fmt.Errorf("failed to verify array started: %w", err)
	}

	return nil
}

func (a *AutoUnlock) retrieveSecret(state state.State) ([]byte, error) {
	sharePaths, err := secrets.ReadPathsFromFile(a.fs, a.args.Config)
	if err != nil {
		return nil, fmt.Errorf("failed to read paths from config file: %w", err)
	}

	shares, err := secrets.GetShares(
		a.fs,
		sharePaths,
		state,
		a.args.Unlock.RetryDelay,
		a.args.Unlock.ServerTimeout,
		a.args.Unlock.Test,
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
