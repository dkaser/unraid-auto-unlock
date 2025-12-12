package encryption

// Testing objectives:
// - Verify that the trimKey function correctly trims or errors based on key length.
// - Ensure EncryptFile handles file reading/writing errors appropriately.
// - Confirm that EncryptFile successfully encrypts data with valid inputs.
// - Confirm that DecryptFile handles file reading/writing errors appropriately.
// - Ensure DecryptFile successfully decrypts data with valid inputs.
// - Ensure that DecryptFile results in the original data after encryption and decryption.

import (
	"bytes"
	"testing"

	"github.com/spf13/afero"
)

func TestTrimKey(t *testing.T) {
	tests := []struct {
		name      string
		key       []byte
		length    int
		wantLen   int
		wantError bool
	}{
		{
			name:      "exact length key",
			key:       make([]byte, 32),
			length:    32,
			wantLen:   32,
			wantError: false,
		},
		{
			name:      "longer key gets trimmed",
			key:       make([]byte, 64),
			length:    32,
			wantLen:   32,
			wantError: false,
		},
		{
			name:      "short key returns error",
			key:       make([]byte, 16),
			length:    32,
			wantLen:   0,
			wantError: true,
		},
		{
			name:      "empty key returns error",
			key:       []byte{},
			length:    32,
			wantLen:   0,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := trimKey(tt.key, tt.length)
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error, got nil")
				}

				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)

				return
			}

			if len(result) != tt.wantLen {
				t.Errorf("expected length %d, got %d", tt.wantLen, len(result))
			}
		})
	}
}

func TestEncryptFile_ReadError(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	key := make([]byte, 32)
	nonce := make([]byte, 12)

	err := svc.EncryptFile("/nonexistent", "/output", key, nonce)
	if err == nil {
		t.Error("expected error for nonexistent input file")
	}
}

func TestEncryptFile_WriteError(t *testing.T) {
	fs := afero.NewMemMapFs()
	key := make([]byte, 32)
	nonce := make([]byte, 12)

	// Create input file
	afero.WriteFile(fs, "/input.txt", []byte("test data"), 0o644)

	// Create read-only filesystem to simulate write error
	roFs := afero.NewReadOnlyFs(fs)
	svc := NewService(roFs)

	err := svc.EncryptFile("/input.txt", "/output", key, nonce)
	if err == nil {
		t.Error("expected error when writing to read-only filesystem")
	}
}

func TestEncryptFile_ShortKey(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	shortKey := make([]byte, 16) // Too short
	nonce := make([]byte, 12)

	afero.WriteFile(fs, "/input.txt", []byte("test data"), 0o644)

	err := svc.EncryptFile("/input.txt", "/output.enc", shortKey, nonce)
	if err == nil {
		t.Error("expected error for short key")
	}
}

func TestEncryptFile_Success(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	key := make([]byte, 32)
	nonce := make([]byte, 12)
	plaintext := []byte("hello world, this is test data!")

	afero.WriteFile(fs, "/input.txt", plaintext, 0o644)

	err := svc.EncryptFile("/input.txt", "/output.enc", key, nonce)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify output file exists and is different from input
	ciphertext, err := afero.ReadFile(fs, "/output.enc")
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	if bytes.Equal(ciphertext, plaintext) {
		t.Error("ciphertext should not equal plaintext")
	}
}

func TestDecryptFile_ReadError(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	key := make([]byte, 32)
	nonce := make([]byte, 12)

	err := svc.DecryptFile("/nonexistent", "/output", key, nonce)
	if err == nil {
		t.Error("expected error for nonexistent input file")
	}
}

func TestDecryptFile_WriteError(t *testing.T) {
	fs := afero.NewMemMapFs()
	key := make([]byte, 32)
	nonce := make([]byte, 12)
	plaintext := []byte("test data")

	// First encrypt to get valid ciphertext
	svc := NewService(fs)

	afero.WriteFile(fs, "/input.txt", plaintext, 0o644)
	svc.EncryptFile("/input.txt", "/encrypted.enc", key, nonce)

	// Use read-only filesystem to simulate write error
	roFs := afero.NewReadOnlyFs(fs)
	svcRO := NewService(roFs)

	err := svcRO.DecryptFile("/encrypted.enc", "/decrypted.txt", key, nonce)
	if err == nil {
		t.Error("expected error when writing to read-only filesystem")
	}
}

func TestDecryptFile_ShortKey(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	shortKey := make([]byte, 16) // Too short
	nonce := make([]byte, 12)

	afero.WriteFile(fs, "/encrypted.enc", []byte("fake ciphertext"), 0o644)

	err := svc.DecryptFile("/encrypted.enc", "/output.txt", shortKey, nonce)
	if err == nil {
		t.Error("expected error for short key")
	}
}

func TestDecryptFile_InvalidCiphertext(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	key := make([]byte, 32)
	nonce := make([]byte, 12)

	// Write invalid ciphertext
	afero.WriteFile(fs, "/invalid.enc", []byte("not valid ciphertext"), 0o644)

	err := svc.DecryptFile("/invalid.enc", "/output.txt", key, nonce)
	if err == nil {
		t.Error("expected error for invalid ciphertext")
	}
}

func TestDecryptFile_Success(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	key := make([]byte, 32)
	nonce := make([]byte, 12)
	plaintext := []byte("test data for decryption")

	// Encrypt first
	afero.WriteFile(fs, "/input.txt", plaintext, 0o644)

	err := svc.EncryptFile("/input.txt", "/encrypted.enc", key, nonce)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	// Then decrypt
	err = svc.DecryptFile("/encrypted.enc", "/decrypted.txt", key, nonce)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	// Verify decrypted content
	decrypted, err := afero.ReadFile(fs, "/decrypted.txt")
	if err != nil {
		t.Fatalf("failed to read decrypted file: %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("decrypted content mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	key := []byte("this-is-a-32-byte-key-for-test!!")
	nonce := []byte("12-byte-nonc")

	testCases := []struct {
		name string
		data []byte
	}{
		{"empty data", []byte{}},
		{"small data", []byte("hello")},
		{"medium data", []byte("this is some medium length test data for encryption")},
		{"binary data", []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			inputPath := "/input_" + tc.name
			encPath := "/enc_" + tc.name
			decPath := "/dec_" + tc.name

			afero.WriteFile(fs, inputPath, tc.data, 0o644)

			err := svc.EncryptFile(inputPath, encPath, key, nonce)
			if err != nil {
				t.Fatalf("encryption failed: %v", err)
			}

			err = svc.DecryptFile(encPath, decPath, key, nonce)
			if err != nil {
				t.Fatalf("decryption failed: %v", err)
			}

			result, _ := afero.ReadFile(fs, decPath)
			if !bytes.Equal(result, tc.data) {
				t.Errorf("round-trip failed: got %v, want %v", result, tc.data)
			}
		})
	}
}

func TestEncryptDecrypt_DifferentKeysOrNonces(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	key2[0] = 1 // Different key
	nonce := make([]byte, 12)

	plaintext := []byte("secret message")
	afero.WriteFile(fs, "/input.txt", plaintext, 0o644)

	// Encrypt with key1
	err := svc.EncryptFile("/input.txt", "/encrypted.enc", key1, nonce)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	// Try to decrypt with key2 - should fail
	err = svc.DecryptFile("/encrypted.enc", "/decrypted.txt", key2, nonce)
	if err == nil {
		t.Error("expected error when decrypting with wrong key")
	}
}
