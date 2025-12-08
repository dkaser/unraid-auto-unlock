package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/spf13/afero"
)

const (
	encryptionKeyBytes = 32
	encryptionFileMode = 0o600
	minPaddingLength   = 64
	maxPaddingLength   = 1048576
)

type encryptionData struct {
	Plaintext []byte `json:"plaintext"`
	Padding   []byte `json:"padding"`
}

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

	// Create an object with two fileBytes as base64 and a random length chunk of padding
	// This will help obscure the length of the original keyfile
	paddingLength, err := rand.Int(rand.Reader, big.NewInt(maxPaddingLength-minPaddingLength))
	if err != nil {
		return fmt.Errorf("failed to generate random padding length: %w", err)
	}

	padding := make([]byte, minPaddingLength+int(paddingLength.Int64()))

	_, err = rand.Read(padding)
	if err != nil {
		return fmt.Errorf("failed to generate random padding: %w", err)
	}

	encryptionData := encryptionData{
		Plaintext: fileBytes,
		Padding:   padding,
	}

	// Serialize the object to JSON
	encryptionDataJSON, err := json.Marshal(encryptionData)
	if err != nil {
		return fmt.Errorf("failed to serialize encryption data: %w", err)
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

	ciphertext := gcm.Seal(nil, nonce, encryptionDataJSON, nil)

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

	var encryptionData encryptionData

	err = json.Unmarshal(plaintext, &encryptionData)
	if err != nil {
		return fmt.Errorf("failed to deserialize encryption data: %w", err)
	}

	plaintext = encryptionData.Plaintext

	err = afero.WriteFile(fs, outputPath, plaintext, encryptionFileMode)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}
