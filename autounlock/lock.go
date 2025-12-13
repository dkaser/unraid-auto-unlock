package main

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
