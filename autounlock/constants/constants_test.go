package constants

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
