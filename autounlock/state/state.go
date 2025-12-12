package state

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/dkaser/unraid-auto-unlock/autounlock/constants"
	"github.com/spf13/afero"
)

// Service provides state management operations.
type Service struct {
	fs afero.Fs
}

// NewService creates a new state service.
func NewService(fs afero.Fs) *Service {
	return &Service{fs: fs}
}

// State represents the application state.
type State struct {
	VerificationKey []byte `json:"verificationKey"`
	SigningKey      []byte `json:"signingKey"`
	Nonce           []byte `json:"nonce"`
	Threshold       uint16 `json:"threshold"`
}

// WriteStateToFile writes the state to a file.
func (s *Service) WriteStateToFile(
	verificationKey []byte,
	signingKey []byte,
	nonce []byte,
	stateFile string,
	threshold uint16,
) error {
	state := State{
		VerificationKey: verificationKey,
		SigningKey:      signingKey,
		Nonce:           nonce,
		Threshold:       threshold,
	}

	// Marshal the state to JSON
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state to JSON: %w", err)
	}

	// Ensure the directory for the state file exists
	err = s.fs.MkdirAll(filepath.Dir(stateFile), constants.StateDirMode)
	if err != nil {
		return fmt.Errorf("failed to create directory for state file: %w", err)
	}

	// Write the JSON data to the state file
	err = afero.WriteFile(s.fs, stateFile, data, constants.StateFileMode)
	if err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// ReadStateFromFile reads the state from a file.
func (s *Service) ReadStateFromFile(stateFile string) (State, error) {
	var state State

	// Read the JSON data from the state file
	data, err := afero.ReadFile(s.fs, stateFile)
	if err != nil {
		return state, fmt.Errorf("failed to read state file: %w", err)
	}

	// Unmarshal the JSON data into the State struct
	err = json.Unmarshal(data, &state)
	if err != nil {
		return state, fmt.Errorf("failed to unmarshal state JSON: %w", err)
	}

	return state, nil
}
