package common

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const configDirName = ".austinhome"

// configDir returns the absolute path to ~/.austinhome/
func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %v", err)
	}
	return filepath.Join(home, configDirName), nil
}

// ConfigSave writes a value to ~/.austinhome/<key>
func ConfigSave(key, value string) error {
	dir, err := configDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory %s: %v", dir, err)
	}

	path := filepath.Join(dir, key)
	if err := os.WriteFile(path, []byte(value), 0600); err != nil {
		return fmt.Errorf("failed to write config %s: %v", key, err)
	}

	fmt.Printf("✅ Config saved: %s\n", key)
	return nil
}

// ConfigLoad reads a value from ~/.austinhome/<key>
func ConfigLoad(key string) (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}

	path := filepath.Join(dir, key)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read config %s: %v", key, err)
	}

	return strings.TrimSpace(string(data)), nil
}
