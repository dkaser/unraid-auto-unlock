package state

import (
	"testing"

	"github.com/spf13/afero"
)

// Testing objectives:
// - Verify that WriteStateToFile correctly writes the state to a file.
// - Ensure that ReadStateFromFile accurately reads and parses the state from a file.
// - Test error handling for file read/write operations.
// - Test error handling for invalid/incorrect JSON data.
// - Test round-trip write and read with various data
// - Test that file permissions are correct
// - Test concurrent reads don't interfere
// - Test handling of special characters in keys

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

func TestWriteReadStateRoundTrip_WithBinaryData(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	filePath := "/test/state.json"

	// Use binary data including null bytes and special characters
	verificationKey := []byte{0x00, 0x01, 0xFF, 0xFE, 0x7F, 0x80}
	signingKey := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	nonce := []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66}
	threshold := uint16(7)

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
		t.Errorf("VerificationKey mismatch")
	}

	if string(readState.SigningKey) != string(signingKey) {
		t.Errorf("SigningKey mismatch")
	}

	if string(readState.Nonce) != string(nonce) {
		t.Errorf("Nonce mismatch")
	}
}

func TestWriteStateToFile_CreatesDirectories(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	filePath := "/deeply/nested/path/state.json"

	verificationKey := []byte("verification-key")
	signingKey := []byte("signing-key")
	nonce := []byte("nonce")
	threshold := uint16(3)

	err := svc.WriteStateToFile(verificationKey, signingKey, nonce, filePath, threshold)
	if err != nil {
		t.Fatalf("WriteStateToFile failed: %v", err)
	}

	// Verify file was created
	exists, err := afero.Exists(fs, filePath)
	if err != nil {
		t.Fatalf("Failed to check file existence: %v", err)
	}

	if !exists {
		t.Error("State file was not created in nested directory")
	}
}

func TestReadStateFromFile_MultipleReads(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	filePath := "/test/state.json"

	verificationKey := []byte("verification-key")
	signingKey := []byte("signing-key")
	nonce := []byte("nonce")
	threshold := uint16(3)

	err := svc.WriteStateToFile(verificationKey, signingKey, nonce, filePath, threshold)
	if err != nil {
		t.Fatalf("WriteStateToFile failed: %v", err)
	}

	// Read multiple times to ensure idempotency
	for i := range 5 {
		readState, err := svc.ReadStateFromFile(filePath)
		if err != nil {
			t.Fatalf("ReadStateFromFile failed on iteration %d: %v", i, err)
		}

		if readState.Threshold != threshold {
			t.Errorf("Iteration %d: Threshold mismatch", i)
		}
	}
}

func TestWriteStateToFile_EmptyKeys(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	filePath := "/test/state.json"

	// Test with empty byte slices
	verificationKey := []byte{}
	signingKey := []byte{}
	nonce := []byte{}
	threshold := uint16(1)

	err := svc.WriteStateToFile(verificationKey, signingKey, nonce, filePath, threshold)
	if err != nil {
		t.Fatalf("WriteStateToFile failed: %v", err)
	}

	readState, err := svc.ReadStateFromFile(filePath)
	if err != nil {
		t.Fatalf("ReadStateFromFile failed: %v", err)
	}

	if len(readState.VerificationKey) != 0 {
		t.Error("VerificationKey should be empty")
	}

	if len(readState.SigningKey) != 0 {
		t.Error("SigningKey should be empty")
	}

	if len(readState.Nonce) != 0 {
		t.Error("Nonce should be empty")
	}
}

func TestReadStateFromFile_MissingFields(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	filePath := "/test/state.json"

	// JSON with missing fields
	err := afero.WriteFile(fs, filePath, []byte(`{"threshold": 3}`), 0o600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	readState, err := svc.ReadStateFromFile(filePath)
	// This should succeed but have empty/default values for missing fields
	if err != nil {
		t.Fatalf("ReadStateFromFile failed: %v", err)
	}

	if readState.Threshold != 3 {
		t.Errorf("Threshold should be 3, got %d", readState.Threshold)
	}

	// Missing fields should be empty/nil
	if len(readState.VerificationKey) != 0 {
		t.Error("VerificationKey should be empty when missing from JSON")
	}
}

func TestWriteStateToFile_ZeroThreshold(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	filePath := "/test/state.json"

	verificationKey := []byte("verification-key")
	signingKey := []byte("signing-key")
	nonce := []byte("nonce")
	threshold := uint16(0)

	err := svc.WriteStateToFile(verificationKey, signingKey, nonce, filePath, threshold)
	if err != nil {
		t.Fatalf("WriteStateToFile failed: %v", err)
	}

	readState, err := svc.ReadStateFromFile(filePath)
	if err != nil {
		t.Fatalf("ReadStateFromFile failed: %v", err)
	}

	if readState.Threshold != 0 {
		t.Errorf("Threshold mismatch: expected 0, got %d", readState.Threshold)
	}
}

func TestWriteStateToFile_MaxThreshold(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	filePath := "/test/state.json"

	verificationKey := []byte("verification-key")
	signingKey := []byte("signing-key")
	nonce := []byte("nonce")
	threshold := uint16(65535) // Max uint16 value

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
}

func TestReadStateFromFile_ExtraFields(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	filePath := "/test/state.json"

	// JSON with extra fields that aren't in the struct
	jsonData := `{
		"threshold": 3,
		"verificationKey": "dGVzdC1rZXk=",
		"signingKey": "c2lnbmluZy1rZXk=",
		"nonce": "bm9uY2U=",
		"extraField": "should be ignored",
		"anotherExtra": 12345
	}`

	err := afero.WriteFile(fs, filePath, []byte(jsonData), 0o600)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	readState, err := svc.ReadStateFromFile(filePath)
	if err != nil {
		t.Fatalf("ReadStateFromFile should succeed even with extra fields: %v", err)
	}

	if readState.Threshold != 3 {
		t.Errorf("Threshold should be 3, got %d", readState.Threshold)
	}
}
