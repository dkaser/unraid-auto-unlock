package main

import (
	"os"

	"github.com/alexflint/go-arg"
	"github.com/dkaser/unraid-auto-unlock/autounlock/version"
	"github.com/rs/zerolog/log"
	"github.com/spf13/afero"
)

type AutoUnlock struct {
	fs   afero.Fs
	args CmdArgs
}

func NewAutoUnlock(fs afero.Fs, args CmdArgs) *AutoUnlock {
	return &AutoUnlock{
		fs:   fs,
		args: args,
	}
}

func main() {
	var args CmdArgs

	fs := afero.NewOsFs()

	parser := arg.MustParse(&args)
	if parser.Subcommand() == nil {
		parser.WriteHelp(os.Stdout)
		os.Exit(1)
	}

	autoUnlock := NewAutoUnlock(fs, args)

	autoUnlock.InitializeLogging()

	version.OutputToDebug()

	err := autoUnlock.Prechecks()
	if err != nil {
		log.Error().Stack().Err(err).Msg("Prechecks failed")
		os.Exit(1)
	}

	switch {
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
		log.Error().Stack().Err(err).Msg("Failed to execute command")
		os.Exit(1)
	}
}
