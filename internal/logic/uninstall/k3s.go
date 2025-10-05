package uninstall

import (
	"austinhome/internal/logic/common"
	"fmt"
	"os"
	"path/filepath"
)

const colimaInstanceName = "k3s-homeserver"

func stopColima() {
	fmt.Println("⏹️ Stopping Colima instance...")
	if err := common.RunCommand("colima", "stop", colimaInstanceName); err != nil {
		fmt.Printf("Warning: failed to stop Colima: %v\n", err)
	}
}

func deleteColima() {
	fmt.Println("💥 Deleting Colima instance...")
	if err := common.RunCommand("colima", "delete", colimaInstanceName, "--force"); err != nil {
		fmt.Printf("Warning: failed to delete Colima: %v\n", err)
	}
}

func cleanupDirectories() error {
	fmt.Println("🧽 Cleaning up remaining files...")

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
		fmt.Printf("Removing directory: %s\n", dir)
		if err := os.RemoveAll(dir); err != nil {
			fmt.Printf("Warning: failed to remove %s: %v\n", dir, err)
		}
	}
}

func killRemainingProcesses() {
	fmt.Println("🔄 Cleaning up remaining processes...")
	// Colima manages its own processes, so no manual cleanup needed
	fmt.Println("✅ No additional processes to clean up")
}

func cleanHomebrew() {
	fmt.Println("🧹 Cleaning Homebrew cache...")
	common.RunCommand("brew", "cleanup")
}

func cleanupKubectlConfig() {
	fmt.Println("🔧 Cleaning kubectl configuration...")
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("Warning: failed to get home directory: %v\n", err)
		return
	}

	kubeDir := filepath.Join(homeDir, ".kube")
	common.RunCommand("rm", "-rf", kubeDir)
}