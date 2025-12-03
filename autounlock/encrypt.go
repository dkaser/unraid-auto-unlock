package main

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"os"
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

func EncryptFile(inputPath string, outputPath string, key []byte, nonce []byte) error {
	fileBytes, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	key, err = trimKey(key, 32)
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

	err = os.WriteFile(outputPath, ciphertext, 0o600)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

func DecryptFile(inputPath string, outputPath string, key []byte, nonce []byte) error {
	ciphertext, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	key, err = trimKey(key, 32)
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

	err = os.WriteFile(outputPath, plaintext, 0o600)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}
