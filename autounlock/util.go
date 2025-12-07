package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/dkaser/unraid-auto-unlock/autounlock/secrets"
	"github.com/dkaser/unraid-auto-unlock/autounlock/state"
	"github.com/dkaser/unraid-auto-unlock/autounlock/unraid"
	"github.com/rclone/rclone/fs/config/obscure"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
	"golang.org/x/term"
)

func ObscureSecretFromStdin() error {
	var secret string

	_, err := fmt.Scanln(&secret)
	if err != nil {
		return fmt.Errorf("failed to read secret from stdin: %w", err)
	}

	obscured, err := obscure.Obscure(secret)
	if err != nil {
		return fmt.Errorf("failed to obscure secret: %w", err)
	}

	fmt.Println(obscured)

	return nil
}

func InitializeLogging() {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if term.IsTerminal(int(os.Stdout.Fd())) || args.Pretty {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:     os.Stderr,
			NoColor: !term.IsTerminal(int(os.Stderr.Fd())),
		})
	}

	// File to enable debug mode for testing/startup
	_, err := os.Stat("/boot/config/plugins/auto-unlock/debug")
	if err == nil {
		args.Debug = true
	}

	if args.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Debug logging enabled")
	}
}

func Prechecks(fs afero.Fs) error {
	if !unraid.IsUnraid(fs) {
		return errors.New("not running on Unraid")
	}

	err := unraid.WaitForVarIni(fs)
	if err != nil {
		return fmt.Errorf("failed to wait for var.ini: %w", err)
	}

	return nil
}

func RemoveKeyfile(fs afero.Fs) {
	// Remove keyfile
	err := fs.Remove(args.KeyFile)
	if errors.Is(err, afero.ErrFileNotFound) {
		log.Debug().Str("keyfile", args.KeyFile).Msg("Keyfile already removed")

		return
	}

	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to remove keyfile")

		return
	}

	log.Info().Str("keyfile", args.KeyFile).Msg("Removed keyfile")
}

func TestPath(fs afero.Fs) error {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(args.ServerTimeout)*time.Second,
	)
	defer cancel()

	shareStr, err := secrets.FetchShare(ctx, args.TestPath)
	if err != nil {
		return fmt.Errorf("failed to fetch share: %w", err)
	}

	log.Info().Msg("Retrieved share from remote server")

	appState, err := state.ReadStateFromFile(fs, args.State)
	if err != nil {
		return fmt.Errorf("failed to read state from file: %w", err)
	}

	_, err = secrets.GetShare(shareStr, appState.SigningKey)
	if err != nil {
		return fmt.Errorf("failed to decode/verify share: %w", err)
	}

	log.Info().Msg("Successfully retrieved and verified share")

	return nil
}
