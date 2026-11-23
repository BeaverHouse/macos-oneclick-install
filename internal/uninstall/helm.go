package uninstall

import (
	"austinhome/internal/command"
	"austinhome/internal/ui"
	"os"
	"path/filepath"

	"github.com/BeaverHouse/go-common/logger"
)

func UninstallHelm() error {
	ui.Log.Info("Uninstalling Helm...")

	removeHelmBinary()
	cleanupHelmDirectories()
	removeHelmFromPath()

	ui.Log.Info("Helm uninstallation completed")
	return nil
}

func removeHelmBinary() {
	ui.Log.Info("Removing Helm binary...")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		ui.Log.Warn("Failed to get home directory", logger.Field{Key: "error", Value: err})
		return
	}

	helmPaths := []string{
		"/usr/local/bin/helm",
		"/opt/homebrew/bin/helm",
		filepath.Join(homeDir, ".local/bin/helm"),
	}

	for _, path := range helmPaths {
		if _, err := os.Stat(path); err == nil {
			if err := os.Remove(path); err != nil {
				ui.Log.Warn("Failed to remove helm binary", logger.Field{Key: "path", Value: path}, logger.Field{Key: "error", Value: err})
			} else {
				ui.Log.Info("Removed", logger.Field{Key: "path", Value: path})
			}
		}
	}
}

func cleanupHelmDirectories() {
	ui.Log.Info("Cleaning up Helm directories...")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		ui.Log.Warn("Failed to get home directory", logger.Field{Key: "error", Value: err})
		return
	}

	helmDirs := []string{
		filepath.Join(homeDir, ".helm"),
		filepath.Join(homeDir, ".config", "helm"),
		filepath.Join(homeDir, ".cache", "helm"),
		filepath.Join(homeDir, "Library", "Caches", "helm"),
	}

	for _, dir := range helmDirs {
		if _, err := os.Stat(dir); err == nil {
			if err := os.RemoveAll(dir); err != nil {
				ui.Log.Warn("Failed to remove directory", logger.Field{Key: "dir", Value: dir}, logger.Field{Key: "error", Value: err})
			} else {
				ui.Log.Info("Removed directory", logger.Field{Key: "dir", Value: dir})
			}
		}
	}
}

func removeHelmFromPath() {
	ui.Log.Info("Checking for Helm in system...")

	if command.IsCommandAvailable("helm") {
		ui.Log.Warn("Helm is still available in PATH. You may need to restart your shell or manually remove it.")
	} else {
		ui.Log.Info("Helm successfully removed from system")
	}
}
