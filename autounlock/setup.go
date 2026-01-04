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
	"encoding/base64"
	"fmt"

	"github.com/rs/zerolog/log"
)

// Setup configures the auto-unlock system.
func (a *AutoUnlock) Setup() error {
	err := a.unraid.TestKeyfile(a.args.KeyFile)
	if err != nil {
		return fmt.Errorf("keyfile test failed: %w", err)
	}

	log.Info().Msg("Keyfile test succeeded")

	secret, err := a.secrets.CreateSecret(a.args.Setup.Threshold, a.args.Setup.Shares)
	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	err = a.state.WriteStateToFile(
		secret.VerificationKey,
		secret.SigningKey,
		secret.Nonce,
		a.args.State,
		a.args.Setup.Threshold,
	)
	if err != nil {
		return fmt.Errorf("failed to write state to file: %w", err)
	}

	log.Info().Str("state", a.args.State).Msg("Wrote state")

	err = a.encryption.EncryptFile(
		a.args.KeyFile,
		a.args.EncryptedFile,
		secret.Secret,
		secret.Nonce,
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
