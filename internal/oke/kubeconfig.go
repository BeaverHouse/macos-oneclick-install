package oke

import (
	"austinhome/internal/command"
	"austinhome/internal/constants"
	"austinhome/internal/ui"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/BeaverHouse/go-common/errorhandle"
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
		return constants.ErrOperationFailed("create export directory", err)
	}

	exportPath := filepath.Join(exportDirPath, exportFileName)
	kubeconfig, clusterName, port, err := buildExportKubeconfig()
	if err != nil {
		return err
	}
	if err := os.WriteFile(exportPath, []byte(kubeconfig), 0600); err != nil {
		return constants.ErrOperationFailed("write kubeconfig", err)
	}
	externalServer := "https://" + macMiniLANIP + ":" + port
	if err := command.RunCommand("kubectl", "--kubeconfig", exportPath, "config", "set-cluster", clusterName, "--server", externalServer, "--tls-server-name", "127.0.0.1"); err != nil {
		return constants.ErrOperationFailed("update exported K3s cluster entry", err)
	}

	ui.Log.Info("Kubeconfig exported", logger.Field{Key: "path", Value: exportPath})
	ui.Log.Info("K3s server address exported", logger.Field{Key: "host", Value: macMiniLANIP})
	if err := command.RunCommand("open", "-R", exportPath); err != nil {
		ui.Log.Warn("Failed to reveal kubeconfig in Finder", logger.Field{Key: "error", Value: err})
	}
	return nil
}

func buildExportKubeconfig() (kubeconfig, clusterName, port string, err error) {
	clusterName, err = kubeconfigJSONPath(fmt.Sprintf(`{.contexts[?(@.name==%q)].context.cluster}`, k3sContextName))
	if err != nil {
		return "", "", "", constants.ErrOperationFailed("read K3s cluster name", err)
	}
	if clusterName == "" {
		return "", "", "", errorhandle.ErrNotFound(fmt.Sprintf("K3s context %q in kubeconfig", k3sContextName))
	}

	localServer, err := kubeconfigJSONPath(fmt.Sprintf(`{.clusters[?(@.name==%q)].cluster.server}`, clusterName))
	if err != nil {
		return "", "", "", constants.ErrOperationFailed("read K3s server", err)
	}
	port, err = serverPort(localServer)
	if err != nil {
		return "", "", "", err
	}

	rawConfig, err := command.RunCommandOutput("kubectl", "config", "view", "--raw", "--flatten")
	if err != nil {
		return "", "", "", constants.ErrOperationFailed("flatten kubeconfig", err)
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
		return "", constants.ErrOperationFailed(fmt.Sprintf("parse kubeconfig server %q", server), err)
	}
	if parsed.Port() == "" {
		return "", errorhandle.ErrValidationFailed(fmt.Sprintf("kubeconfig server %q has no port", server))
	}
	return parsed.Port(), nil
}
