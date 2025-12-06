package unraid

import (
	"testing"

	"github.com/spf13/afero"
)

// TestKeyfile, StartArray, and WaitForArrayStarted will not be tested here because they are
// highly dependent on the Unraid environment and system state.

// Testing objectives:
// - Verify that IsUnraid correctly identifies an Unraid environment.
// - Verify that IsUnraid returns false for non-Unraid environments.
// - Verify that WaitForVarIni correctly waits for /boot/config/var.ini to be available.
// - Verify that GetFsState correctly reads the fsState from var.ini.
// - Verify that GetFsState returns an error if var.ini cannot be read or if fsState is not defined
// - Verify that VerifyArrayStatus correctly checks the array status.

func TestIsUnraid_True(t *testing.T) {
	fs := afero.NewMemMapFs()
	_ = afero.WriteFile(fs, "/etc/unraid-version", []byte("6.12.0"), 0o644)

	if !IsUnraid(fs) {
		t.Error("IsUnraid should return true when /etc/unraid-version exists")
	}
}

func TestIsUnraid_False(t *testing.T) {
	fs := afero.NewMemMapFs()

	if IsUnraid(fs) {
		t.Error("IsUnraid should return false when /etc/unraid-version does not exist")
	}
}

func TestGetFsState_Success(t *testing.T) {
	fs := afero.NewMemMapFs()
	varIniContent := `fsState=Started
mdState=STARTED
`
	_ = afero.WriteFile(fs, "/var/local/emhttp/var.ini", []byte(varIniContent), 0o644)

	fsState, err := GetFsState(fs)
	if err != nil {
		t.Errorf("GetFsState should not return error: %v", err)
	}

	if fsState != "Started" {
		t.Errorf("GetFsState should return 'Started', got '%s'", fsState)
	}
}

func TestGetFsState_FileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()

	_, err := GetFsState(fs)
	if err == nil {
		t.Error("GetFsState should return error when var.ini does not exist")
	}
}

func TestGetFsState_NoFsState(t *testing.T) {
	fs := afero.NewMemMapFs()
	varIniContent := `mdState=STARTED
`
	_ = afero.WriteFile(fs, "/var/local/emhttp/var.ini", []byte(varIniContent), 0o644)

	fsState, err := GetFsState(fs)
	if err != nil {
		t.Errorf("GetFsState should not return error: %v", err)
	}

	if fsState != "" {
		t.Errorf(
			"GetFsState should return empty string when fsState is not defined, got '%s'",
			fsState,
		)
	}
}

func TestVerifyArrayStatus_Match(t *testing.T) {
	fs := afero.NewMemMapFs()
	varIniContent := `fsState=Started
`
	_ = afero.WriteFile(fs, "/var/local/emhttp/var.ini", []byte(varIniContent), 0o644)

	if !VerifyArrayStatus(fs, "Started") {
		t.Error("VerifyArrayStatus should return true when status matches")
	}
}

func TestVerifyArrayStatus_CaseInsensitive(t *testing.T) {
	fs := afero.NewMemMapFs()
	varIniContent := `fsState=Started
`
	_ = afero.WriteFile(fs, "/var/local/emhttp/var.ini", []byte(varIniContent), 0o644)

	if !VerifyArrayStatus(fs, "started") {
		t.Error("VerifyArrayStatus should be case insensitive")
	}
}

func TestVerifyArrayStatus_NoMatch(t *testing.T) {
	fs := afero.NewMemMapFs()
	varIniContent := `fsState=Stopped
`
	_ = afero.WriteFile(fs, "/var/local/emhttp/var.ini", []byte(varIniContent), 0o644)

	if VerifyArrayStatus(fs, "Started") {
		t.Error("VerifyArrayStatus should return false when status does not match")
	}
}

func TestVerifyArrayStatus_FileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()

	if VerifyArrayStatus(fs, "Started") {
		t.Error("VerifyArrayStatus should return false when var.ini does not exist")
	}
}

func TestWaitForVarIni_AlreadyReady(t *testing.T) {
	fs := afero.NewMemMapFs()
	varIniContent := `fsState=Stopped
`
	_ = afero.WriteFile(fs, "/var/local/emhttp/var.ini", []byte(varIniContent), 0o644)

	err := WaitForVarIni(fs)
	if err != nil {
		t.Errorf("WaitForVarIni should not return error when var.ini is ready: %v", err)
	}
}
