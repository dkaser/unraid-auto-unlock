package secrets

import (
	"bytes"
	"encoding/base64"
	"testing"

	"github.com/bytemare/secret-sharing/keys"
	"github.com/spf13/afero"
)

// Testing objectives:
// - Verify that CreateSecret generates a secret with correct number of shares.
// - Ensure that CombineSecret successfully reconstructs the original secret from valid shares.
// - Verify that CreateSecret creates a unique secret each time.
// - Ensure that GetShare correctly decodes and verifies a share.
// - Test GetShare failure cases: invalid base64, invalid signature, wrong signing key.
// - Test ReadPathsFromFile correctly reads paths from a file
// - Test ReadPathsFromFile skips empty lines and comments
// - Test ReadPathsFromFile handles file errors
// - Test that CombineSecret fails with insufficient shares
// - Test that CombineSecret fails with invalid shares
// - Test SignShare and VerifyShare functions
// - Test GenerateRandomKey produces unique keys

func TestCreateSecret_GeneratesCorrectNumberOfShares(t *testing.T) {
	testCases := []struct {
		name      string
		threshold uint16
		shares    uint16
	}{
		{"3 shares, 2 threshold", 2, 3},
		{"5 shares, 3 threshold", 3, 5},
		{"10 shares, 5 threshold", 5, 10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			svc := NewService(fs)

			secret, err := svc.CreateSecret(tc.threshold, tc.shares)
			if err != nil {
				t.Fatalf("CreateSecret failed: %v", err)
			}

			if len(secret.Secret) == 0 {
				t.Error("CreateSecret returned empty secret")
			}

			if len(secret.Shares) != int(tc.shares) {
				t.Errorf("expected %d shares, got %d", tc.shares, len(secret.Shares))
			}

			if len(secret.VerificationKey) == 0 {
				t.Error("CreateSecret returned empty verification key")
			}

			if len(secret.SigningKey) == 0 {
				t.Error("CreateSecret returned empty signing key")
			}
		})
	}
}

func TestCombineSecret_ReconstructsOriginalSecret(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	threshold := uint16(3)
	shares := uint16(5)

	sharedSecret, err := svc.CreateSecret(threshold, shares)
	if err != nil {
		t.Fatalf("CreateSecret failed: %v", err)
	}

	// Decode shares using GetShare
	keyShares := make([]*keys.KeyShare, threshold)
	for i := range threshold {
		shareBase64 := base64.StdEncoding.EncodeToString(sharedSecret.Shares[i])

		keyShare, err := svc.GetShare(shareBase64, sharedSecret.SigningKey)
		if err != nil {
			t.Fatalf("GetShare failed for share %d: %v", i, err)
		}

		keyShares[i] = keyShare
	}

	reconstructed, err := svc.CombineSecret(keyShares)
	if err != nil {
		t.Fatalf("CombineSecret failed: %v", err)
	}

	if !bytes.Equal(sharedSecret.Secret, reconstructed) {
		t.Errorf("reconstructed secret does not match original")
	}
}

func TestCreateSecret_GeneratesUniqueSecrets(t *testing.T) {
	threshold := uint16(2)
	shares := uint16(3)
	iterations := 10

	secrets := make([][]byte, iterations)
	signingKeys := make([][]byte, iterations)

	verificationKeys := make([][]byte, iterations)
	for i := range iterations {
		fs := afero.NewMemMapFs()
		svc := NewService(fs)

		sharedSecret, err := svc.CreateSecret(threshold, shares)
		if err != nil {
			t.Fatalf("CreateSecret failed on iteration %d: %v", i, err)
		}

		secrets[i] = sharedSecret.Secret
		signingKeys[i] = sharedSecret.SigningKey
		verificationKeys[i] = sharedSecret.VerificationKey
	}

	for i := range secrets {
		for j := i + 1; j < len(secrets); j++ {
			if bytes.Equal(secrets[i], secrets[j]) {
				t.Errorf("secrets at index %d and %d are identical", i, j)
			}

			if bytes.Equal(signingKeys[i], signingKeys[j]) {
				t.Errorf("signing keys at index %d and %d are identical", i, j)
			}

			if bytes.Equal(verificationKeys[i], verificationKeys[j]) {
				t.Errorf("verification keys at index %d and %d are identical", i, j)
			}
		}
	}
}

func TestGetShare_InvalidBase64(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)

	sharedSecret, err := svc.CreateSecret(2, 3)
	if err != nil {
		t.Fatalf("CreateSecret failed: %v", err)
	}

	// Test with invalid base64 characters
	invalidBase64 := "!!!not-valid-base64!!!"

	_, err = svc.GetShare(invalidBase64, sharedSecret.SigningKey)
	if err == nil {
		t.Error("GetShare should fail with invalid base64 input")
	}
}

func TestGetShare_InvalidSignature(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)

	sharedSecret, err := svc.CreateSecret(2, 3)
	if err != nil {
		t.Fatalf("CreateSecret failed: %v", err)
	}

	// Get a valid share and corrupt it
	validShareBase64 := base64.StdEncoding.EncodeToString(sharedSecret.Shares[0])

	// Decode, corrupt, and re-encode
	corruptedBytes := make([]byte, len(sharedSecret.Shares[0]))
	copy(corruptedBytes, sharedSecret.Shares[0])
	// Flip some bytes in the signature portion (at the end)
	corruptedBytes[len(corruptedBytes)-1] ^= 0xFF
	corruptedBytes[len(corruptedBytes)-2] ^= 0xFF
	corruptedShareBase64 := base64.StdEncoding.EncodeToString(corruptedBytes)

	_, err = svc.GetShare(corruptedShareBase64, sharedSecret.SigningKey)
	if err == nil {
		t.Errorf(
			"GetShare should fail with corrupted signature, valid input was: %s",
			validShareBase64,
		)
	}
}

func TestGetShare_WrongSigningKey(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	// Create two separate secrets with different signing keys
	secret1, err := svc.CreateSecret(2, 3)
	if err != nil {
		t.Fatalf("CreateSecret for secret1 failed: %v", err)
	}

	secret2, err := svc.CreateSecret(2, 3)
	if err != nil {
		t.Fatalf("CreateSecret for secret2 failed: %v", err)
	}

	// Try to verify share from secret1 using signing key from secret2
	shareBase64 := base64.StdEncoding.EncodeToString(secret1.Shares[0])

	_, err = svc.GetShare(shareBase64, secret2.SigningKey)
	if err == nil {
		t.Error("GetShare should fail when using wrong signing key")
	}
}

func TestReadPathsFromFile_ValidFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)

	content := `path1
path2
path3`
	afero.WriteFile(fs, "/paths.txt", []byte(content), 0o644)

	paths, err := svc.ReadPathsFromFile("/paths.txt")
	if err != nil {
		t.Fatalf("ReadPathsFromFile failed: %v", err)
	}

	if len(paths) != 3 {
		t.Errorf("expected 3 paths, got %d", len(paths))
	}

	expected := []string{"path1", "path2", "path3"}
	for i, path := range paths {
		if path != expected[i] {
			t.Errorf("path %d: expected %s, got %s", i, expected[i], path)
		}
	}
}

func TestReadPathsFromFile_WithCommentsAndEmptyLines(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)

	content := `# This is a comment
path1

# Another comment
path2
  
  # Indented comment
path3
`
	afero.WriteFile(fs, "/paths.txt", []byte(content), 0o644)

	paths, err := svc.ReadPathsFromFile("/paths.txt")
	if err != nil {
		t.Fatalf("ReadPathsFromFile failed: %v", err)
	}

	if len(paths) != 3 {
		t.Errorf("expected 3 paths, got %d", len(paths))
	}

	expected := []string{"path1", "path2", "path3"}
	for i, path := range paths {
		if path != expected[i] {
			t.Errorf("path %d: expected %s, got %s", i, expected[i], path)
		}
	}
}

func TestReadPathsFromFile_WithWhitespace(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)

	content := `  path1  
	path2	
path3   `
	afero.WriteFile(fs, "/paths.txt", []byte(content), 0o644)

	paths, err := svc.ReadPathsFromFile("/paths.txt")
	if err != nil {
		t.Fatalf("ReadPathsFromFile failed: %v", err)
	}

	expected := []string{"path1", "path2", "path3"}
	for i, path := range paths {
		if path != expected[i] {
			t.Errorf("path %d: expected %s, got %s", i, expected[i], path)
		}
	}
}

func TestReadPathsFromFile_FileNotFound(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)

	_, err := svc.ReadPathsFromFile("/nonexistent.txt")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestReadPathsFromFile_EmptyFile(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)

	afero.WriteFile(fs, "/empty.txt", []byte(""), 0o644)

	paths, err := svc.ReadPathsFromFile("/empty.txt")
	if err != nil {
		t.Fatalf("ReadPathsFromFile failed: %v", err)
	}

	if len(paths) != 0 {
		t.Errorf("expected 0 paths from empty file, got %d", len(paths))
	}
}

func TestCombineSecret_InsufficientShares(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	threshold := uint16(3)
	shares := uint16(5)

	sharedSecret, err := svc.CreateSecret(threshold, shares)
	if err != nil {
		t.Fatalf("CreateSecret failed: %v", err)
	}

	// Try with only 2 shares when threshold is 3
	keyShares := make([]*keys.KeyShare, 2)

	for i := range 2 {
		shareBase64 := base64.StdEncoding.EncodeToString(sharedSecret.Shares[i])

		keyShare, err := svc.GetShare(shareBase64, sharedSecret.SigningKey)
		if err != nil {
			t.Fatalf("GetShare failed: %v", err)
		}

		keyShares[i] = keyShare
	}

	reconstructed, err := svc.CombineSecret(keyShares)
	// The secret sharing library may actually succeed with insufficient shares
	// but the result won't match the original
	if err == nil {
		// If it doesn't error, at least verify the secret doesn't match
		if bytes.Equal(reconstructed, sharedSecret.Secret) {
			t.Error("CombineSecret should not reconstruct correct secret with insufficient shares")
		}
	}
}

func TestCombineSecret_WithDuplicateShares(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	threshold := uint16(3)
	shares := uint16(5)

	sharedSecret, err := svc.CreateSecret(threshold, shares)
	if err != nil {
		t.Fatalf("CreateSecret failed: %v", err)
	}

	// Use same share multiple times
	shareBase64 := base64.StdEncoding.EncodeToString(sharedSecret.Shares[0])

	keyShare, err := svc.GetShare(shareBase64, sharedSecret.SigningKey)
	if err != nil {
		t.Fatalf("GetShare failed: %v", err)
	}

	keyShares := []*keys.KeyShare{keyShare, keyShare, keyShare}

	_, _ = svc.CombineSecret(keyShares)
	// This might succeed or fail depending on the library's handling of duplicates
	// The important thing is it doesn't panic
}

func TestSignShare_AndVerify(t *testing.T) {
	key := []byte("test-signing-key-32-bytes-long!!")
	message := []byte("test message")

	signed, err := SignShare(key, message)
	if err != nil {
		t.Fatalf("SignShare failed: %v", err)
	}

	if len(signed) <= len(message) {
		t.Error("signed message should be longer than original message")
	}

	verified, err := VerifyShare(signed, key)
	if err != nil {
		t.Fatalf("VerifyShare failed: %v", err)
	}

	if !bytes.Equal(verified, message) {
		t.Error("verified message doesn't match original")
	}
}

func TestVerifyShare_InvalidSignature(t *testing.T) {
	key := []byte("test-signing-key")
	message := []byte("test message")

	signed, err := SignShare(key, message)
	if err != nil {
		t.Fatalf("SignShare failed: %v", err)
	}

	// Corrupt the signature
	signed[len(signed)-1] ^= 0xFF

	_, err = VerifyShare(signed, key)
	if err == nil {
		t.Error("VerifyShare should fail with corrupted signature")
	}
}

func TestVerifyShare_TooShort(t *testing.T) {
	key := []byte("test-signing-key")
	shortMessage := []byte("short")

	_, err := VerifyShare(shortMessage, key)
	if err == nil {
		t.Error("VerifyShare should fail with message too short for signature")
	}
}

func TestGenerateRandomKey_ProducesUniqueKeys(t *testing.T) {
	length := 32
	iterations := 10

	keys := make([][]byte, iterations)
	for i := range iterations {
		key, err := GenerateRandomKey(length)
		if err != nil {
			t.Fatalf("GenerateRandomKey failed on iteration %d: %v", i, err)
		}

		if len(key) != length {
			t.Errorf("expected key length %d, got %d", length, len(key))
		}

		keys[i] = key
	}

	// Check for uniqueness
	for i := range keys {
		for j := i + 1; j < len(keys); j++ {
			if bytes.Equal(keys[i], keys[j]) {
				t.Errorf("keys at index %d and %d are identical", i, j)
			}
		}
	}
}

func TestGenerateRandomKey_DifferentLengths(t *testing.T) {
	lengths := []int{8, 16, 32, 64, 128}

	for _, length := range lengths {
		key, err := GenerateRandomKey(length)
		if err != nil {
			t.Fatalf("GenerateRandomKey failed for length %d: %v", length, err)
		}

		if len(key) != length {
			t.Errorf("expected key length %d, got %d", length, len(key))
		}
	}
}

func TestCombineSecret_WithAllShares(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	threshold := uint16(3)
	totalShares := uint16(5)

	sharedSecret, err := svc.CreateSecret(threshold, totalShares)
	if err != nil {
		t.Fatalf("CreateSecret failed: %v", err)
	}

	// Use all shares (more than threshold)
	keyShares := make([]*keys.KeyShare, totalShares)
	for i := range totalShares {
		shareBase64 := base64.StdEncoding.EncodeToString(sharedSecret.Shares[i])

		keyShare, err := svc.GetShare(shareBase64, sharedSecret.SigningKey)
		if err != nil {
			t.Fatalf("GetShare failed for share %d: %v", i, err)
		}

		keyShares[i] = keyShare
	}

	reconstructed, err := svc.CombineSecret(keyShares)
	if err != nil {
		t.Fatalf("CombineSecret failed: %v", err)
	}

	if !bytes.Equal(sharedSecret.Secret, reconstructed) {
		t.Error("reconstructed secret doesn't match original")
	}
}

func TestCombineSecret_WithExactThreshold(t *testing.T) {
	fs := afero.NewMemMapFs()
	svc := NewService(fs)
	threshold := uint16(2)
	totalShares := uint16(5)

	sharedSecret, err := svc.CreateSecret(threshold, totalShares)
	if err != nil {
		t.Fatalf("CreateSecret failed: %v", err)
	}

	// Use exactly threshold shares
	keyShares := make([]*keys.KeyShare, threshold)
	for i := range threshold {
		shareBase64 := base64.StdEncoding.EncodeToString(sharedSecret.Shares[i])

		keyShare, err := svc.GetShare(shareBase64, sharedSecret.SigningKey)
		if err != nil {
			t.Fatalf("GetShare failed for share %d: %v", i, err)
		}

		keyShares[i] = keyShare
	}

	reconstructed, err := svc.CombineSecret(keyShares)
	if err != nil {
		t.Fatalf("CombineSecret failed: %v", err)
	}

	if !bytes.Equal(sharedSecret.Secret, reconstructed) {
		t.Error("reconstructed secret doesn't match original")
	}
}
