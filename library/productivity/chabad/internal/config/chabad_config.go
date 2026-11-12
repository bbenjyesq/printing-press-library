// Copyright 2026 Bob Benjy and contributors. Licensed under Apache-2.0. See LICENSE.

package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// SavedLocation is the default Chabad.org location remembered by
// `chabad-pp-cli locations use`.
type SavedLocation struct {
	ID   string `json:"id"`
	Type int    `json:"type"` // 1 = Chabad.org city id, 2 = US ZIP
	Name string `json:"name"`
}

// DefaultLocationPath returns the file that stores the default location, next
// to the CLI's config file.
func DefaultLocationPath(configPath string) (string, error) {
	path, _, err := resolveConfigPath(configPath)
	if err != nil {
		return "", err
	}
	return filepath.Join(filepath.Dir(path), "default-location.json"), nil
}

// LoadDefaultLocation returns the saved default location, or ok=false when
// none has been saved.
func LoadDefaultLocation(configPath string) (SavedLocation, bool, error) {
	p, err := DefaultLocationPath(configPath)
	if err != nil {
		return SavedLocation{}, false, err
	}
	data, err := os.ReadFile(p) // #nosec G304 -- fixed filename inside the CLI's own config directory
	if errors.Is(err, os.ErrNotExist) {
		return SavedLocation{}, false, nil
	}
	if err != nil {
		return SavedLocation{}, false, err
	}
	var loc SavedLocation
	if err := json.Unmarshal(data, &loc); err != nil {
		return SavedLocation{}, false, err
	}
	if loc.ID == "" || (loc.Type != 1 && loc.Type != 2) {
		return SavedLocation{}, false, nil
	}
	return loc, true, nil
}

// SaveDefaultLocation writes the default location with owner-only permissions.
func SaveDefaultLocation(configPath string, loc SavedLocation) (string, error) {
	p, err := DefaultLocationPath(configPath)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o700); err != nil {
		return "", err
	}
	data, err := json.MarshalIndent(loc, "", "  ")
	if err != nil {
		return "", err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, append(data, '\n'), 0o600); err != nil {
		return "", err
	}
	return p, os.Rename(tmp, p)
}
