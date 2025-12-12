package secrets

import (
	"encoding/base64"
	"fmt"

	"github.com/bytemare/ecc"
	secretsharing "github.com/bytemare/secret-sharing"
	"github.com/bytemare/secret-sharing/keys"
	"github.com/dkaser/unraid-auto-unlock/autounlock/constants"
	"github.com/spf13/afero"
)

// Service provides secret sharing operations.
type Service struct {
	fs afero.Fs
}

// NewService creates a new secrets service.
func NewService(fs afero.Fs) *Service {
	return &Service{fs: fs}
}

// SharedSecret represents a shared secret with all its components.
type SharedSecret struct {
	VerificationKey []byte
	SigningKey      []byte
	Shares          [][]byte
	Secret          []byte
	Nonce           []byte
}

// CreateSecret creates a new shared secret.
func (s *Service) CreateSecret(threshold uint16, shares uint16) (SharedSecret, error) {
	secret := SharedSecret{}

	// Then, split the secret into shares using the specified threshold and number of shares.
	curve := ecc.Ristretto255Sha512
	secretKey := curve.NewScalar().Random()

	shareVals, err := secretsharing.Shard(curve, secretKey, threshold, shares)
	if err != nil {
		return SharedSecret{}, fmt.Errorf("failed to split secret: %w", err)
	}

	secret.Secret = secretKey.Encode()

	// Save the verification key from the first share (they all have the same verification key).
	secret.VerificationKey = shareVals[0].VerificationKey.Encode()

	secret.SigningKey, err = GenerateRandomKey(constants.SignatureBytes)
	if err != nil {
		return SharedSecret{}, fmt.Errorf("failed to generate signing key: %w", err)
	}

	secret.Nonce, err = GenerateRandomKey(constants.NonceBytes)
	if err != nil {
		return SharedSecret{}, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Finally, output the shares.
	for _, share := range shareVals {
		bytes := share.Encode()

		signedShare, err := SignShare(secret.SigningKey, bytes)
		if err != nil {
			return SharedSecret{}, fmt.Errorf("failed to sign share: %w", err)
		}

		secret.Shares = append(secret.Shares, signedShare)
	}

	return secret, nil
}

// CombineSecret combines shares to reconstruct the secret.
func (s *Service) CombineSecret(shares []*keys.KeyShare) ([]byte, error) {
	recovered, err := secretsharing.CombineShares(shares)
	if err != nil {
		return nil, fmt.Errorf("failed to combine shares: %w", err)
	}

	return recovered.Encode(), nil
}

// GetShare retrieves and verifies a share.
func (s *Service) GetShare(shareStr string, signingKey []byte) (*keys.KeyShare, error) {
	decodedShareBytes, err := base64.StdEncoding.DecodeString(shareStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 share: %w", err)
	}

	decodedShare, err := VerifyShare(decodedShareBytes, signingKey)
	if err != nil {
		return nil, fmt.Errorf("failed to verify share: %w", err)
	}

	keyShare := &keys.KeyShare{}

	err = keyShare.Decode(decodedShare)
	if err != nil {
		return nil, fmt.Errorf("failed to decode share: %w", err)
	}

	return keyShare, nil
}
