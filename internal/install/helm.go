package install

import (
	"austinhome/internal/command"
	"austinhome/internal/ui"
	"fmt"
)

func InstallHelm() error {
	ui.Log.Info("Installing Helm...")

	if !command.IsCommandAvailable("helm") {
		ui.Log.Info("Installing Helm via Homebrew...")
		if err := command.RunCommand("brew", "install", "helm"); err != nil {
			return fmt.Errorf("failed to install Helm: %v", err)
		}
	} else {
		ui.Log.Info("Helm is already installed")
	}

	return setupHelmForK3s()
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
