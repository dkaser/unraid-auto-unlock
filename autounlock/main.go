package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/dkaser/unraid-auto-unlock/autounlock/version"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/term"
)

var args struct {
	Setup         bool   `arg:"--setup"         help:"Setup mode"`
	Test          bool   `arg:"--test"          help:"Run in test mode"`
	TestPath      string `arg:"--test-path"     help:"URI to use in test mode"`
	Threshold     uint16 `arg:"--threshold"     default:"3"                                           help:"Threshold for setup mode"`
	Shares        uint16 `arg:"--shares"        default:"5"                                           help:"Number of shares to split into"`
	Config        string `arg:"--config"        default:"/boot/config/plugins/auto-unlock/config.txt" help:"Path to config file"`
	State         string `arg:"--state"         default:"/boot/config/plugins/auto-unlock/state.json" help:"Path to state file"`
	KeyFile       string `arg:"--keyfile"       default:"/root/keyfile"                               help:"Path to file to encrypt"`
	EncryptedFile string `arg:"--encryptedfile" default:"/boot/config/plugins/auto-unlock/unlock.enc" help:"Path to output encrypted file"`
	RetryDelay    uint16 `arg:"--retry-delay"   default:"60"                                          help:"Delay between retries in seconds"`
	Debug         bool   `arg:"--debug"         help:"Enable debug logging"`
	Pretty        bool   `arg:"--pretty"        help:"Enable pretty logging output"`
	Version       bool   `arg:"--version"       help:"Show version information"`
}

func main() {
	err := arg.Parse(&args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse arguments: %v\n", err)
		os.Exit(1)
	}

	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	if term.IsTerminal(int(os.Stdout.Fd())) || args.Pretty {
		log.Logger = log.Output(zerolog.ConsoleWriter{
			Out:     os.Stderr,
			NoColor: !term.IsTerminal(int(os.Stderr.Fd())),
		})
	}

	// File to enable debug mode for testing/startup
	_, err = os.Stat("/boot/config/plugins/auto-unlock/debug")
	if err == nil {
		args.Debug = true
	}

	if args.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Debug logging enabled")
	}

	if !IsUnraid() {
		log.Error().Msg("This program can only be run on Unraid OS")
		os.Exit(1)
	}

	err = WaitForVarIni()
	if err != nil {
		log.Error().Stack().Err(err).Msg("emhttp initialization timeout")
		os.Exit(1)
	}

	switch {
	case args.Setup:
		err = Setup()
	case args.TestPath != "":
		err = TestPath()
	case args.Version:
		fmt.Print(version.BuildInfoString())
	default:
		err = Unlock()
	}

	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to execute command")
		os.Exit(1)
	}
}

func Setup() error {
	err := TestKeyfile()
	if err != nil {
		return fmt.Errorf("keyfile test failed: %w", err)
	}

	log.Info().Msg("Keyfile test succeeded")

	secret, err := CreateSecret(args.Threshold, args.Shares)
	if err != nil {
		return fmt.Errorf("failed to create secret: %w", err)
	}

	err = WriteStateToFile(secret, args.State, args.Threshold)
	if err != nil {
		return fmt.Errorf("failed to write state to file: %w", err)
	}

	log.Info().Str("state", args.State).Msg("Wrote state")

	err = EncryptFile(args.KeyFile, args.EncryptedFile, secret.Secret, secret.VerificationKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt file: %w", err)
	}

	err = os.Remove(args.KeyFile)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to remove key file")
	}

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

func Unlock() error {
	if !VerifyArrayStatus("Stopped") && !args.Test {
		return errors.New("array is not stopped, cannot unlock")
	}

	sharePaths, err := readPathsFromFile(args.Config)
	if err != nil {
		return fmt.Errorf("failed to read paths from config file: %w", err)
	}

	state, err := ReadStateFromFile(args.State)
	if err != nil {
		return fmt.Errorf("failed to read state from file: %w", err)
	}

	shares, err := GetShares(sharePaths, state, args.RetryDelay, args.Test)
	if err != nil {
		return fmt.Errorf("failed to get shares: %w", err)
	}

	secret, err := CombineSecret(shares)
	if err != nil {
		return fmt.Errorf("failed to combine secret: %w", err)
	}

	err = DecryptFile(args.EncryptedFile, args.KeyFile, secret, state.VerificationKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt file: %w", err)
	}

	defer RemoveKeyfile()

	log.Info().
		Str("encryptedfile", args.EncryptedFile).
		Str("keyfile", args.KeyFile).
		Msg("Decrypted file")

	if args.Test {
		err := TestKeyfile()
		if err != nil {
			return fmt.Errorf("keyfile test failed: %w", err)
		}

		log.Info().Msg("Keyfile test succeeded")

		return nil
	}

	err = StartArray()
	if err != nil {
		return fmt.Errorf("failed to start array: %w", err)
	}

	err = WaitForArrayStarted()
	if err != nil {
		return fmt.Errorf("failed to verify array started: %w", err)
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
	shareStr, err := FetchShare(context.Background(), args.TestPath)
	if err != nil {
		return fmt.Errorf("failed to fetch share: %w", err)
	}

	log.Info().Msg("Retrieved share from remote server")

	state, err := ReadStateFromFile(args.State)
	if err != nil {
		return fmt.Errorf("failed to read state from file: %w", err)
	}

	_, err = GetShare(shareStr, state.SigningKey)
	if err != nil {
		return fmt.Errorf("failed to decode/verify share: %w", err)
	}

	log.Info().Msg("Successfully retrieved and verified share")

	return nil
}
