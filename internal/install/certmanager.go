package install

import (
	"austinhome/internal/command"
	"austinhome/internal/ui"
	"fmt"
	"time"

	"github.com/BeaverHouse/go-common/logger"
)

const (
	certManagerVersion     = "1.18.2"
	certManagerNamespace   = "cert-manager"
	certManagerMaxWaitTime = 3 * time.Minute
	route53SecretURL       = "https://raw.githubusercontent.com/BeaverHouse/cicd/refs/heads/main/charts/oss-cert-manager/resources/route53-secret.yaml"
	clusterIssuerURL       = "https://raw.githubusercontent.com/BeaverHouse/cicd/refs/heads/main/charts/oss-cert-manager/resources/cluster-issuer.yaml"
)

func InstallCertManager() error {
	ui.Log.Info("Installing Cert-Manager...")

	if err := applyCertManagerManifests(); err != nil {
		return err
	}

	if err := command.WaitForPodsReady(certManagerNamespace, "app.kubernetes.io/instance=cert-manager", certManagerMaxWaitTime); err != nil {
		return err
	}

	if err := applyRoute53Secret(); err != nil {
		return err
	}

	if err := applyClusterIssuer(); err != nil {
		return err
	}

	ui.Log.Info("Successfully installed Cert-Manager")
	return nil
}

func applyCertManagerManifests() error {
	ui.Log.Info("Applying Cert-Manager manifests...")
	manifestURL := fmt.Sprintf("https://github.com/cert-manager/cert-manager/releases/download/v%s/cert-manager.yaml", certManagerVersion)
	return command.RunCommand("kubectl", "apply", "-f", manifestURL)
}

func applyRoute53Secret() error {
	ui.Log.Info("Applying Route53 secret...")
	return command.RunCommand("kubectl", "apply", "-f", route53SecretURL)
}

func applyClusterIssuer() error {
	ui.Log.Info("Applying ClusterIssuer...")
	return command.RunCommand("kubectl", "apply", "-f", clusterIssuerURL)
}

func verifyCertManagerInstallation() error {
	ui.Log.Info("Verifying Cert-Manager installation...")

	ui.Log.Info("Cert-Manager pods status:")
	if err := command.RunCommand("kubectl", "get", "pods", "-n", certManagerNamespace); err != nil {
		return err
	}

	ui.Log.Info("ClusterIssuer status:")
	if err := command.RunCommand("kubectl", "get", "clusterissuer"); err != nil {
		ui.Log.Warn("Failed to get ClusterIssuer", logger.F("error", err))
	}

	ui.Log.Info("Route53 secret status:")
	if err := command.RunCommand("kubectl", "get", "secret", "-n", certManagerNamespace); err != nil {
		ui.Log.Warn("Failed to get secrets", logger.F("error", err))
	}

	return nil
}
