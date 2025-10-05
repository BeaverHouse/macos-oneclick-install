package uninstall

import (
	"austinhome/internal/logic/common"
	"fmt"
	"os"
	"path/filepath"
)

func UninstallHelm() error {
	fmt.Println("⛵ Uninstalling Helm...")

	removeHelmBinary()
	cleanupHelmDirectories()
	removeHelmFromPath()

	fmt.Println("✅ Helm uninstallation completed")
	return nil
}

func removeHelmBinary() {
	fmt.Println("🗑️ Removing Helm binary...")

	helmPaths := []string{
		"/usr/local/bin/helm",
		"/opt/homebrew/bin/helm",
		filepath.Join(os.Getenv("HOME"), ".local/bin/helm"),
	}

	for _, path := range helmPaths {
		if _, err := os.Stat(path); err == nil {
			if err := os.Remove(path); err != nil {
				fmt.Printf("Warning: failed to remove %s: %v\n", path, err)
			} else {
				fmt.Printf("Removed: %s\n", path)
			}
		}
	}
}

func cleanupHelmDirectories() {
	fmt.Println("🧹 Cleaning up Helm directories...")

	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Warning: failed to get home directory: %v\n", err)
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
				fmt.Printf("Warning: failed to remove %s: %v\n", dir, err)
			} else {
				fmt.Printf("Removed directory: %s\n", dir)
			}
		}
	}
}

func removeHelmFromPath() {
	fmt.Println("🔄 Checking for Helm in system...")

	if common.IsCommandAvailable("helm") {
		fmt.Println("⚠️ Helm is still available in PATH. You may need to restart your shell or manually remove it.")
	} else {
		fmt.Println("✅ Helm successfully removed from system")
	}
}