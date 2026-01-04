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
