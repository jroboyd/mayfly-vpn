package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type State struct {
	Region          string `json:"region"`
	InstanceID      string `json:"instance_id,omitempty"`
	SecurityGroupID string `json:"security_group_id,omitempty"`
}

func path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".mayfly", "state.json"), nil
}

// Save writes the state to disk.
func Save(s *State) error {
	p, err := path()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(p), 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(p, data, 0600)
}

// Load reads the state from disk. Returns nil if no state file exists.
func Load() (*State, error) {
	p, err := path()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}

	return &s, nil
}

// Clear removes the state file.
func Clear() error {
	p, err := path()
	if err != nil {
		return err
	}

	err = os.Remove(p)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
