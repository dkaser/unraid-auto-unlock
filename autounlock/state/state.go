package state

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	stateDirMode  = 0o700
	stateFileMode = 0o600
)

type State struct {
	VerificationKey []byte `json:"verificationKey"`
	SigningKey      []byte `json:"signingKey"`
	Threshold       uint16 `json:"threshold"`
}

func WriteStateToFile(
	verificationKey []byte,
	signingKey []byte,
	stateFile string,
	threshold uint16,
) error {
	// Write the verification key, signing key, and threshold to the state file as JSON
	state := State{
		VerificationKey: verificationKey,
		SigningKey:      signingKey,
		Threshold:       threshold,
	}

	// Marshal the state to JSON
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state to JSON: %w", err)
	}

	// Ensure the directory for the state file exists
	err = os.MkdirAll(filepath.Dir(stateFile), stateDirMode)
	if err != nil {
		return fmt.Errorf("failed to create directory for state file: %w", err)
	}

	// Write the JSON data to the state file
	err = os.WriteFile(stateFile, data, stateFileMode)
	if err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

func ReadStateFromFile(stateFile string) (State, error) {
	var state State

	// Read the JSON data from the state file
	data, err := os.ReadFile(stateFile)
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
