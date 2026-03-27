package config

import (
	"austinhome/internal/ui"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BeaverHouse/go-common/logger"
)

const configDirName = ".austinhome"

func configDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %v", err)
	}
	return filepath.Join(home, configDirName), nil
}

func Save(key, value string) error {
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

	ui.Log.Info("Config saved", logger.F("key", key))
	return nil
}

func Load(key string) (string, error) {
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
