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
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/dkaser/unraid-auto-unlock/autounlock/secrets"
	"github.com/manifoldco/promptui"
	"github.com/rclone/rclone/fs/config/obscure"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"golang.org/x/term"
)

func (a *AutoUnlock) ObscureSecretFromStdin() error {
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return fmt.Errorf("failed to read secret from stdin: %w", scanner.Err())
	}

	secret := scanner.Text()

	obscured, err := obscure.Obscure(secret)
	if err != nil {
		return fmt.Errorf("failed to obscure secret: %w", err)
	}

	fmt.Println(obscured)

	return nil
}

// InitializeLogging sets up the logging configuration.
func (a *AutoUnlock) InitializeLogging() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if term.IsTerminal(int(os.Stdout.Fd())) || a.args.Pretty {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:     os.Stderr,
			NoColor: !term.IsTerminal(int(os.Stderr.Fd())),
		})
	}

	// File to enable debug mode for testing/startup
	_, err := os.Stat("/boot/config/plugins/auto-unlock/debug")
	if err == nil {
		a.args.Debug = true
	}

	if a.args.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Debug logging enabled")
	}
}

// Prechecks verifies the system is ready for auto-unlock operations.
func (a *AutoUnlock) Prechecks() error {
	if !a.unraid.IsUnraid() {
		return errors.New("not running on Unraid")
	}

	err := a.unraid.WaitForVarIni()
	if err != nil {
		return fmt.Errorf("failed to wait for var.ini: %w", err)
	}

	return nil
}

// RemoveKeyfile safely removes the keyfile from the filesystem.
func (a *AutoUnlock) RemoveKeyfile() {
	// Remove keyfile
	err := a.fs.Remove(a.args.KeyFile)
	if errors.Is(err, afero.ErrFileNotFound) {
		log.Debug().Str("keyfile", a.args.KeyFile).Msg("Keyfile already removed")

		return
	}

	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to remove keyfile")

		return
	}

	log.Info().Str("keyfile", a.args.KeyFile).Msg("Removed keyfile")
}

// TestPath tests access to a given path.
func (a *AutoUnlock) TestPath() error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(a.args.TestPath.ServerTimeout)*time.Second,
	)
	defer cancel()

	shareStr, err := secrets.FetchShare(ctx, a.args.TestPath.Path)
	if err != nil {
		return fmt.Errorf("failed to fetch share: %w", err)
	}

	log.Info().Msg("Retrieved share from remote server")

	appState, err := a.state.ReadStateFromFile(a.args.State)
	if err != nil {
		return fmt.Errorf("failed to read state from file: %w", err)
	}

	_, err = a.secrets.GetShare(shareStr, appState.SigningKey)
	if err != nil {
		return fmt.Errorf("failed to decode/verify share: %w", err)
	}

	log.Info().Msg("Successfully retrieved and verified share")

	return nil
}

// ResetConfiguration resets the auto-unlock configuration.
func (a *AutoUnlock) ResetConfiguration() error {
	if !a.args.Reset.Force {
		prompt := promptui.Prompt{
			Label:     "Are you sure you want to reset the auto-unlock configuration? This will delete the state and encrypted files",
			IsConfirm: true,
			Default:   "N",
		}

		_, err := prompt.Run()
		if err != nil {
			fmt.Println("Reset cancelled.")

			return nil //nolint:nilerr
		}
	}

	files := []string{a.args.State, a.args.EncryptedFile, a.args.Config}
	for _, file := range files {
		err := a.safeRemoveFile(file)
		if err != nil {
			return fmt.Errorf("failed to remove %s: %w", file, err)
		}
	}

	return nil
}

func (a *AutoUnlock) safeRemoveFile(file string) error {
	err := a.fs.Remove(file)
	if errors.Is(err, afero.ErrFileNotFound) {
		log.Debug().Str("file", file).Msg("File already removed")

		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to remove file %s: %w", file, err)
	}

	log.Info().Str("file", file).Msg("Removed file")

	return nil
}
