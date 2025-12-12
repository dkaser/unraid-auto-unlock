package main

import (
	"fmt"

	"github.com/dkaser/unraid-auto-unlock/autounlock/encryption"
	"github.com/dkaser/unraid-auto-unlock/autounlock/secrets"
	"github.com/dkaser/unraid-auto-unlock/autounlock/state"
	"github.com/dkaser/unraid-auto-unlock/autounlock/unraid"
	"github.com/dkaser/unraid-auto-unlock/autounlock/version"
	"github.com/spf13/afero"
)

// AutoUnlock manages the auto-unlock operations.
type AutoUnlock struct {
	fs         afero.Fs
	args       CmdArgs
	unraid     *unraid.Service
	encryption *encryption.Service
	state      *state.Service
	secrets    *secrets.Service
}

// NewAutoUnlock creates a new AutoUnlock instance.
func NewAutoUnlock(fs afero.Fs, args CmdArgs) (*AutoUnlock, error) {
	// Create new instance
	autoUnlock := &AutoUnlock{
		fs:         fs,
		args:       args,
		unraid:     unraid.NewService(fs),
		encryption: encryption.NewService(fs),
		state:      state.NewService(fs),
		secrets:    secrets.NewService(fs),
	}

	autoUnlock.InitializeLogging()

	version.OutputToDebug()

	err := autoUnlock.Prechecks()
	if err != nil {
		return nil, fmt.Errorf("prechecks failed: %w", err)
	}

	return autoUnlock, nil
}
