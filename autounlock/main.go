package main

import (
	"os"

	"github.com/alexflint/go-arg"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

func main() {
	var args CmdArgs

	fs := afero.NewOsFs()

	parser := arg.MustParse(&args)
	if parser.Subcommand() == nil {
		parser.WriteHelp(os.Stdout)
		os.Exit(1)
	}

	autoUnlock, err := NewAutoUnlock(fs, args)
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Failed to initialize AutoUnlock")
	}

	lockFile, err := lockApp()
	if err != nil {
		log.Fatal().Stack().Err(err).Msg("Another instance of the application is already running")
	}
	defer lockFile.Close()

	switch {
	case args.Reset != nil:
		err = autoUnlock.ResetConfiguration()
	case args.Obscure != nil:
		err = autoUnlock.ObscureSecretFromStdin()
	case args.Setup != nil:
		err = autoUnlock.Setup()
	case args.TestPath != nil:
		err = autoUnlock.TestPath()
	case args.Unlock != nil:
		err = autoUnlock.Unlock()
	}

	if err != nil {
		lockFile.Close()
		log.Fatal().Stack().Err(err).Msg("Failed to execute command") //nolint:gocritic
	}
}
