package main

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
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

	if args.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
		log.Debug().Msg("Debug logging enabled")
	}

	switch {
	case args.Setup:
		Setup()
	case args.TestPath != "":
		TestPath()
	default:
		Unlock()
	}
}

func Setup() {
	err := TestKeyfile()
	if err != nil {
		log.Error().Stack().Err(err).Msg("Keyfile test failed")
		os.Exit(1)
	}

	log.Info().Msg("Keyfile test succeeded")

	secret, err := CreateSecret(args.Threshold, args.Shares)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to create secret")
		os.Exit(1)
	}

	err = WriteStateToFile(secret, args.State, args.Threshold)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to write state to file")
		os.Exit(1)
	}

	log.Info().Str("state", args.State).Msg("Wrote state")

	err = EncryptFile(args.KeyFile, args.EncryptedFile, secret.Secret, secret.VerificationKey)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to encrypt file")
		os.Exit(1)
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
}

func Unlock() {
	shardPaths, err := readPathsFromFile(args.Config)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to read paths from config file")
		os.Exit(1)
	}

	state, err := ReadStateFromFile(args.State)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to read state from file")
		os.Exit(1)
	}

	shares, err := GetShares(shardPaths, state, args.RetryDelay, args.Test)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to get shares")
		os.Exit(1)
	}

	secret, err := CombineSecret(shares)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to combine secret")
		os.Exit(1)
	}

	err = DecryptFile(args.EncryptedFile, args.KeyFile, secret, state.VerificationKey)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to decrypt file")
		os.Exit(1)
	}

	log.Info().
		Str("encryptedfile", args.EncryptedFile).
		Str("keyfile", args.KeyFile).
		Msg("Decrypted file")

	if args.Test {
		err := TestKeyfile()
		if err != nil {
			log.Error().Stack().Err(err).Msg("Keyfile test failed")
			os.Exit(1)
		}

		log.Info().Msg("Keyfile test succeeded")

		RemoveKeyfile()

		return
	}

	err = StartArray()
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to start array")
		RemoveKeyfile()
		os.Exit(1)
	}
}

func RemoveKeyfile() {
	err := os.Remove(args.KeyFile)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to remove keyfile")
	}

	log.Info().Str("keyfile", args.KeyFile).Msg("Removed keyfile")
}

func TestPath() {
	shareStr, err := FetchShare(context.Background(), args.TestPath)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to fetch share")
		os.Exit(1)
	}

	log.Info().Msg("Retrieved share from remote server")

	state, err := ReadStateFromFile(args.State)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to read state from file")
		os.Exit(1)
	}

	_, err = GetShare(shareStr, state.SigningKey)
	if err != nil {
		log.Error().Stack().Err(err).Msg("Failed to decode/verify share")
		os.Exit(1)
	}

	log.Info().Msg("Successfully retrieved and verified share")
}
