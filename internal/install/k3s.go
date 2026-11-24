package install

import (
	"austinhome/internal/colima"
	"austinhome/internal/command"
	"austinhome/internal/constants"
	"austinhome/internal/ui"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/BeaverHouse/go-common/errorhandle"
	"github.com/BeaverHouse/go-common/logger"
)

const (
	colimaCPUs   = "4"
	colimaMemory = "8"

	// Colima's default k3s is fixed per Colima release, not the newest k3s.
	k3sVersion = "v1.36.6+k3s1"

	k3sReadyTimeout = 600 * time.Second
)

func validatePrerequisites() error {
	if !command.IsCommandAvailable("brew") {
		return errorhandle.ErrInternalMsg("Homebrew is required but not installed. Visit https://brew.sh/ to install it")
	}
	return nil
}

// installColima also upgrades an outdated Colima, because brew install does.
func installColima() error {
	ui.Log.Info("Installing or upgrading Colima...")
	if err := command.RunCommand("brew", "install", "colima"); err != nil {
		return constants.ErrOperationFailed("install Colima", err)
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
		"--kubernetes-version", k3sVersion,
		"--kubernetes-disable", "servicelb",
		"--dns", "8.8.8.8",
		"--dns", "8.8.4.4")

	if err != nil {
		return constants.ErrOperationFailed("start Colima with K3s", err)
	}

	ui.Log.Info("Colima with K3s started successfully")
	return nil
}

// upgradeVMKernel boots the VM on Ubuntu's current kernel.
// The Colima disk image predates fixes such as CVE-2026-98432, and the VM is
// recreated on every reinstall, so the upgrade has to run each time.
func upgradeVMKernel() error {
	ui.Log.Info("Upgrading Colima VM kernel...")

	aptGet := []string{"ssh", "--profile", colima.InstanceName, "--",
		"sudo", "env", "DEBIAN_FRONTEND=noninteractive", "apt-get", "-q", "-o", "DPkg::Lock::Timeout=120"}
	// Without --error-on=any a failed fetch still exits 0 and the old kernel stays the newest one apt knows.
	if err := command.RunCommand("colima", append(aptGet, "update", "--error-on=any")...); err != nil {
		return constants.ErrOperationFailed("update VM package index", err)
	}
	if err := command.RunCommand("colima", append(aptGet, "install", "-y", "linux-image-virtual")...); err != nil {
		return constants.ErrOperationFailed("install VM kernel", err)
	}
	if err := command.RunCommand("colima", "restart", colima.InstanceName); err != nil {
		return constants.ErrOperationFailed("restart Colima to boot the new kernel", err)
	}

	installedLink, err := command.RunCommandOutput("colima", "ssh", "--profile", colima.InstanceName, "--", "readlink", "/boot/vmlinuz")
	if err != nil {
		return constants.ErrOperationFailed("read installed VM kernel", err)
	}
	bootedOutput, err := command.RunCommandOutput("colima", "ssh", "--profile", colima.InstanceName, "--", "uname", "-r")
	if err != nil {
		return constants.ErrOperationFailed("read booted VM kernel", err)
	}

	installed := strings.TrimPrefix(strings.TrimSpace(installedLink), "vmlinuz-")
	booted := strings.TrimSpace(bootedOutput)
	if booted != installed {
		return errorhandle.ErrInternalMsg(fmt.Sprintf("Colima VM booted kernel %s instead of the installed %s", booted, installed))
	}
	ui.Log.Info("Colima VM kernel upgraded", logger.Field{Key: "kernel", Value: installed})
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

		ui.Log.Info("Still waiting...", logger.Field{Key: "elapsed", Value: time.Since(startTime).Truncate(time.Second)})
		time.Sleep(10 * time.Second)
	}

	return constants.ErrTimeout("K3s cluster readiness", k3sReadyTimeout)
}

func normalizeLocalKubeconfig() error {
	ui.Log.Info("Normalizing local kubeconfig server to 127.0.0.1")

	clusterName, err := command.RunCommandOutput("kubectl", "config", "view", "--minify", "-o", "jsonpath={.clusters[0].name}")
	if err != nil {
		return constants.ErrOperationFailed("read current kubeconfig cluster name", err)
	}
	clusterName = strings.TrimSpace(clusterName)
	if clusterName == "" {
		return errorhandle.ErrInternalMsg("current kubeconfig cluster name is empty")
	}

	server, err := command.RunCommandOutput("kubectl", "config", "view", "--minify", "-o", "jsonpath={.clusters[0].cluster.server}")
	if err != nil {
		return constants.ErrOperationFailed("read current kubeconfig server", err)
	}
	server = strings.TrimSpace(server)

	parsed, err := url.Parse(server)
	if err != nil {
		return constants.ErrOperationFailed(fmt.Sprintf("parse kubeconfig server %q", server), err)
	}
	if parsed.Port() == "" {
		return errorhandle.ErrValidationFailed(fmt.Sprintf("kubeconfig server %q has no port", server))
	}

	parsed.Host = "127.0.0.1:" + parsed.Port()
	normalizedServer := parsed.String()
	if normalizedServer == server {
		ui.Log.Info("Local kubeconfig server already normalized", logger.Field{Key: "server", Value: server})
		return nil
	}

	if err := command.RunCommand("kubectl", "config", "set-cluster", clusterName, "--server", normalizedServer); err != nil {
		return constants.ErrOperationFailed("normalize kubeconfig server", err)
	}

	ui.Log.Info("Local kubeconfig server normalized", logger.Field{Key: "from", Value: server}, logger.Field{Key: "to", Value: normalizedServer})
	return nil
}

func getColimaIPAddress() (string, error) {
	ui.Log.Info("Getting Colima VM IP address...")

	output, err := command.RunCommandOutput("colima", "list", "--format", "{{.IPAddress}}")
	if err != nil {
		return "", constants.ErrOperationFailed("get Colima IP", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && line != "IPAddress" {
			ui.Log.Info("Found Colima IP", logger.Field{Key: "ip", Value: line})
			return line, nil
		}
	}

	return "", errorhandle.ErrNotFound("Colima VM IP address")
}

func disableTraefik() error {
	ui.Log.Info("Disabling default Traefik ingress controller...")

	if err := command.RunCommand("kubectl", "delete", "namespace", "traefik-system", "--ignore-not-found"); err != nil {
		ui.Log.Info("Traefik namespace deletion", logger.Field{Key: "result", Value: err})
	}

	if err := command.RunCommand("kubectl", "delete", "ingressclass", "traefik", "--ignore-not-found"); err != nil {
		ui.Log.Info("Traefik ingress class deletion", logger.Field{Key: "result", Value: err})
	}

	ui.Log.Info("Traefik disabled successfully")
	return nil
}

func setNodeLabel(envLabel string) error {
	ui.Log.Info("Setting node label...")

	nodeOutput, err := command.RunCommandOutput("kubectl", "get", "nodes", "-o", "jsonpath={.items[0].metadata.name}")
	if err != nil {
		return constants.ErrOperationFailed("get node name", err)
	}

	nodeName := strings.TrimSpace(nodeOutput)
	labelValue := fmt.Sprintf("env=%s", envLabel)

	return command.RunCommand("kubectl", "label", "node", nodeName, labelValue, "--overwrite")
}

func testKubectlAccess() error {
	ui.Log.Info("Testing kubectl access to K3s cluster...")

	if err := command.RunCommand("kubectl", "version", "--client"); err != nil {
		return constants.ErrOperationFailed("kubectl availability check", err)
	}

	if err := command.RunCommand("kubectl", "cluster-info"); err != nil {
		return constants.ErrOperationFailed("kubectl connection to K3s cluster", err)
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
		ui.Log.Warn("Health check failed", logger.Field{Key: "error", Value: err})
	}

	ui.Log.Info("Testing kubectl access...")
	if err := testKubectlAccess(); err != nil {
		ui.Log.Warn("kubectl access test failed", logger.Field{Key: "error", Value: err})
		ui.Log.Info("Tip: Check if ~/.kube/config exists and contains valid K3s cluster configuration")
	} else {
		ui.Log.Info("kubectl access is working correctly!")
	}

	if ip, err := getColimaIPAddress(); err == nil {
		ui.Log.Info("Colima VM IP", logger.Field{Key: "ip", Value: ip})
		ui.Log.Info("This IP will be used for LoadBalancer services")
	}

	ui.Log.Info("Colima K3s installation and setup completed successfully!")
	ui.Log.Info("Colima instance name", logger.Field{Key: "name", Value: colima.InstanceName})
	ui.Log.Info("Access your cluster with: kubectl get nodes")

	return nil
}

func setupK3sCluster() error {
	ui.Log.Info("Setting up Colima K3s cluster...")

	if err := stopExistingColima(); err != nil {
		return constants.ErrOperationFailed("stop existing Colima", err)
	}

	if err := startColimaWithK3s(); err != nil {
		return constants.ErrOperationFailed("start Colima with K3s", err)
	}

	if err := upgradeVMKernel(); err != nil {
		return constants.ErrOperationFailed("upgrade Colima VM kernel", err)
	}

	if err := normalizeLocalKubeconfig(); err != nil {
		return constants.ErrOperationFailed("normalize local kubeconfig", err)
	}

	if err := waitForK3sReady(); err != nil {
		return constants.ErrOperationFailed("K3s cluster readiness check", err)
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
