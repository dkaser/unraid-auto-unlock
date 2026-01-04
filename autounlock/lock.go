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
	"os"
	"syscall"

	"github.com/dkaser/unraid-auto-unlock/autounlock/constants"
)

func lockApp() (*os.File, error) {
	file, err := os.OpenFile(constants.LockFile, os.O_CREATE|os.O_RDWR, constants.LockFileMode)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		file.Close()

		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	return file, nil
}
