package install

import (
	"austinhome/internal/logic/common"
	"fmt"
	"strings"
	"time"
)

// Colima and K3s configuration variables
const (
	colimaName   = "k3s-homeserver"
	colimaCPUs   = "4"
	colimaMemory = "8"

	k3sReadyTimeout = 180 * time.Second
)

func validatePrerequisites() error {
	if !common.IsCommandAvailable("brew") {
		return fmt.Errorf("Homebrew is required but not installed. Visit https://brew.sh/ to install it")
	}
	return nil
}

func installColimaIfNeeded() error {
	if !common.IsCommandAvailable("colima") {
		fmt.Println("🔧 Installing Colima...")
		if err := common.RunCommand("brew", "install", "colima"); err != nil {
			return fmt.Errorf("failed to install Colima: %v", err)
		}
	} else {
		fmt.Println("✅ Colima is already installed")
	}
	return nil
}

func stopExistingColima() error {
	fmt.Println("🛑 Stopping existing Colima instances if any...")

	// Check if Colima is running
	if err := common.RunCommand("colima", "status", colimaName); err == nil {
		fmt.Printf("🗑️ Stopping existing Colima instance: %s\n", colimaName)
		if err := common.RunCommand("colima", "stop", colimaName); err != nil {
			fmt.Printf("Warning: failed to stop Colima: %v\n", err)
		}

		// Delete the instance
		fmt.Printf("🗑️ Deleting existing Colima instance: %s\n", colimaName)
		if err := common.RunCommand("colima", "delete", colimaName, "--force"); err != nil {
			fmt.Printf("Warning: failed to delete Colima: %v\n", err)
		}
	} else {
		fmt.Println("ℹ️ No existing Colima instance found")
	}

	return nil
}

func startColimaWithK3s() error {
	fmt.Println("🚀 Starting Colima with Kubernetes (K3s) enabled...")

	// Start Colima with containerd runtime and bridged network mode
	err := common.RunCommand("colima", "start", colimaName,
		"--cpu", colimaCPUs,
		"--memory", colimaMemory,
		"--runtime", "containerd",
		"--network-address",
		"--network-mode", "bridged",
		"--network-interface", "en1",
		"--kubernetes")

	if err != nil {
		return fmt.Errorf("failed to start Colima with K3s: %v", err)
	}

	fmt.Println("✅ Colima with K3s started successfully")
	return nil
}

func waitForK3sReady() error {
	fmt.Println("⏳ Waiting for K3s cluster to be ready...")

	startTime := time.Now()
	for time.Since(startTime) < k3sReadyTimeout {
		// Check if kubectl can connect to the cluster
		if err := common.RunCommand("kubectl", "get", "nodes"); err == nil {
			fmt.Println("✅ K3s cluster is ready!")
			return nil
		}

		fmt.Printf("⏳ Still waiting... (%v elapsed)\n", time.Since(startTime).Truncate(time.Second))
		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("timeout: K3s cluster not ready after %v", k3sReadyTimeout)
}

func getColimaIPAddress() (string, error) {
	fmt.Println("🔍 Getting Colima VM IP address...")

	// Get Colima VM IP
	output, err := common.RunCommandOutput("colima", "list", "--format", "{{.IPAddress}}")
	if err != nil {
		return "", fmt.Errorf("failed to get Colima IP: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && line != "IPAddress" {
			fmt.Printf("✅ Found Colima IP: %s\n", line)
			return line, nil
		}
	}

	return "", fmt.Errorf("could not find Colima VM IP address")
}

func disableTraefik() error {
	fmt.Println("🚫 Disabling default Traefik ingress controller...")

	// Delete Traefik namespace if it exists
	if err := common.RunCommand("kubectl", "delete", "namespace", "traefik-system", "--ignore-not-found"); err != nil {
		fmt.Printf("Info: Traefik namespace deletion: %v\n", err)
	}

	// Delete Traefik ingress class if it exists
	if err := common.RunCommand("kubectl", "delete", "ingressclass", "traefik", "--ignore-not-found"); err != nil {
		fmt.Printf("Info: Traefik ingress class deletion: %v\n", err)
	}

	fmt.Println("✅ Traefik disabled successfully")
	return nil
}

func enableEssentialAddons() error {
	fmt.Println("🔧 Installing essential addons...")

	// Install metrics-server if not already present
	fmt.Println("📊 Installing metrics-server...")
	if err := common.RunCommand("kubectl", "apply", "-f",
		"https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml"); err != nil {
		fmt.Printf("Warning: failed to install metrics-server: %v\n", err)
	} else {
		fmt.Println("✅ Metrics-server installed")
	}

	return nil
}

func setNodeLabel(envLabel string) error {
	fmt.Println("🏷️ Setting node label...")

	// Get node name first
	nodeOutput, err := common.RunCommandOutput("kubectl", "get", "nodes", "-o", "jsonpath={.items[0].metadata.name}")
	if err != nil {
		return fmt.Errorf("failed to get node name: %v", err)
	}

	nodeName := strings.TrimSpace(nodeOutput)
	labelValue := fmt.Sprintf("env=%s", envLabel)

	return common.RunCommand("kubectl", "label", "node", nodeName, labelValue, "--overwrite")
}

func testKubectlAccess() error {
	fmt.Println("🧪 Testing kubectl access to K3s cluster...")

	if err := common.RunCommand("kubectl", "version", "--client"); err != nil {
		return fmt.Errorf("kubectl not available: %v", err)
	}

	if err := common.RunCommand("kubectl", "cluster-info"); err != nil {
		return fmt.Errorf("kubectl cannot connect to K3s cluster: %v", err)
	}

	return nil
}

func verifyInstallation() error {
	fmt.Println("✅ Final verification - checking nodes, labels, and health...")

	fmt.Println("\n📋 Node information with labels:")
	if err := common.RunCommand("kubectl", "get", "nodes", "--show-labels"); err != nil {
		return err
	}

	fmt.Println("\n🏥 K3s cluster health status:")
	if err := common.RunCommand("kubectl", "get", "pods", "--all-namespaces"); err != nil {
		fmt.Printf("Warning: health check failed: %v\n", err)
	}

	fmt.Println("\n🔄 Testing kubectl access...")
	if err := testKubectlAccess(); err != nil {
		fmt.Printf("Warning: kubectl access test failed: %v\n", err)
		fmt.Println("💡 Tip: Check if ~/.kube/config exists and contains valid K3s cluster configuration")
	} else {
		fmt.Println("✅ kubectl access is working correctly!")
	}

	// Get and display Colima IP
	if ip, err := getColimaIPAddress(); err == nil {
		fmt.Printf("\n🌐 Colima VM IP: %s\n", ip)
		fmt.Println("📝 This IP will be used for LoadBalancer services")
	}

	fmt.Println("\n🎉 Colima K3s installation and setup completed successfully!")
	fmt.Printf("📝 Colima instance name: %s\n", colimaName)
	fmt.Println("📝 Access your cluster with: kubectl get nodes")

	return nil
}

// Main setup functions
func setupK3sCluster() error {
	fmt.Println("⚙️ Setting up Colima K3s cluster...")

	if err := stopExistingColima(); err != nil {
		return fmt.Errorf("failed to stop existing Colima: %v", err)
	}

	if err := startColimaWithK3s(); err != nil {
		return fmt.Errorf("failed to start Colima with K3s: %v", err)
	}

	if err := waitForK3sReady(); err != nil {
		return fmt.Errorf("K3s cluster not ready: %v", err)
	}

	return nil
}

func setupPostInstallation(envLabel string) error {
	fmt.Println("⚙️ Setting up post-installation configuration...")

	// Colima automatically configures kubectl context, so no manual kubeconfig setup needed
	fmt.Println("✅ kubectl context automatically configured by Colima")

	if err := disableTraefik(); err != nil {
		return err
	}

	if err := setNodeLabel(envLabel); err != nil {
		return err
	}

	return nil
}
