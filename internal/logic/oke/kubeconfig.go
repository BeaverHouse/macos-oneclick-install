package oke

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"austinhome/internal/logic/common"
)

const (
	macMiniLANIP   = "192.168.0.34"
	exportDirName  = "Downloads"
	exportFileName = "kubeconfig"
)

// ExportKubeconfig exports the merged kubeconfig with K3s server address
// replaced by the Mac Mini LAN IP, so it can be used from the MacBook.
func ExportKubeconfig() error {
	fmt.Println("\n📦 Step: Export kubeconfig for MacBook")

	// 1. Get flattened kubeconfig
	output, err := common.RunCommandOutput("kubectl", "config", "view", "--flatten")
	if err != nil {
		return fmt.Errorf("failed to flatten kubeconfig: %v", err)
	}

	// 2. Replace 127.0.0.1 with Mac Mini LAN IP for K3s access
	replaced := strings.ReplaceAll(output, "https://127.0.0.1:", "https://"+macMiniLANIP+":")

	// 3. Write to ~/shared/kubeconfig
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	exportDir := filepath.Join(home, exportDirName)
	if err := os.MkdirAll(exportDir, 0755); err != nil {
		return fmt.Errorf("failed to create export directory: %v", err)
	}

	exportPath := filepath.Join(exportDir, exportFileName)
	if err := os.WriteFile(exportPath, []byte(replaced), 0600); err != nil {
		return fmt.Errorf("failed to write kubeconfig: %v", err)
	}

	fmt.Printf("✅ Kubeconfig exported to %s\n", exportPath)
	fmt.Println("   K3s server address replaced: 127.0.0.1 → " + macMiniLANIP)
	return nil
}
