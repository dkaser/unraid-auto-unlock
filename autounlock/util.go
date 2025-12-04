package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/dkaser/unraid-auto-unlock/autounlock/secrets"
	"github.com/dkaser/unraid-auto-unlock/autounlock/state"
	"github.com/dkaser/unraid-auto-unlock/autounlock/unraid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/term"
)

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

func Prechecks() error {
	if !unraid.IsUnraid() {
		return errors.New("not running on Unraid")
	}

	err := unraid.WaitForVarIni()
	if err != nil {
		return fmt.Errorf("failed to wait for var.ini: %w", err)
	}

	return nil
}

func RemoveKeyfile() {
	err := os.Remove(args.KeyFile)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to remove keyfile")
	}

	log.Info().Str("keyfile", args.KeyFile).Msg("Removed keyfile")
}

func TestPath() error {
	shareStr, err := secrets.FetchShare(context.Background(), args.TestPath)
	if err != nil {
		return fmt.Errorf("failed to fetch share: %w", err)
	}

	log.Info().Msg("Retrieved share from remote server")

	state, err := state.ReadStateFromFile(args.State)
	if err != nil {
		return fmt.Errorf("failed to read state from file: %w", err)
	}

	_, err = secrets.GetShare(shareStr, state.SigningKey)
	if err != nil {
		return fmt.Errorf("failed to decode/verify share: %w", err)
	}

	log.Info().Msg("Successfully retrieved and verified share")

	return nil
}
