package state

import (
	"testing"

	"github.com/spf13/afero"
)

// Testing objectives:
// - Verify that WriteStateToFile correctly writes the state to a file.
// - Ensure that ReadStateFromFile accurately reads and parses the state from a file.
// - Test error handling for file read/write operations.
// Test error handling for invalid/incorrect JSON data.

func TestWriteStateToFile_WritesCorrectly(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	filePath := "/test/state.json"

	verificationKey := []byte("test-verification-key")
	signingKey := []byte("test-signing-key")
	nonce := []byte("test-nonce")
	threshold := uint16(3)

	err := svc.WriteStateToFile(verificationKey, signingKey, nonce, filePath, threshold)
	if err != nil {
		t.Fatalf("WriteStateToFile failed: %v", err)
	}

	// Verify file exists
	exists, err := afero.Exists(fs, filePath)
	if err != nil {
		t.Fatalf("Failed to check file existence: %v", err)
	}

	if !exists {
		t.Error("State file was not created")
	}

	// Verify content is valid JSON by reading it back
	content, err := afero.ReadFile(fs, filePath)
	if err != nil {
		t.Fatalf("Failed to read state file: %v", err)
	}

	if len(content) == 0 {
		t.Error("State file is empty")
	}
}

func TestReadStateFromFile_ReadsCorrectly(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	filePath := "/test/state.json"

	verificationKey := []byte("test-verification-key")
	signingKey := []byte("test-signing-key")
	nonce := []byte("test-nonce")
	threshold := uint16(3)

	err := svc.WriteStateToFile(verificationKey, signingKey, nonce, filePath, threshold)
	if err != nil {
		t.Fatalf("WriteStateToFile failed: %v", err)
	}

	readState, err := svc.ReadStateFromFile(filePath)
	if err != nil {
		t.Fatalf("ReadStateFromFile failed: %v", err)
	}

	if readState.Threshold != threshold {
		t.Errorf("Threshold mismatch: expected %d, got %d", threshold, readState.Threshold)
	}

	if string(readState.VerificationKey) != string(verificationKey) {
		t.Errorf(
			"VerificationKey mismatch: expected %s, got %s",
			verificationKey,
			readState.VerificationKey,
		)
	}

	if string(readState.SigningKey) != string(signingKey) {
		t.Errorf("SigningKey mismatch: expected %s, got %s", signingKey, readState.SigningKey)
	}
}

func TestReadStateFromFile_FileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	filePath := "/nonexistent/state.json"

	_, err := svc.ReadStateFromFile(filePath)
	if err == nil {
		t.Error("ReadStateFromFile should fail when file does not exist")
	}
}

func TestWriteStateToFile_InvalidPath(t *testing.T) {
	fs := afero.NewReadOnlyFs(afero.NewMemMapFs())
	svc := NewService(fs)
	filePath := "/readonly/state.json"

	err := svc.WriteStateToFile([]byte("key"), []byte("key"), []byte("key"), filePath, 3)
	if err == nil {
		t.Error("WriteStateToFile should fail on read-only filesystem")
	}
}

func TestReadStateFromFile_InvalidJSON(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	filePath := "/test/state.json"

	err := afero.WriteFile(fs, filePath, []byte("not valid json {{{"), 0o600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err = svc.ReadStateFromFile(filePath)
	if err == nil {
		t.Error("ReadStateFromFile should fail with invalid JSON")
	}
}

func TestReadStateFromFile_EmptyFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	filePath := "/test/state.json"

	err := afero.WriteFile(fs, filePath, []byte(""), 0o600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err = svc.ReadStateFromFile(filePath)
	if err == nil {
		t.Error("ReadStateFromFile should fail with empty file")
	}
}

func TestReadStateFromFile_PartialJSON(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	filePath := "/test/state.json"

	err := afero.WriteFile(fs, filePath, []byte(`{"threshold": 3, "signingKey":`), 0o600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err = svc.ReadStateFromFile(filePath)
	if err == nil {
		t.Error("ReadStateFromFile should fail with incomplete JSON")
	}
}

func TestReadStateFromFile_WrongJSONStructure(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	filePath := "/test/state.json"

	err := afero.WriteFile(
		fs,
		filePath,
		[]byte(`{"threshold": "not-a-number", "signingKey": "key"}`),
		0o600,
	)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	_, err = svc.ReadStateFromFile(filePath)
	if err == nil {
		t.Error("ReadStateFromFile should fail with wrong JSON types")
	}
}
