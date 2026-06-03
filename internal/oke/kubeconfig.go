package oke

import (
	"austinhome/internal/command"
	"austinhome/internal/ui"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/BeaverHouse/go-common/logger"
)

const (
	macMiniLANIP   = "192.168.0.34"
	k3sContextName = "colima-k3s-homeserver"
	exportDirPath  = "/Users/Shared/austinhome"
	exportFileName = "config"
)

func ExportKubeconfig() error {
	ui.Log.Info("Step: Export kubeconfig for MacBook")

	if err := os.MkdirAll(exportDirPath, 0755); err != nil {
		return fmt.Errorf("failed to create export directory: %v", err)
	}

	exportPath := filepath.Join(exportDirPath, exportFileName)
	kubeconfig, clusterName, port, err := buildExportKubeconfig()
	if err != nil {
		return err
	}
	if err := os.WriteFile(exportPath, []byte(kubeconfig), 0600); err != nil {
		return fmt.Errorf("failed to write kubeconfig: %v", err)
	}
	externalServer := "https://" + macMiniLANIP + ":" + port
	if err := command.RunCommand("kubectl", "--kubeconfig", exportPath, "config", "set-cluster", clusterName, "--server", externalServer, "--tls-server-name", "127.0.0.1"); err != nil {
		return fmt.Errorf("failed to update exported K3s cluster entry: %v", err)
	}

	ui.Log.Info("Kubeconfig exported", logger.F("path", exportPath))
	ui.Log.Info("K3s server address exported", logger.F("host", macMiniLANIP))
	if err := command.RunCommand("open", "-R", exportPath); err != nil {
		ui.Log.Warn("Failed to reveal kubeconfig in Finder", logger.F("error", err))
	}
	return nil
}

func buildExportKubeconfig() (kubeconfig, clusterName, port string, err error) {
	clusterName, err = kubeconfigJSONPath(fmt.Sprintf(`{.contexts[?(@.name==%q)].context.cluster}`, k3sContextName))
	if err != nil {
		return "", "", "", fmt.Errorf("failed to read K3s cluster name: %v", err)
	}
	if clusterName == "" {
		return "", "", "", fmt.Errorf("K3s context %q not found in kubeconfig", k3sContextName)
	}

	localServer, err := kubeconfigJSONPath(fmt.Sprintf(`{.clusters[?(@.name==%q)].cluster.server}`, clusterName))
	if err != nil {
		return "", "", "", fmt.Errorf("failed to read K3s server: %v", err)
	}
	port, err = serverPort(localServer)
	if err != nil {
		return "", "", "", err
	}

	rawConfig, err := command.RunCommandOutput("kubectl", "config", "view", "--raw", "--flatten")
	if err != nil {
		return "", "", "", fmt.Errorf("failed to flatten kubeconfig: %v", err)
	}
	return rawConfig, clusterName, port, nil
}

func kubeconfigJSONPath(path string) (string, error) {
	output, err := command.RunCommandOutput("kubectl", "config", "view", "--raw", "-o", "jsonpath="+path)
	return strings.TrimSpace(output), err
}

func serverPort(server string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(server))
	if err != nil {
		return "", fmt.Errorf("failed to parse kubeconfig server %q: %v", server, err)
	}
	if parsed.Port() == "" {
		return "", fmt.Errorf("kubeconfig server %q has no port", server)
	}
	return parsed.Port(), nil
}
