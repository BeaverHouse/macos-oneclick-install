package oke

import (
	"austinhome/internal/command"
	"austinhome/internal/ui"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BeaverHouse/go-common/logger"
)

const (
	macMiniLANIP   = "192.168.0.34"
	exportDirName  = "Downloads"
	exportFileName = "kubeconfig"
)

func ExportKubeconfig() error {
	ui.Log.Info("Step: Export kubeconfig for MacBook")

	output, err := command.RunCommandOutput("kubectl", "config", "view", "--flatten")
	if err != nil {
		return fmt.Errorf("failed to flatten kubeconfig: %v", err)
	}

	replaced := strings.ReplaceAll(output, "https://127.0.0.1:", "https://"+macMiniLANIP+":")

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

	ui.Log.Info("Kubeconfig exported", logger.F("path", exportPath))
	ui.Log.Info("K3s server address replaced", logger.F("from", "127.0.0.1"), logger.F("to", macMiniLANIP))
	return nil
}
