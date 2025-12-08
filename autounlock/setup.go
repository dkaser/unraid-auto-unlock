package main

import (
	"encoding/base64"
	"fmt"

	"github.com/dkaser/unraid-auto-unlock/autounlock/encryption"
	"github.com/dkaser/unraid-auto-unlock/autounlock/secrets"
	"github.com/dkaser/unraid-auto-unlock/autounlock/state"
	"github.com/dkaser/unraid-auto-unlock/autounlock/unraid"
	"github.com/rs/zerolog/log"
)

func (a *AutoUnlock) Setup() error {
	err := unraid.TestKeyfile(a.args.KeyFile)
	if err != nil {
		return fmt.Errorf("keyfile test failed: %w", err)
	}

	log.Info().Msg("Keyfile test succeeded")

	secret, err := secrets.CreateSecret(a.args.Setup.Threshold, a.args.Setup.Shares)
	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	err = state.WriteStateToFile(
		a.fs,
		secret.VerificationKey,
		secret.SigningKey,
		a.args.State,
		a.args.Setup.Threshold,
	)
	if err != nil {
		return fmt.Errorf("failed to write state to file: %w", err)
	}

	log.Info().Str("state", a.args.State).Msg("Wrote state")

	err = encryption.EncryptFile(
		a.fs,
		a.args.KeyFile,
		a.args.EncryptedFile,
		secret.Secret,
		secret.VerificationKey,
	)
	if err != nil {
		return fmt.Errorf("failed to encrypt file: %w", err)
	}

	a.RemoveKeyfile()

	log.Info().
		Str("keyfile", a.args.KeyFile).
		Str("encryptedfile", a.args.EncryptedFile).
		Msg("Encrypted file")

	// Output the threshold and shares
	fmt.Printf("Total Shares: %d\n", a.args.Setup.Shares)
	fmt.Printf("Unlock Threshold: %d\n\n", a.args.Setup.Threshold)

	fmt.Println("Share values (base64 encoded):")

	// Output each share as base64, one per line
	for _, share := range secret.Shares {
		shareB64 := base64.StdEncoding.EncodeToString(share)
		fmt.Println(shareB64)
	}

	return nil
}
