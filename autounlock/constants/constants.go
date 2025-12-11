package constants

import "time"

const (
	ArrayRetryDelay    = 15 * time.Second
	ArrayStatusTimeout = 120 * time.Second
	ArrayTimeout       = 15 * time.Minute

	EncryptionKeyBytes = 32
	EncryptionFileMode = 0o600
	MinPaddingLength   = 64
	MaxPaddingLength   = 1048576
	SignatureBytes     = 32
	NonceBytes         = 12

	StateFileMode = 0o600
	StateDirMode  = 0o700
)
