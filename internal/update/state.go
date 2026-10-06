package update

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"go.yaml.in/yaml/v3"
)

var fileName = "update-check.yaml"
var tempFileName = "update-check.yaml.tmp"

type State struct {
	LastCheckedAt       time.Time `yaml:"last_checked_at"`
	LatestVersion       string    `yaml:"latest_version"`
	UpdateCheckInterval string    `yaml:"update_check_interval"`
}

type Store struct {
	Path string
}

func DefaultPath() (string, error) {
	xdgStateHome := os.Getenv("XDG_STATE_HOME")
	if xdgStateHome == "" || !filepath.IsAbs(xdgStateHome) {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, ".local/state/forgesync", fileName), nil
	}

	return filepath.Join(xdgStateHome, "forgesync", fileName), nil
}

func (s *Store) Load() (State, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return State{}, nil
		}
		return State{}, fmt.Errorf("read state %s: %w", s.Path, err)
	}

	var state State
	if err := yaml.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("read state %s: %w", s.Path, err)
	}

	if _, err := time.ParseDuration(state.UpdateCheckInterval); err != nil {
		state.UpdateCheckInterval = "6h"
	}

	return state, nil
}

func (s *Store) Save(state State) error {
	data, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("write state %s: %w", s.Path, err)
	}

	if err := os.MkdirAll(filepath.Dir(s.Path), 0700); err != nil {
		return fmt.Errorf("write state %s: %w", s.Path, err)
	}

	tempFile, err := os.CreateTemp(filepath.Dir(s.Path), tempFileName)
	if err != nil {
		return fmt.Errorf("write state %s: %w", s.Path, err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write(data); err != nil {
		tempFile.Close()
		return fmt.Errorf("write state %s: %w", s.Path, err)
	}
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("write state %s: %w", s.Path, err)
	}

	if err := os.Rename(tempFile.Name(), s.Path); err != nil {
		return fmt.Errorf("rename state %s: %w", s.Path, err)
	}

	return nil
}
