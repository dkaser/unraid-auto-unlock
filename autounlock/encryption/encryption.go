package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"

	"github.com/spf13/afero"
)

const (
	encryptionKeyBytes = 32
	encryptionFileMode = 0o600
)

func trimKey(key []byte, length int) ([]byte, error) {
	if len(key) < length {
		return nil, fmt.Errorf(
			"key too short, must be at least %d bytes, length: %d",
			length,
			len(key),
		)
	}

	return key[:length], nil
}

func EncryptFile(fs afero.Fs, inputPath string, outputPath string, key []byte, nonce []byte) error {
	fileBytes, err := afero.ReadFile(fs, inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	key, err = trimKey(key, encryptionKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to trim key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce, err = trimKey(nonce, gcm.NonceSize())
	if err != nil {
		return fmt.Errorf("failed to trim nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, fileBytes, nil)

	err = afero.WriteFile(fs, outputPath, ciphertext, encryptionFileMode)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

func DecryptFile(fs afero.Fs, inputPath string, outputPath string, key []byte, nonce []byte) error {
	ciphertext, err := afero.ReadFile(fs, inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	key, err = trimKey(key, encryptionKeyBytes)
	if err != nil {
		return fmt.Errorf("failed to trim key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce, err = trimKey(nonce, gcm.NonceSize())
	if err != nil {
		return fmt.Errorf("failed to trim nonce: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("failed to decrypt file: %w", err)
	}

	err = afero.WriteFile(fs, outputPath, plaintext, encryptionFileMode)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}
