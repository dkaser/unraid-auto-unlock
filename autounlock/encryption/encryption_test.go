package encryption

// Testing objectives:
// - Verify that the trimKey function correctly trims or errors based on key length.
// - Ensure EncryptFile handles file reading/writing errors appropriately.
// - Confirm that EncryptFile successfully encrypts data with valid inputs.
// - Confirm that DecryptFile handles file reading/writing errors appropriately.
// - Ensure DecryptFile successfully decrypts data with valid inputs.
// - Ensure that DecryptFile results in the original data after encryption and decryption.
// - Test encryption with different nonce sizes
// - Test that encrypted files include padding to obscure length
// - Test decryption with wrong key fails
// - Test decryption with wrong nonce fails
// - Test round-trip encryption/decryption with various data types and sizes
// - Test that ciphertext is different with different nonces

import (
	"bytes"
	"fmt"
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

func TestEncryptFile_DifferentNoncesSamePlaintext(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	key := make([]byte, 32)
	nonce1 := []byte("nonce1-12345")
	nonce2 := []byte("nonce2-12345")
	plaintext := []byte("same plaintext")

	afero.WriteFile(fs, "/input.txt", plaintext, 0o644)

	// Encrypt with nonce1
	err := svc.EncryptFile("/input.txt", "/encrypted1.enc", key, nonce1)
	if err != nil {
		t.Fatalf("encryption with nonce1 failed: %v", err)
	}

	// Encrypt with nonce2
	err = svc.EncryptFile("/input.txt", "/encrypted2.enc", key, nonce2)
	if err != nil {
		t.Fatalf("encryption with nonce2 failed: %v", err)
	}

	// Ciphertexts should be different
	ciphertext1, _ := afero.ReadFile(fs, "/encrypted1.enc")
	ciphertext2, _ := afero.ReadFile(fs, "/encrypted2.enc")

	if bytes.Equal(ciphertext1, ciphertext2) {
		t.Error("ciphertexts with different nonces should be different")
	}
}

func TestDecryptFile_WrongNonce(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	key := make([]byte, 32)
	nonce1 := []byte("nonce1-12345")
	nonce2 := []byte("nonce2-12345")
	plaintext := []byte("secret message")

	afero.WriteFile(fs, "/input.txt", plaintext, 0o644)

	// Encrypt with nonce1
	err := svc.EncryptFile("/input.txt", "/encrypted.enc", key, nonce1)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	// Try to decrypt with nonce2 - should fail
	err = svc.DecryptFile("/encrypted.enc", "/decrypted.txt", key, nonce2)
	if err == nil {
		t.Error("expected error when decrypting with wrong nonce")
	}
}

func TestEncryptFile_ShortNonce(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	key := make([]byte, 32)
	shortNonce := make([]byte, 6) // Too short
	plaintext := []byte("test data")

	afero.WriteFile(fs, "/input.txt", plaintext, 0o644)

	err := svc.EncryptFile("/input.txt", "/encrypted.enc", key, shortNonce)
	if err == nil {
		t.Error("expected error for short nonce")
	}
}

func TestDecryptFile_ShortNonce(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	key := make([]byte, 32)
	shortNonce := make([]byte, 6) // Too short

	afero.WriteFile(fs, "/encrypted.enc", []byte("fake ciphertext"), 0o644)

	err := svc.DecryptFile("/encrypted.enc", "/output.txt", key, shortNonce)
	if err == nil {
		t.Error("expected error for short nonce")
	}
}

func TestEncryptFile_LongNonceIsTrimmed(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	key := make([]byte, 32)

	longNonce := make([]byte, 24) // Longer than needed, should be trimmed
	for i := range longNonce {
		longNonce[i] = byte(i)
	}

	plaintext := []byte("test data")

	afero.WriteFile(fs, "/input.txt", plaintext, 0o644)

	err := svc.EncryptFile("/input.txt", "/encrypted.enc", key, longNonce)
	if err != nil {
		t.Fatalf("encryption should succeed with long nonce (trimmed): %v", err)
	}

	// Verify we can decrypt using the same long nonce (trimmed to same value)
	err = svc.DecryptFile("/encrypted.enc", "/decrypted.txt", key, longNonce)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	result, _ := afero.ReadFile(fs, "/decrypted.txt")
	if !bytes.Equal(result, plaintext) {
		t.Error("decrypted data doesn't match original")
	}
}

func TestEncryptFile_IncludesPadding(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	key := make([]byte, 32)
	nonce := make([]byte, 12)

	// Encrypt same plaintext multiple times to verify padding varies
	plaintext := []byte("test data")
	ciphertextLengths := make(map[int]bool)

	// Run multiple encryptions
	for i := range 10 {
		inputPath := fmt.Sprintf("/input_%d", i)
		encPath := fmt.Sprintf("/enc_%d", i)

		afero.WriteFile(fs, inputPath, plaintext, 0o644)

		err := svc.EncryptFile(inputPath, encPath, key, nonce)
		if err != nil {
			t.Fatalf("encryption failed: %v", err)
		}

		ciphertext, _ := afero.ReadFile(fs, encPath)
		ciphertextLengths[len(ciphertext)] = true
	}

	// Verify that we got different ciphertext lengths (due to random padding)
	// With 10 encryptions and random padding, we should see variation
	if len(ciphertextLengths) < 9 {
		t.Errorf(
			"expected different ciphertext lengths due to random padding, got only %d unique length(s)",
			len(ciphertextLengths),
		)
	}

	// Also verify ciphertexts are substantially larger than plaintext
	for length := range ciphertextLengths {
		if length <= len(plaintext)+50 {
			t.Errorf(
				"ciphertext length %d should be substantially larger than plaintext length %d",
				length,
				len(plaintext),
			)
		}
	}
}

func TestEncryptDecrypt_EmptyData(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	key := make([]byte, 32)
	nonce := make([]byte, 12)

	// Test that empty data can be encrypted and decrypted
	afero.WriteFile(fs, "/empty.txt", []byte{}, 0o644)

	err := svc.EncryptFile("/empty.txt", "/encrypted.enc", key, nonce)
	if err != nil {
		t.Fatalf("encryption of empty data failed: %v", err)
	}

	err = svc.DecryptFile("/encrypted.enc", "/decrypted.txt", key, nonce)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	result, _ := afero.ReadFile(fs, "/decrypted.txt")
	if len(result) != 0 {
		t.Errorf("expected empty result, got %d bytes", len(result))
	}
}

func TestDecryptFile_CorruptedEnvelope(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	key := make([]byte, 32)
	nonce := make([]byte, 12)

	// Create a valid encryption then corrupt it by changing the ciphertext
	// in a way that makes the JSON invalid after decryption
	plaintext := []byte("test")
	afero.WriteFile(fs, "/input.txt", plaintext, 0o644)

	err := svc.EncryptFile("/input.txt", "/encrypted.enc", key, nonce)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	// Read the encrypted file and flip a bit
	ciphertext, _ := afero.ReadFile(fs, "/encrypted.enc")
	if len(ciphertext) > 10 {
		ciphertext[5] ^= 0xFF // Corrupt a byte
		afero.WriteFile(fs, "/corrupted.enc", ciphertext, 0o644)

		err = svc.DecryptFile("/corrupted.enc", "/decrypted.txt", key, nonce)
		if err == nil {
			t.Error("expected error when decrypting corrupted data")
		}
	} else {
		t.Errorf("ciphertext too short: %d bytes", len(ciphertext))
	}
}
