package install

import (
	"austinhome/internal/logic/common"
	"fmt"
	"time"
)

const (
	certManagerVersion     = "1.18.2"
	certManagerNamespace   = "cert-manager"
	certManagerMaxWaitTime = 3 * time.Minute
	route53SecretURL       = "https://raw.githubusercontent.com/BeaverHouse/hybrid-cicd/refs/heads/main/charts/oss-cert-manager/resources/route53-secret.yaml"
	clusterIssuerURL       = "https://raw.githubusercontent.com/BeaverHouse/hybrid-cicd/refs/heads/main/charts/oss-cert-manager/resources/cluster-issuer.yaml"
)

func InstallCertManager() error {
	fmt.Println("🔒 Installing Cert-Manager...")

	if err := applyCertManagerManifests(); err != nil {
		return err
	}

	if err := common.WaitForPodsReady(certManagerNamespace, "app.kubernetes.io/instance=cert-manager", certManagerMaxWaitTime); err != nil {
		return err
	}

	if err := applyRoute53Secret(); err != nil {
		return err
	}

	if err := applyClusterIssuer(); err != nil {
		return err
	}

	fmt.Println("✅ Successfully installed Cert-Manager")
	return nil
}

func applyCertManagerManifests() error {
	fmt.Println("📦 Applying Cert-Manager manifests...")
	manifestURL := fmt.Sprintf("https://github.com/cert-manager/cert-manager/releases/download/v%s/cert-manager.yaml", certManagerVersion)
	return common.RunCommand("kubectl", "apply", "-f", manifestURL)
}

func applyRoute53Secret() error {
	fmt.Println("🔑 Applying Route53 secret...")
	return common.RunCommand("kubectl", "apply", "-f", route53SecretURL)
}

func applyClusterIssuer() error {
	fmt.Println("📋 Applying ClusterIssuer...")
	return common.RunCommand("kubectl", "apply", "-f", clusterIssuerURL)
}

func verifyCertManagerInstallation() error {
	fmt.Println("🔍 Verifying Cert-Manager installation...")

	fmt.Println("\n📋 Cert-Manager pods status:")
	if err := common.RunCommand("kubectl", "get", "pods", "-n", certManagerNamespace); err != nil {
		return err
	}

	fmt.Println("\n🔒 ClusterIssuer status:")
	if err := common.RunCommand("kubectl", "get", "clusterissuer"); err != nil {
		fmt.Printf("Warning: failed to get ClusterIssuer: %v\n", err)
	}

	fmt.Println("\n🔑 Route53 secret status:")
	if err := common.RunCommand("kubectl", "get", "secret", "-n", certManagerNamespace); err != nil {
		fmt.Printf("Warning: failed to get secrets: %v\n", err)
	}

	return nil
}
