package install

import (
	"austinhome/internal/logic/common"
	"fmt"
	"time"
)

// MetalLB configuration
const (
	metalLBVersion         = "0.15.2"
	metalLBNamespace       = "metallb-system"
	metalLBPodReadyTimeout = 3 * time.Minute
	metalLBNamespaceURL    = "https://raw.githubusercontent.com/BeaverHouse/cicd/refs/heads/main/charts/oss-metallb/resources/namespace.yaml"
	metalLBIPConfigURL     = "https://raw.githubusercontent.com/BeaverHouse/cicd/refs/heads/main/charts/oss-metallb/resources/ipconfig.yaml"
)

func InstallMetalLB() error {
	fmt.Println("🔩 Installing MetalLB...")

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

	fmt.Println("✅ Successfully installed MetalLB")
	return nil
}

func applyNamespace() error {
	fmt.Println("📋 Applying MetalLB namespace...")
	return common.RunCommand("kubectl", "apply", "-f", metalLBNamespaceURL)
}

func applyMetalLBManifests() error {
	fmt.Println("📦 Applying MetalLB manifests...")
	manifestURL := fmt.Sprintf("https://raw.githubusercontent.com/metallb/metallb/v%s/config/manifests/metallb-native.yaml", metalLBVersion)
	return common.RunCommand("kubectl", "apply", "-f", manifestURL)
}

func waitForMetalLBPods() error {
	return common.WaitForPodsReady(metalLBNamespace, "app=metallb", metalLBPodReadyTimeout)
}

func applyIPConfig() error {
	fmt.Println("🌐 Applying MetalLB IP configuration...")
	return common.RunCommand("kubectl", "apply", "-f", metalLBIPConfigURL)
}

func verifyMetalLBInstallation() error {
	fmt.Println("🔍 Verifying MetalLB installation...")

	fmt.Println("\n📋 MetalLB pods status:")
	if err := common.RunCommand("kubectl", "get", "pods", "-n", metalLBNamespace); err != nil {
		return err
	}

	fmt.Println("\n⚙️ MetalLB configuration:")
	if err := common.RunCommand("kubectl", "get", "ipaddresspool", "-n", metalLBNamespace); err != nil {
		fmt.Printf("Warning: failed to get IP address pool: %v\n", err)
	}

	return nil
}
