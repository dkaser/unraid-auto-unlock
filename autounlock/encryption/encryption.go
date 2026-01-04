package encryption

/*
	autounlock - Unraid Auto Unlock
	Copyright (C) 2025-2026 Derek Kaser

	This program is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	This program is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/dkaser/unraid-auto-unlock/autounlock/constants"
	"github.com/spf13/afero"
)

// Service provides encryption operations.
type Service struct {
	fs afero.Fs
}

// NewService creates a new encryption service.
func NewService(fs afero.Fs) *Service {
	return &Service{fs: fs}
}

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

func generatePadding() ([]byte, error) {
	paddingLength, err := rand.Int(
		rand.Reader,
		big.NewInt(constants.MaxPaddingLength-constants.MinPaddingLength),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random padding length: %w", err)
	}

	padding := make([]byte, constants.MinPaddingLength+int(paddingLength.Int64()))

	_, err = rand.Read(padding)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random padding: %w", err)
	}

	return padding, nil
}

func encryptData(data []byte, key []byte, nonce []byte) ([]byte, error) {
	key, err := trimKey(key, constants.EncryptionKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to trim key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce, err = trimKey(nonce, gcm.NonceSize())
	if err != nil {
		return nil, fmt.Errorf("failed to trim nonce: %w", err)
	}

	return gcm.Seal(nil, nonce, data, nil), nil
}

// EncryptFile encrypts a file using AES-GCM.
func (s *Service) EncryptFile(
	inputPath string,
	outputPath string,
	key []byte,
	nonce []byte,
) error {
	fileBytes, err := afero.ReadFile(s.fs, inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	// Create an object with the plaintext and a random length chunk of padding
	// This will help obscure the length of the original keyfile
	padding, err := generatePadding()
	if err != nil {
		return err
	}

	envelope := encryptionData{
		Plaintext: fileBytes,
		Padding:   padding,
	}

	// Serialize the object to JSON
	envelopeJSON, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("failed to serialize encryption data: %w", err)
	}

	ciphertext, err := encryptData(envelopeJSON, key, nonce)
	if err != nil {
		return err
	}

	err = afero.WriteFile(s.fs, outputPath, ciphertext, constants.EncryptionFileMode)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}

// DecryptFile decrypts a file using AES-GCM.
func (s *Service) DecryptFile(
	inputPath string,
	outputPath string,
	key []byte,
	nonce []byte,
) error {
	ciphertext, err := afero.ReadFile(s.fs, inputPath)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}

	key, err = trimKey(key, constants.EncryptionKeyBytes)
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

	var envelope encryptionData

	err = json.Unmarshal(plaintext, &envelope)
	if err != nil {
		return fmt.Errorf(
			"failed to deserialize encryption data (file may be in old format): %w",
			err,
		)
	}

	plaintext = envelope.Plaintext

	err = afero.WriteFile(s.fs, outputPath, plaintext, constants.EncryptionFileMode)
	if err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}
