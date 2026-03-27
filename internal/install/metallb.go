package install

import (
	"austinhome/internal/command"
	"austinhome/internal/ui"
	"fmt"
	"time"

	"github.com/BeaverHouse/go-common/logger"
)

const (
	metalLBVersion         = "0.15.2"
	metalLBNamespace       = "metallb-system"
	metalLBPodReadyTimeout = 3 * time.Minute
	metalLBNamespaceURL    = "https://raw.githubusercontent.com/BeaverHouse/cicd/refs/heads/main/charts/oss-metallb/resources/namespace.yaml"
	metalLBIPConfigURL     = "https://raw.githubusercontent.com/BeaverHouse/cicd/refs/heads/main/charts/oss-metallb/resources/ipconfig.yaml"
)

func InstallMetalLB() error {
	ui.Log.Info("Installing MetalLB...")

	if err := applyNamespace(); err != nil {
		return err
	}

	if err := applyMetalLBManifests(); err != nil {
		return err
	}

	if err := waitForMetalLBPods(); err != nil {
		return err
	}

	if err := applyIPConfig(); err != nil {
		return err
	}

	ui.Log.Info("Successfully installed MetalLB")
	return nil
}

func applyNamespace() error {
	ui.Log.Info("Applying MetalLB namespace...")
	return command.RunCommand("kubectl", "apply", "-f", metalLBNamespaceURL)
}

func applyMetalLBManifests() error {
	ui.Log.Info("Applying MetalLB manifests...")
	manifestURL := fmt.Sprintf("https://raw.githubusercontent.com/metallb/metallb/v%s/config/manifests/metallb-native.yaml", metalLBVersion)
	return command.RunCommand("kubectl", "apply", "-f", manifestURL)
}

func waitForMetalLBPods() error {
	return command.WaitForPodsReady(metalLBNamespace, "app=metallb", metalLBPodReadyTimeout)
}

func applyIPConfig() error {
	ui.Log.Info("Applying MetalLB IP configuration...")
	return command.RunCommand("kubectl", "apply", "-f", metalLBIPConfigURL)
}

func verifyMetalLBInstallation() error {
	ui.Log.Info("Verifying MetalLB installation...")

	ui.Log.Info("MetalLB pods status:")
	if err := command.RunCommand("kubectl", "get", "pods", "-n", metalLBNamespace); err != nil {
		return err
	}

	ui.Log.Info("MetalLB configuration:")
	if err := command.RunCommand("kubectl", "get", "ipaddresspool", "-n", metalLBNamespace); err != nil {
		ui.Log.Warn("Failed to get IP address pool", logger.F("error", err))
	}

	return nil
}
