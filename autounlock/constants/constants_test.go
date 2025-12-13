package constants

import (
	"testing"
)

func TestEncryptionConstants(t *testing.T) {
	// EncryptionKeyBytes should match AES-256 requirement (32 bytes)
	if EncryptionKeyBytes != 32 {
		t.Error("EncryptionKeyBytes should be 32 for AES-256")
	}

	// NonceBytes should match GCM standard nonce size (12 bytes)
	if NonceBytes != 12 {
		t.Error("NonceBytes should be 12 for GCM")
	}
}

func TestPaddingConstants(t *testing.T) {
	// Sanity check: MaxPaddingLength should be greater than MinPaddingLength
	if MaxPaddingLength <= MinPaddingLength {
		t.Error("MaxPaddingLength should be greater than MinPaddingLength")
	}

	// MinPaddingLength should be positive
	if MinPaddingLength <= 0 {
		t.Error("MinPaddingLength should be positive")
	}
}
