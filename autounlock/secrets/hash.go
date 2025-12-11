package secrets

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
