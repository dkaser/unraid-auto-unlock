package unraid

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
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
)

// TestKeyfile, StartArray, and WaitForArrayStarted will not be tested here because they are
// highly dependent on the Unraid environment and system state.

// Testing objectives:
// - Verify that IsUnraid correctly identifies an Unraid environment.
// - Verify that IsUnraid returns false for non-Unraid environments.
// - Verify that WaitForVarIni correctly waits for /var/local/emhttp/var.ini to be available.
// - Verify that GetFsState correctly reads the fsState from var.ini.
// - Verify that GetFsState returns an error if var.ini cannot be read or if fsState is not defined
// - Verify that VerifyArrayStatus correctly checks the array status.
// - Test GetFsState with various fsState values
// - Test WaitForVarIni timeout behavior
// - Test GetFsState with malformed var.ini
// - Test VerifyArrayStatus with empty fsState
// - Test var.ini parsing with multiple fields

func TestIsUnraid_True(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	_ = afero.WriteFile(fs, "/etc/unraid-version", []byte("6.12.0"), 0o644)

	if !svc.IsUnraid() {
		t.Error("IsUnraid should return true when /etc/unraid-version exists")
	}
}

func TestIsUnraid_False(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)

	if svc.IsUnraid() {
		t.Error("IsUnraid should return false when /etc/unraid-version does not exist")
	}
}

func TestGetFsState_Success(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	varIniContent := `fsState=Started
mdState=STARTED
`
	_ = afero.WriteFile(fs, "/var/local/emhttp/var.ini", []byte(varIniContent), 0o644)

	fsState, err := svc.GetFsState()
	if err != nil {
		t.Errorf("GetFsState should not return error: %v", err)
	}

	if fsState != "Started" {
		t.Errorf("GetFsState should return 'Started', got '%s'", fsState)
	}
}

func TestGetFsState_FileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)

	_, err := svc.GetFsState()
	if err == nil {
		t.Error("GetFsState should return error when var.ini does not exist")
	}
}

func TestGetFsState_NoFsState(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	varIniContent := `mdState=STARTED
`
	_ = afero.WriteFile(fs, "/var/local/emhttp/var.ini", []byte(varIniContent), 0o644)

	fsState, err := svc.GetFsState()
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
	svc := NewService(fs)
	varIniContent := `fsState=Started
`
	_ = afero.WriteFile(fs, "/var/local/emhttp/var.ini", []byte(varIniContent), 0o644)

	if !svc.VerifyArrayStatus("Started") {
		t.Error("VerifyArrayStatus should return true when status matches")
	}
}

func TestVerifyArrayStatus_CaseInsensitive(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	varIniContent := `fsState=Started
`
	_ = afero.WriteFile(fs, "/var/local/emhttp/var.ini", []byte(varIniContent), 0o644)

	if !svc.VerifyArrayStatus("started") {
		t.Error("VerifyArrayStatus should be case insensitive")
	}
}

func TestVerifyArrayStatus_NoMatch(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	varIniContent := `fsState=Stopped
`
	_ = afero.WriteFile(fs, "/var/local/emhttp/var.ini", []byte(varIniContent), 0o644)

	if svc.VerifyArrayStatus("Started") {
		t.Error("VerifyArrayStatus should return false when status does not match")
	}
}

func TestVerifyArrayStatus_FileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)

	if svc.VerifyArrayStatus("Started") {
		t.Error("VerifyArrayStatus should return false when var.ini does not exist")
	}
}

func TestWaitForVarIni_AlreadyReady(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	varIniContent := `fsState=Stopped
`
	_ = afero.WriteFile(fs, "/var/local/emhttp/var.ini", []byte(varIniContent), 0o644)

	err := svc.WaitForVarIni()
	if err != nil {
		t.Errorf("WaitForVarIni should not return error when var.ini is ready: %v", err)
	}
}

func TestGetFsState_VariousStates(t *testing.T) {
	testCases := []struct {
		name          string
		varIniContent string
		expectedState string
	}{
		{
			name:          "Started state",
			varIniContent: "fsState=Started\n",
			expectedState: "Started",
		},
		{
			name:          "Stopped state",
			varIniContent: "fsState=Stopped\n",
			expectedState: "Stopped",
		},
		{
			name: "State with other fields",
			varIniContent: `mdState=STARTED
fsState=Started
otherField=value
`,
			expectedState: "Started",
		},
		{
			name: "State at end of file",
			varIniContent: `field1=value1
field2=value2
fsState=Stopped
`,
			expectedState: "Stopped",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			svc := NewService(fs)
			_ = afero.WriteFile(fs, "/var/local/emhttp/var.ini", []byte(tc.varIniContent), 0o644)

			fsState, err := svc.GetFsState()
			if err != nil {
				t.Errorf("GetFsState should not return error: %v", err)
			}

			if fsState != tc.expectedState {
				t.Errorf("Expected fsState '%s', got '%s'", tc.expectedState, fsState)
			}
		})
	}
}

func TestGetFsState_MalformedIni(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)

	// INI parser is generally forgiving, so this tests various edge cases
	testCases := []struct {
		name    string
		content string
	}{
		{
			name:    "No equals sign",
			content: "fsStateStarted\n",
		},
		{
			name:    "Multiple equals",
			content: "fsState=Started=Extra\n",
		},
		{
			name:    "Only whitespace",
			content: "   \n\t\n",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(_ *testing.T) {
			_ = afero.WriteFile(fs, "/var/local/emhttp/var.ini", []byte(tc.content), 0o644)

			// These should not panic or crash
			_, _ = svc.GetFsState()
		})
	}
}

func TestVerifyArrayStatus_EmptyFsState(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	varIniContent := `otherField=value
`
	_ = afero.WriteFile(fs, "/var/local/emhttp/var.ini", []byte(varIniContent), 0o644)

	// When fsState is empty, it should not match "Started"
	if svc.VerifyArrayStatus("Started") {
		t.Error("VerifyArrayStatus should return false for empty fsState")
	}

	// Should match empty string (case insensitive)
	if !svc.VerifyArrayStatus("") {
		t.Error("VerifyArrayStatus should return true for empty string when fsState is empty")
	}
}

func TestVerifyArrayStatus_VariousCases(t *testing.T) {
	testCases := []struct {
		name           string
		varIniContent  string
		checkStatus    string
		expectedResult bool
	}{
		{
			name:           "Exact match lowercase",
			varIniContent:  "fsState=started\n",
			checkStatus:    "started",
			expectedResult: true,
		},
		{
			name:           "Exact match uppercase",
			varIniContent:  "fsState=STARTED\n",
			checkStatus:    "STARTED",
			expectedResult: true,
		},
		{
			name:           "Mixed case var.ini, lowercase check",
			varIniContent:  "fsState=StArTeD\n",
			checkStatus:    "started",
			expectedResult: true,
		},
		{
			name:           "Different status",
			varIniContent:  "fsState=Started\n",
			checkStatus:    "Stopped",
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			svc := NewService(fs)
			_ = afero.WriteFile(fs, "/var/local/emhttp/var.ini", []byte(tc.varIniContent), 0o644)

			result := svc.VerifyArrayStatus(tc.checkStatus)
			if result != tc.expectedResult {
				t.Errorf("Expected %v, got %v", tc.expectedResult, result)
			}
		})
	}
}

func TestWaitForVarIni_FileAppearsLater(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)

	// Create file in a goroutine after a short delay
	go func() {
		time.Sleep(15 * time.Second)

		varIniContent := `fsState=Started
`
		_ = afero.WriteFile(fs, "/var/local/emhttp/var.ini", []byte(varIniContent), 0o644)
	}()

	err := svc.WaitForVarIni()
	if err != nil {
		t.Errorf("WaitForVarIni should succeed when file appears: %v", err)
	}
}

func TestWaitForVarIni_EmptyFileBecomesValid(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)

	// Create empty file first
	_ = afero.WriteFile(fs, "/var/local/emhttp/var.ini", []byte(""), 0o644)

	// Update file with valid content in background
	go func() {
		time.Sleep(15 * time.Second)

		varIniContent := `fsState=Started
`
		_ = afero.WriteFile(fs, "/var/local/emhttp/var.ini", []byte(varIniContent), 0o644)
	}()

	err := svc.WaitForVarIni()
	if err != nil {
		t.Errorf("WaitForVarIni should succeed when file becomes valid: %v", err)
	}
}

func TestGetFsState_WithWhitespace(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)

	testCases := []struct {
		name          string
		varIniContent string
		expectedState string
	}{
		{
			name:          "Leading spaces trimmed by INI parser",
			varIniContent: "fsState=  Started\n",
			expectedState: "Started", // INI library trims whitespace
		},
		{
			name:          "Trailing spaces trimmed by INI parser",
			varIniContent: "fsState=Started  \n",
			expectedState: "Started", // INI library trims whitespace
		},
		{
			name:          "No newline at end",
			varIniContent: "fsState=Started",
			expectedState: "Started",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_ = afero.WriteFile(fs, "/var/local/emhttp/var.ini", []byte(tc.varIniContent), 0o644)

			fsState, err := svc.GetFsState()
			if err != nil {
				t.Errorf("GetFsState should not return error: %v", err)
			}

			if fsState != tc.expectedState {
				t.Errorf("Expected fsState '%s', got '%s'", tc.expectedState, fsState)
			}
		})
	}
}

func TestIsUnraid_WithDifferentVersionContent(t *testing.T) {
	testCases := []struct {
		name    string
		content string
	}{
		{"Version 6.12", "6.12.0"},
		{"Version 7.0", "7.0.0"},
		{"Empty version file", ""},
		{"Multi-line content", "6.12.0\nBuild 2023.01.01\n"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			svc := NewService(fs)
			_ = afero.WriteFile(fs, "/etc/unraid-version", []byte(tc.content), 0o644)

			if !svc.IsUnraid() {
				t.Error(
					"IsUnraid should return true when /etc/unraid-version exists regardless of content",
				)
			}
		})
	}
}

func TestGetFsState_LargeFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)

	// Create a large var.ini with many fields
	content := ""

	var contentSb413 strings.Builder
	for i := range 100 {
		// Use valid key names (alphanumeric)
		fmt.Fprintf(&contentSb413, "field%d=value%d\n", i, i)
	}

	content += contentSb413.String()

	content += "fsState=Started\n"

	var contentSb418 strings.Builder
	for i := 100; i < 200; i++ {
		fmt.Fprintf(&contentSb418, "field%d=value%d\n", i, i)
	}

	content += contentSb418.String()

	_ = afero.WriteFile(fs, "/var/local/emhttp/var.ini", []byte(content), 0o644)

	fsState, err := svc.GetFsState()
	if err != nil {
		t.Errorf("GetFsState should handle large files: %v", err)
	}

	if fsState != "Started" {
		t.Errorf("Expected 'Started', got '%s'", fsState)
	}
}

// Full sample lsblk output from a real Unraid system with the array started.
// Used by multiple ParseLUKSDevices tests.
const lsblkFullSample = `{
   "blockdevices": [
      {"name": "/dev/loop0", "fstype": "squashfs", "type": "loop"},
      {"name": "/dev/loop1", "fstype": "squashfs", "type": "loop"},
      {"name": "/dev/loop2", "fstype": "btrfs",    "type": "loop"},
      {"name": "/dev/sda",  "fstype": null, "type": "disk",
         "children": [{"name": "/dev/sda1", "fstype": "vfat", "type": "part"}]},
      {"name": "/dev/sdb",  "fstype": null, "type": "disk"},
      {"name": "/dev/sdc",  "fstype": null, "type": "disk"},
      {"name": "/dev/sdd",  "fstype": null, "type": "disk"},
      {"name": "/dev/sde",  "fstype": null, "type": "disk"},
      {"name": "/dev/sdf",  "fstype": null, "type": "disk",
         "children": [{"name": "/dev/sdf1", "fstype": "crypto_LUKS", "type": "part",
            "children": [{"name": "/dev/mapper/sdf1", "fstype": "zfs_member", "type": "crypt"}]}]},
      {"name": "/dev/sdg",  "fstype": "zfs_member", "type": "disk",
         "children": [{"name": "/dev/sdg1", "fstype": "zfs_member", "type": "part"}]},
      {"name": "/dev/sdh",  "fstype": null, "type": "disk",
         "children": [{"name": "/dev/sdh1", "fstype": "crypto_LUKS", "type": "part"}]},
      {"name": "/dev/sdi",  "fstype": null, "type": "disk",
         "children": [{"name": "/dev/sdi1", "fstype": "crypto_LUKS", "type": "part"}]},
      {"name": "/dev/sdj",  "fstype": null, "type": "disk",
         "children": [{"name": "/dev/sdj1", "fstype": "crypto_LUKS", "type": "part"}]},
      {"name": "/dev/sdk",  "fstype": null, "type": "disk",
         "children": [{"name": "/dev/sdk1", "fstype": "crypto_LUKS", "type": "part"}]},
      {"name": "/dev/md1p1", "fstype": null, "type": "md",
         "children": [{"name": "/dev/mapper/md1p1", "fstype": "xfs", "type": "crypt"}]},
      {"name": "/dev/md2p1", "fstype": null, "type": "md",
         "children": [{"name": "/dev/mapper/md2p1", "fstype": "xfs", "type": "crypt"}]},
      {"name": "/dev/md3p1", "fstype": null, "type": "md",
         "children": [{"name": "/dev/mapper/md3p1", "fstype": "xfs", "type": "crypt"}]},
      {"name": "/dev/zram0", "fstype": null, "type": "disk"},
      {"name": "/dev/nvme1n1", "fstype": null, "type": "disk",
         "children": [{"name": "/dev/nvme1n1p1", "fstype": "crypto_LUKS", "type": "part",
            "children": [{"name": "/dev/mapper/nvme1n1p1", "fstype": "zfs_member", "type": "crypt"}]}]},
      {"name": "/dev/nvme2n1", "fstype": null, "type": "disk",
         "children": [{"name": "/dev/nvme2n1p1", "fstype": "crypto_LUKS", "type": "part",
            "children": [{"name": "/dev/mapper/nvme2n1p1", "fstype": "zfs_member", "type": "crypt"}]}]},
      {"name": "/dev/nvme0n1", "fstype": null, "type": "disk",
         "children": [{"name": "/dev/nvme0n1p1", "fstype": "crypto_LUKS", "type": "part",
            "children": [{"name": "/dev/mapper/nvme0n1p1", "fstype": "zfs_member", "type": "crypt"}]}]}
   ]
}`

func TestParseLUKSDevices_FullSample(t *testing.T) {
	want := []string{
		"/dev/sdf1",
		"/dev/sdh1",
		"/dev/sdi1",
		"/dev/sdj1",
		"/dev/sdk1",
		"/dev/md1p1",
		"/dev/md2p1",
		"/dev/md3p1",
		"/dev/nvme1n1p1",
		"/dev/nvme2n1p1",
		"/dev/nvme0n1p1",
	}

	got, err := ParseLUKSDevices([]byte(lsblkFullSample))
	if err != nil {
		t.Fatalf("ParseLUKSDevices returned unexpected error: %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("expected %d devices, got %d: %v", len(want), len(got), got)
	}

	wantSet := make(map[string]struct{}, len(want))
	for _, d := range want {
		wantSet[d] = struct{}{}
	}

	for _, d := range got {
		if _, ok := wantSet[d]; !ok {
			t.Errorf("unexpected device in result: %s", d)
		}
	}
}

func TestParseLUKSDevices_NoLUKS(t *testing.T) {
	input := `{"blockdevices": [
		{"name": "/dev/sda", "fstype": null, "type": "disk",
			"children": [{"name": "/dev/sda1", "fstype": "vfat", "type": "part"}]},
		{"name": "/dev/sdb", "fstype": "xfs",  "type": "disk"}
	]}`

	got, err := ParseLUKSDevices([]byte(input))
	if err != nil {
		t.Fatalf("ParseLUKSDevices returned unexpected error: %v", err)
	}

	if len(got) != 0 {
		t.Errorf("expected no devices, got: %v", got)
	}
}

func TestParseLUKSDevices_CryptChildOnly(t *testing.T) {
	// Device has no crypto_LUKS fstype but has a crypt-type child (array already started).
	input := `{"blockdevices": [
		{"name": "/dev/md1p1", "fstype": null, "type": "md",
			"children": [{"name": "/dev/mapper/md1p1", "fstype": "xfs", "type": "crypt"}]}
	]}`

	got, err := ParseLUKSDevices([]byte(input))
	if err != nil {
		t.Fatalf("ParseLUKSDevices returned unexpected error: %v", err)
	}

	if len(got) != 1 || got[0] != "/dev/md1p1" {
		t.Errorf("expected [/dev/md1p1], got: %v", got)
	}
}

func TestParseLUKSDevices_Deduplication(t *testing.T) {
	// Device matches both criteria: fstype=crypto_LUKS AND has a crypt child.
	input := `{"blockdevices": [
		{"name": "/dev/sdf1", "fstype": "crypto_LUKS", "type": "part",
			"children": [{"name": "/dev/mapper/sdf1", "fstype": "zfs_member", "type": "crypt"}]}
	]}`

	got, err := ParseLUKSDevices([]byte(input))
	if err != nil {
		t.Fatalf("ParseLUKSDevices returned unexpected error: %v", err)
	}

	if len(got) != 1 {
		t.Errorf("expected exactly 1 device (deduplication), got %d: %v", len(got), got)
	}
}

func TestParseLUKSDevices_MalformedJSON(t *testing.T) {
	_, err := ParseLUKSDevices([]byte(`{not valid json`))
	if err == nil {
		t.Error("ParseLUKSDevices should return an error for malformed JSON")
	}
}
