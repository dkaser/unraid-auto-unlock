package secrets

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
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"

	"github.com/dkaser/unraid-auto-unlock/autounlock/constants"
)

func SignShare(key []byte, message []byte) ([]byte, error) {
	mac, err := calculateHMAC(key, message)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate HMAC: %w", err)
	}

	return append(message, mac...), nil
}

func calculateHMAC(key []byte, message []byte) ([]byte, error) {
	mac := hmac.New(sha256.New, key)

	_, err := mac.Write(message)
	if err != nil {
		return nil, fmt.Errorf("failed to write message to HMAC: %w", err)
	}

	return mac.Sum(nil), nil
}

func VerifyShare(signedMessage []byte, key []byte) ([]byte, error) {
	if len(signedMessage) < constants.SignatureBytes {
		return nil, errors.New("signed message too short")
	}

	// Split the signed message into the original message and the signature (last 32 bytes).
	message := signedMessage[:len(signedMessage)-constants.SignatureBytes]
	signature := signedMessage[len(signedMessage)-constants.SignatureBytes:]

	expectedMAC, err := calculateHMAC(key, message)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate HMAC: %w", err)
	}

	if !hmac.Equal(signature, expectedMAC) {
		return nil, errors.New("invalid signature")
	}

	return message, nil
}

func GenerateRandomKey(length int) ([]byte, error) {
	key := make([]byte, length)

	_, err := rand.Read(key)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random key: %w", err)
	}

	return key, nil
}
