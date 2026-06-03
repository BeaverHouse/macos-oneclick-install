package install

import (
	"austinhome/internal/colima"
	"austinhome/internal/command"
	"austinhome/internal/ui"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/BeaverHouse/go-common/logger"
)

const (
	colimaCPUs   = "4"
	colimaMemory = "8"

	k3sReadyTimeout = 600 * time.Second
)

func validatePrerequisites() error {
	if !command.IsCommandAvailable("brew") {
		return fmt.Errorf("err: Homebrew is required but not installed. Visit https://brew.sh/ to install it")
	}
	return nil
}

func installColimaIfNeeded() error {
	if !command.IsCommandAvailable("colima") {
		ui.Log.Info("Installing Colima...")
		if err := command.RunCommand("brew", "install", "colima"); err != nil {
			return fmt.Errorf("failed to install Colima: %v", err)
		}
	} else {
		ui.Log.Info("Colima is already installed")
	}
	return nil
}

func stopExistingColima() error {
	ui.Log.Info("Stopping existing Colima instances if any...")

	if colima.IsRunning() {
		colima.Stop()
		colima.Delete()
	} else {
		ui.Log.Info("No existing Colima instance found")
	}

	return nil
}

func startColimaWithK3s() error {
	ui.Log.Info("Starting Colima with Kubernetes (K3s) enabled...")

	err := command.RunCommand("colima", "start", colima.InstanceName,
		"--cpu", colimaCPUs,
		"--memory", colimaMemory,
		"--runtime", "containerd",
		"--network-address",
		"--network-mode", "bridged",
		"--network-interface", "en1",
		"--kubernetes",
		"--kubernetes-disable", "servicelb",
		"--dns", "8.8.8.8",
		"--dns", "8.8.4.4")

	if err != nil {
		return fmt.Errorf("failed to start Colima with K3s: %v", err)
	}

	ui.Log.Info("Colima with K3s started successfully")
	return nil
}

func waitForK3sReady() error {
	ui.Log.Info("Waiting for K3s cluster to be ready...")

	startTime := time.Now()
	for time.Since(startTime) < k3sReadyTimeout {
		if err := command.RunCommand("kubectl", "get", "nodes"); err == nil {
			ui.Log.Info("K3s cluster is ready!")
			return nil
		}

		ui.Log.Info("Still waiting...", logger.F("elapsed", time.Since(startTime).Truncate(time.Second)))
		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("timeout: K3s cluster not ready after %v", k3sReadyTimeout)
}

func normalizeLocalKubeconfig() error {
	ui.Log.Info("Normalizing local kubeconfig server to 127.0.0.1")

	clusterName, err := command.RunCommandOutput("kubectl", "config", "view", "--minify", "-o", "jsonpath={.clusters[0].name}")
	if err != nil {
		return fmt.Errorf("failed to read current kubeconfig cluster name: %v", err)
	}
	clusterName = strings.TrimSpace(clusterName)
	if clusterName == "" {
		return fmt.Errorf("current kubeconfig cluster name is empty")
	}

	server, err := command.RunCommandOutput("kubectl", "config", "view", "--minify", "-o", "jsonpath={.clusters[0].cluster.server}")
	if err != nil {
		return fmt.Errorf("failed to read current kubeconfig server: %v", err)
	}
	server = strings.TrimSpace(server)

	parsed, err := url.Parse(server)
	if err != nil {
		return fmt.Errorf("failed to parse kubeconfig server %q: %v", server, err)
	}
	if parsed.Port() == "" {
		return fmt.Errorf("kubeconfig server %q has no port", server)
	}

	parsed.Host = "127.0.0.1:" + parsed.Port()
	normalizedServer := parsed.String()
	if normalizedServer == server {
		ui.Log.Info("Local kubeconfig server already normalized", logger.F("server", server))
		return nil
	}

	if err := command.RunCommand("kubectl", "config", "set-cluster", clusterName, "--server", normalizedServer); err != nil {
		return fmt.Errorf("failed to normalize kubeconfig server: %v", err)
	}

	ui.Log.Info("Local kubeconfig server normalized", logger.F("from", server), logger.F("to", normalizedServer))
	return nil
}

func getColimaIPAddress() (string, error) {
	ui.Log.Info("Getting Colima VM IP address...")

	output, err := command.RunCommandOutput("colima", "list", "--format", "{{.IPAddress}}")
	if err != nil {
		return "", fmt.Errorf("failed to get Colima IP: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && line != "IPAddress" {
			ui.Log.Info("Found Colima IP", logger.F("ip", line))
			return line, nil
		}
	}

	return "", fmt.Errorf("could not find Colima VM IP address")
}

func disableTraefik() error {
	ui.Log.Info("Disabling default Traefik ingress controller...")

	if err := command.RunCommand("kubectl", "delete", "namespace", "traefik-system", "--ignore-not-found"); err != nil {
		ui.Log.Info("Traefik namespace deletion", logger.F("result", err))
	}

	if err := command.RunCommand("kubectl", "delete", "ingressclass", "traefik", "--ignore-not-found"); err != nil {
		ui.Log.Info("Traefik ingress class deletion", logger.F("result", err))
	}

	ui.Log.Info("Traefik disabled successfully")
	return nil
}

func enableEssentialAddons() error {
	ui.Log.Info("Installing essential addons...")

	ui.Log.Info("Installing metrics-server...")
	if err := command.RunCommand("kubectl", "apply", "-f",
		"https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml"); err != nil {
		ui.Log.Warn("Failed to install metrics-server", logger.F("error", err))
	} else {
		ui.Log.Info("Metrics-server installed")
	}

	return nil
}

func setNodeLabel(envLabel string) error {
	ui.Log.Info("Setting node label...")

	nodeOutput, err := command.RunCommandOutput("kubectl", "get", "nodes", "-o", "jsonpath={.items[0].metadata.name}")
	if err != nil {
		return fmt.Errorf("failed to get node name: %v", err)
	}

	nodeName := strings.TrimSpace(nodeOutput)
	labelValue := fmt.Sprintf("env=%s", envLabel)

	return command.RunCommand("kubectl", "label", "node", nodeName, labelValue, "--overwrite")
}

func testKubectlAccess() error {
	ui.Log.Info("Testing kubectl access to K3s cluster...")

	if err := command.RunCommand("kubectl", "version", "--client"); err != nil {
		return fmt.Errorf("kubectl not available: %v", err)
	}

	if err := command.RunCommand("kubectl", "cluster-info"); err != nil {
		return fmt.Errorf("kubectl cannot connect to K3s cluster: %v", err)
	}

	return nil
}

func verifyInstallation() error {
	ui.Log.Info("Final verification - checking nodes, labels, and health...")

	ui.Log.Info("Node information with labels:")
	if err := command.RunCommand("kubectl", "get", "nodes", "--show-labels"); err != nil {
		return err
	}

	ui.Log.Info("K3s cluster health status:")
	if err := command.RunCommand("kubectl", "get", "pods", "--all-namespaces"); err != nil {
		ui.Log.Warn("Health check failed", logger.F("error", err))
	}

	ui.Log.Info("Testing kubectl access...")
	if err := testKubectlAccess(); err != nil {
		ui.Log.Warn("kubectl access test failed", logger.F("error", err))
		ui.Log.Info("Tip: Check if ~/.kube/config exists and contains valid K3s cluster configuration")
	} else {
		ui.Log.Info("kubectl access is working correctly!")
	}

	if ip, err := getColimaIPAddress(); err == nil {
		ui.Log.Info("Colima VM IP", logger.F("ip", ip))
		ui.Log.Info("This IP will be used for LoadBalancer services")
	}

	ui.Log.Info("Colima K3s installation and setup completed successfully!")
	ui.Log.Info("Colima instance name", logger.F("name", colima.InstanceName))
	ui.Log.Info("Access your cluster with: kubectl get nodes")

	return nil
}

func setupK3sCluster() error {
	ui.Log.Info("Setting up Colima K3s cluster...")

	if err := stopExistingColima(); err != nil {
		return fmt.Errorf("failed to stop existing Colima: %v", err)
	}

	if err := startColimaWithK3s(); err != nil {
		return fmt.Errorf("failed to start Colima with K3s: %v", err)
	}

	if err := normalizeLocalKubeconfig(); err != nil {
		return fmt.Errorf("failed to normalize local kubeconfig: %v", err)
	}

	if err := waitForK3sReady(); err != nil {
		return fmt.Errorf("K3s cluster not ready: %v", err)
	}

	return nil
}

func setupPostInstallation(envLabel string) error {
	ui.Log.Info("Setting up post-installation configuration...")

	ui.Log.Info("kubectl context automatically configured by Colima")

	if err := disableTraefik(); err != nil {
		return err
	}

	if err := setNodeLabel(envLabel); err != nil {
		return err
	}

	return nil
}
