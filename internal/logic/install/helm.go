package install

import (
	"austinhome/internal/logic/common"
	"fmt"
	"os"
)

func InstallHelm() error {
	fmt.Println("⛵ Installing Helm...")

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
	fmt.Println("📥 Downloading Helm installer...")
	return common.RunCommand("curl", "-fsSL", "-o", "get_helm.sh",
		"https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3")
}

func makeInstallerExecutable() error {
	fmt.Println("🔧 Making installer executable...")
	return common.RunCommand("chmod", "700", "get_helm.sh")
}

func runHelmInstaller() error {
	fmt.Println("🚀 Running Helm installer...")
	return common.RunCommand("./get_helm.sh")
}

func cleanupHelmInstaller() error {
	fmt.Println("🧹 Cleaning up installer...")
	if err := os.Remove("get_helm.sh"); err != nil {
		fmt.Printf("Warning: failed to remove installer: %v\n", err)
	}
	return nil
}

func setupHelmForK3s() error {
	fmt.Println("🔧 Setting up Helm for K3s...")

	// Kubeconfig is already set up by configureKubectlAccess() in k3s.go
	// Just verify Helm can connect to the cluster
	if err := common.RunCommand("helm", "list", "--all-namespaces"); err != nil {
		return fmt.Errorf("failed to connect Helm to K3s cluster: %v", err)
	}

	fmt.Println("✅ Helm configured to use K3s cluster")
	return nil
}

func verifyHelmInstallation() error {
	fmt.Println("✅ Verifying Helm installation...")
	return common.RunCommand("helm", "version")
}