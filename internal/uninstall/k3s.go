package uninstall

import (
	"austinhome/internal/command"
	"austinhome/internal/ui"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BeaverHouse/go-common/logger"
)

func cleanupDirectories() error {
	ui.Log.Info("Cleaning up remaining files...")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	directoriesToRemove := []string{
		filepath.Join(homeDir, ".kube"),
		filepath.Join(homeDir, ".colima"),
	}

	for _, dir := range directoriesToRemove {
		removeDirectoryIfExists(dir)
	}

	return nil
}

func removeDirectoryIfExists(dir string) {
	if _, err := os.Stat(dir); err == nil {
		ui.Log.Info("Removing directory", logger.F("dir", dir))
		if err := os.RemoveAll(dir); err != nil {
			ui.Log.Warn("Failed to remove directory", logger.F("dir", dir), logger.F("error", err))
		}
	}
}

func killRemainingProcesses() {
	ui.Log.Info("Cleaning up remaining processes...")
	ui.Log.Info("No additional processes to clean up")
}

func cleanHomebrew() {
	ui.Log.Info("Cleaning Homebrew cache...")
	command.RunCommand("brew", "cleanup")
}

func cleanupKubectlConfig() {
	ui.Log.Info("Cleaning kubectl configuration...")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		ui.Log.Warn("Failed to get home directory", logger.F("error", err))
		return
	}

	kubeDir := filepath.Join(homeDir, ".kube")
	command.RunCommand("rm", "-rf", kubeDir)
}
