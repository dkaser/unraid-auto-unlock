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
	"testing"

	"github.com/spf13/afero"
)

// Testing objectives:
// - Test RemoveKeyfile removes existing files
// - Test RemoveKeyfile handles missing files gracefully
// - Test RemoveKeyfile handles permission errors
// - Test safeRemoveFile with various scenarios
// - Test that InitializeLogging doesn't panic
// - Test Prechecks with Unraid and non-Unraid environments

func TestRemoveKeyfile_FileExists(t *testing.T) {
	fs := afero.NewMemMapFs()
	keyfilePath := "/root/keyfile"

	// Create a keyfile
	err := afero.WriteFile(fs, keyfilePath, []byte("test-key-data"), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test keyfile: %v", err)
	}

	args := CmdArgs{
		KeyFile: keyfilePath,
	}

	autoUnlock := &AutoUnlock{
		fs:   fs,
		args: args,
	}

	autoUnlock.RemoveKeyfile()

	// Verify file was removed
	exists, err := afero.Exists(fs, keyfilePath)
	if err != nil {
		t.Fatalf("Failed to check file existence: %v", err)
	}

	if exists {
		t.Error("Keyfile should have been removed")
	}
}

func TestRemoveKeyfile_FileMissing(_ *testing.T) {
	fs := afero.NewMemMapFs()
	keyfilePath := "/root/keyfile"

	args := CmdArgs{
		KeyFile: keyfilePath,
	}

	autoUnlock := &AutoUnlock{
		fs:   fs,
		args: args,
	}

	// Should not panic or error when file doesn't exist
	autoUnlock.RemoveKeyfile()
}

func TestRemoveKeyfile_MultipleCalls(t *testing.T) {
	fs := afero.NewMemMapFs()
	keyfilePath := "/root/keyfile"

	err := afero.WriteFile(fs, keyfilePath, []byte("test-key-data"), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test keyfile: %v", err)
	}

	args := CmdArgs{
		KeyFile: keyfilePath,
	}

	autoUnlock := &AutoUnlock{
		fs:   fs,
		args: args,
	}

	// Remove multiple times - should not panic
	autoUnlock.RemoveKeyfile()
	autoUnlock.RemoveKeyfile()
	autoUnlock.RemoveKeyfile()
}

func TestSafeRemoveFile_FileExists(t *testing.T) {
	fs := afero.NewMemMapFs()
	filePath := "/test/file.txt"

	err := afero.WriteFile(fs, filePath, []byte("test data"), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	autoUnlock := &AutoUnlock{
		fs: fs,
	}

	err = autoUnlock.safeRemoveFile(filePath)
	if err != nil {
		t.Errorf("safeRemoveFile should not return error: %v", err)
	}

	exists, _ := afero.Exists(fs, filePath)
	if exists {
		t.Error("File should have been removed")
	}
}

func TestSafeRemoveFile_FileMissing(t *testing.T) {
	fs := afero.NewMemMapFs()
	filePath := "/test/missing.txt"

	autoUnlock := &AutoUnlock{
		fs: fs,
	}

	err := autoUnlock.safeRemoveFile(filePath)
	if err != nil {
		t.Errorf("safeRemoveFile should not return error for missing file: %v", err)
	}
}

func TestSafeRemoveFile_ReadOnlyFS(t *testing.T) {
	fs := afero.NewMemMapFs()
	filePath := "/test/file.txt"

	err := afero.WriteFile(fs, filePath, []byte("test data"), 0o600)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Create read-only filesystem
	roFS := afero.NewReadOnlyFs(fs)

	autoUnlock := &AutoUnlock{
		fs: roFS,
	}

	err = autoUnlock.safeRemoveFile(filePath)
	if err == nil {
		t.Error("safeRemoveFile should return error for read-only filesystem")
	}
}

func TestInitializeLogging_DoesNotPanic(t *testing.T) {
	fs := afero.NewMemMapFs()

	args := CmdArgs{
		Debug:  false,
		Pretty: false,
	}

	autoUnlock := &AutoUnlock{
		fs:   fs,
		args: args,
	}

	// Should not panic
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("InitializeLogging panicked: %v", r)
		}
	}()

	autoUnlock.InitializeLogging()
}

func TestInitializeLogging_WithDebugFlag(t *testing.T) {
	fs := afero.NewMemMapFs()

	args := CmdArgs{
		Debug:  true,
		Pretty: false,
	}

	autoUnlock := &AutoUnlock{
		fs:   fs,
		args: args,
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("InitializeLogging panicked: %v", r)
		}
	}()

	autoUnlock.InitializeLogging()
}

func TestInitializeLogging_WithDebugFile(t *testing.T) {
	fs := afero.NewMemMapFs()

	// Create debug file
	err := afero.WriteFile(fs, "/boot/config/plugins/auto-unlock/debug", []byte(""), 0o644)
	if err != nil {
		t.Fatalf("Failed to create debug file: %v", err)
	}

	args := CmdArgs{
		Debug:  false,
		Pretty: false,
	}

	autoUnlock := &AutoUnlock{
		fs:   fs,
		args: args,
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("InitializeLogging panicked: %v", r)
		}
	}()

	autoUnlock.InitializeLogging()
}

func TestInitializeLogging_WithPrettyFlag(t *testing.T) {
	fs := afero.NewMemMapFs()

	args := CmdArgs{
		Debug:  false,
		Pretty: true,
	}

	autoUnlock := &AutoUnlock{
		fs:   fs,
		args: args,
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("InitializeLogging panicked: %v", r)
		}
	}()

	autoUnlock.InitializeLogging()
}
