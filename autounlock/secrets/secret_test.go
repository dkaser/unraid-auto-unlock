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
