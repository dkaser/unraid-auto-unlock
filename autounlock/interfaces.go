package main

import (
	"time"

	"github.com/bytemare/secret-sharing/keys"
	"github.com/dkaser/unraid-auto-unlock/autounlock/secrets"
	"github.com/dkaser/unraid-auto-unlock/autounlock/state"
	"github.com/dkaser/unraid-auto-unlock/autounlock/unraid"
)

// UnraidOperations defines operations for interacting with Unraid system.
// Implemented by *unraid.Service.
type UnraidOperations interface {
	IsUnraid() bool
	TestKeyfile(keyfile string) error
	WaitForVarIni() error
	GetFsState() (string, error)
	VerifyArrayStatus(status string) bool
	StartArray() error
	WaitForArrayStatus(status string, timeout time.Duration) error
}

// EncryptionOperations defines operations for encryption/decryption.
// Implemented by *encryption.Service.
type EncryptionOperations interface {
	EncryptFile(inputPath string, outputPath string, key []byte, nonce []byte) error
	DecryptFile(inputPath string, outputPath string, key []byte, nonce []byte) error
}

// StateOperations defines operations for state management.
// Implemented by *state.Service.
type StateOperations interface {
	WriteStateToFile(
		verificationKey []byte,
		signingKey []byte,
		nonce []byte,
		stateFile string,
		threshold uint16,
	) error
	ReadStateFromFile(stateFile string) (state.State, error)
}

// SecretsOperations defines operations for secret sharing.
// Implemented by *secrets.Service.
type SecretsOperations interface {
	CreateSecret(threshold uint16, shares uint16) (secrets.SharedSecret, error)
	CombineSecret(shares []*keys.KeyShare) ([]byte, error)
	GetShare(shareStr string, signingKey []byte) (*keys.KeyShare, error)
	ReadPathsFromFile(filename string) ([]string, error)
	GetShares(
		paths []string,
		appState state.State,
		retryInterval uint16,
		serverTimeout uint16,
		test bool,
		unraidSvc *unraid.Service,
	) ([]*keys.KeyShare, error)
}
