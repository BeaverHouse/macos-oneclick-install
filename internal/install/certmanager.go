package install

import (
	"austinhome/internal/command"
	"austinhome/internal/ui"
	"time"

	"github.com/BeaverHouse/go-common/logger"
)

const (
	certManagerVersion     = "1.19.1"
	certManagerNamespace   = "cert-manager"
	certManagerChart       = "cert-manager"
	certManagerRepoURL     = "https://charts.jetstack.io"
	certManagerValuesURL   = "https://raw.githubusercontent.com/BeaverHouse/cicd/refs/heads/main/charts/oss-cert-manager/values.yaml"
	certManagerMaxWaitTime = 3 * time.Minute
	clusterIssuerMaxWait   = 2 * time.Minute
	gatewayCertMaxWait     = 4 * time.Minute
	route53SecretURL       = "https://raw.githubusercontent.com/BeaverHouse/cicd/refs/heads/main/charts/oss-cert-manager/resources/route53-secret.yaml"
	route53SecretName      = "route53-secret"
	route53ExternalSecret  = "route53-external-secret"
	clusterIssuerURL       = "https://raw.githubusercontent.com/BeaverHouse/cicd/refs/heads/main/charts/oss-cert-manager/resources/cluster-issuer.yaml"
	clusterIssuerName      = "letsencrypt-cluster-issuer"
	homeGatewayTLSName     = "home-gateway-tls"
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
	if err := waitForRoute53Secret(); err != nil {
		return err
	}

	if err := applyClusterIssuer(); err != nil {
		return err
	}
	if err := waitForClusterIssuerReady(); err != nil {
		return err
	}

	if err := createHomeGateway(); err != nil {
		return err
	}
	if err := waitForGatewayCertificateReady(); err != nil {
		return err
	}

	ui.Log.Info("Successfully installed Cert-Manager")
	return nil
}

func applyCertManagerManifests() error {
	ui.Log.Info("Installing Cert-Manager chart...")
	return command.RunCommand("helm", "upgrade", "--install", "cert-manager",
		certManagerChart,
		"--repo", certManagerRepoURL,
		"--namespace", certManagerNamespace,
		"--create-namespace",
		"--version", "v"+certManagerVersion,
		"--values", certManagerValuesURL)
}

func applyRoute53Secret() error {
	ui.Log.Info("Applying Route53 secret...")
	return command.RunCommand("kubectl", "apply", "-f", route53SecretURL)
}

func waitForRoute53Secret() error {
	ui.Log.Info("Waiting for Route53 secret...")
	if err := command.RunCommand("kubectl", "wait",
		"externalsecret", route53ExternalSecret,
		"-n", certManagerNamespace,
		"--for=condition=Ready",
		"--timeout="+clusterIssuerMaxWait.String()); err != nil {
		return err
	}
	return command.RunCommand("kubectl", "get", "secret", route53SecretName, "-n", certManagerNamespace)
}

func applyClusterIssuer() error {
	ui.Log.Info("Applying ClusterIssuer...")
	return command.RunCommand("kubectl", "apply", "-f", clusterIssuerURL)
}

func waitForClusterIssuerReady() error {
	ui.Log.Info("Waiting for ClusterIssuer...")
	return command.RunCommand("kubectl", "wait",
		"clusterissuer", clusterIssuerName,
		"--for=condition=Ready",
		"--timeout="+clusterIssuerMaxWait.String())
}

func waitForGatewayCertificateReady() error {
	ui.Log.Info("Waiting for Gateway certificate...")
	deadline := time.Now().Add(gatewayCertMaxWait)
	for time.Now().Before(deadline) {
		if err := command.RunCommand("kubectl", "get", "certificate", homeGatewayTLSName, "-n", nginxGatewayNamespace); err == nil {
			return command.RunCommand("kubectl", "wait",
				"certificate", homeGatewayTLSName,
				"-n", nginxGatewayNamespace,
				"--for=condition=Ready",
				"--timeout="+time.Until(deadline).Round(time.Second).String())
		}
		time.Sleep(5 * time.Second)
	}

	return command.RunCommand("kubectl", "wait",
		"certificate", homeGatewayTLSName,
		"-n", nginxGatewayNamespace,
		"--for=condition=Ready",
		"--timeout=0s")
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
