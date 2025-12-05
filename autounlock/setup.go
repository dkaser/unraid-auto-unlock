package main

import (
	"encoding/base64"
	"fmt"

	"github.com/dkaser/unraid-auto-unlock/autounlock/encryption"
	"github.com/dkaser/unraid-auto-unlock/autounlock/secrets"
	"github.com/dkaser/unraid-auto-unlock/autounlock/state"
	"github.com/dkaser/unraid-auto-unlock/autounlock/unraid"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

func Setup(fs afero.Fs) error {
	err := unraid.TestKeyfile(args.KeyFile)
	if err != nil {
		return fmt.Errorf("keyfile test failed: %w", err)
	}

	log.Info().Msg("Keyfile test succeeded")

	secret, err := secrets.CreateSecret(args.Threshold, args.Shares)
	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	err = state.WriteStateToFile(
		fs,
		secret.VerificationKey,
		secret.SigningKey,
		args.State,
		args.Threshold,
	)
	if err != nil {
		return fmt.Errorf("failed to write state to file: %w", err)
	}

	log.Info().Str("state", args.State).Msg("Wrote state")

	err = encryption.EncryptFile(
		fs,
		args.KeyFile,
		args.EncryptedFile,
		secret.Secret,
		secret.VerificationKey,
	)
	if err != nil {
		return fmt.Errorf("failed to encrypt file: %w", err)
	}

	RemoveKeyfile(fs)

	log.Info().
		Str("keyfile", args.KeyFile).
		Str("encryptedfile", args.EncryptedFile).
		Msg("Encrypted file")

	// Output the threshold and shares
	fmt.Printf("Total Shares: %d\n", args.Shares)
	fmt.Printf("Unlock Threshold: %d\n\n", args.Threshold)

	fmt.Println("Share values (base64 encoded):")

	// Output each share as base64, one per line
	for _, share := range secret.Shares {
		shareB64 := base64.StdEncoding.EncodeToString(share)
		fmt.Println(shareB64)
	}

	return nil
}
