package install

import (
	"austinhome/internal/command"
	"austinhome/internal/ui"
	"fmt"
	"os"

	"github.com/BeaverHouse/go-common/logger"
)

func InstallHelm() error {
	ui.Log.Info("Installing Helm...")

	if err := downloadHelmInstaller(); err != nil {
		return err
	}

	if err := makeInstallerExecutable(); err != nil {
		return err
	}

	if err := runHelmInstaller(); err != nil {
		return err
	}

	if err := cleanupHelmInstaller(); err != nil {
		return err
	}

	return setupHelmForK3s()
}

func downloadHelmInstaller() error {
	ui.Log.Info("Downloading Helm installer...")
	return command.RunCommand("curl", "-fsSL", "-o", "get_helm.sh",
		"https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3")
}

func makeInstallerExecutable() error {
	ui.Log.Info("Making installer executable...")
	return command.RunCommand("chmod", "700", "get_helm.sh")
}

func runHelmInstaller() error {
	ui.Log.Info("Running Helm installer...")
	return command.RunCommand("./get_helm.sh")
}

func cleanupHelmInstaller() error {
	ui.Log.Info("Cleaning up installer...")
	if err := os.Remove("get_helm.sh"); err != nil {
		ui.Log.Warn("Failed to remove installer", logger.F("error", err))
	}
	return nil
}

func setupHelmForK3s() error {
	ui.Log.Info("Setting up Helm for K3s...")

	if err := command.RunCommand("helm", "list", "--all-namespaces"); err != nil {
		return fmt.Errorf("failed to connect Helm to K3s cluster: %v", err)
	}

	ui.Log.Info("Helm configured to use K3s cluster")
	return nil
}

func verifyHelmInstallation() error {
	ui.Log.Info("Verifying Helm installation...")
	return command.RunCommand("helm", "version")
}
