package main

import (
	"fmt"

	"github.com/dkaser/unraid-auto-unlock/autounlock/version"
	"github.com/spf13/afero"
)

type AutoUnlock struct {
	fs   afero.Fs
	args CmdArgs
}

func NewAutoUnlock(fs afero.Fs, args CmdArgs) (*AutoUnlock, error) {
	// Create new instance
	autoUnlock := AutoUnlock{
		fs:   fs,
		args: args,
	}

	autoUnlock.InitializeLogging()

	version.OutputToDebug()

	err := autoUnlock.Prechecks()
	if err != nil {
		return nil, fmt.Errorf("prechecks failed: %w", err)
	}

	return &autoUnlock, nil
}
