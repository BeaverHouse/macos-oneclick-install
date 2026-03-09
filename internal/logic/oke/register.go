package oke

import (
	"austinhome/internal/logic/common"
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	configKeyClusterOCID = "oke-cluster-ocid"
	configKeyRegion      = "oke-region"
	argocdNS             = "argo-project"
)

// PromptAndSaveOKEConfig ensures OCI CLI is set up, then asks for cluster OCID and region.
// Accepts a shared bufio.Reader to avoid stdin buffer conflicts.
func PromptAndSaveOKEConfig(reader *bufio.Reader) error {
	// 1. Ensure OCI CLI is installed
	if err := ensureOCICLI(); err != nil {
		return err
	}

	// 2. Check if ~/.oci/config exists; if not, run oci setup config
	if err := ensureOCIConfig(); err != nil {
		return err
	}

	// 3. Prompt for cluster OCID
	fmt.Print("Enter OKE cluster OCID (leave empty to skip OKE setup): ")
	ocidInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %v", err)
	}
	ocid := strings.TrimSpace(ocidInput)
	if ocid == "" {
		fmt.Println("⏭️  OKE setup skipped")
		return nil
	}

	// 4. Prompt for region
	fmt.Print("Enter OKE region (e.g., ap-chuncheon-1): ")
	regionInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %v", err)
	}
	region := strings.TrimSpace(regionInput)
	if region == "" {
		return fmt.Errorf("OKE region cannot be empty")
	}

	if err := common.ConfigSave(configKeyClusterOCID, ocid); err != nil {
		return err
	}
	if err := common.ConfigSave(configKeyRegion, region); err != nil {
		return err
	}

	fmt.Println("✅ OKE config saved")
	return nil
}

// ensureOCIConfig checks if ~/.oci/config exists. If not, runs `oci setup config` interactively.
func ensureOCIConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	ociConfigPath := home + "/.oci/config"
	if _, err := os.Stat(ociConfigPath); err == nil {
		fmt.Println("✅ OCI config already exists (~/.oci/config)")
		return nil
	}

	fmt.Println("📦 OCI config not found. Running 'oci setup config'...")
	fmt.Println("   (Follow the prompts to enter your tenancy OCID, user OCID, region, and API key path)")
	return common.RunCommandInteractive("oci", "setup", "config")
}

// loadOKEConfig reads cluster OCID and region from ~/.austinhome/.
func loadOKEConfig() (clusterOCID, region string, err error) {
	clusterOCID, err = common.ConfigLoad(configKeyClusterOCID)
	if err != nil {
		return "", "", fmt.Errorf("OKE cluster OCID not configured: %v\nRun 'austinhome install' first", err)
	}
	region, err = common.ConfigLoad(configKeyRegion)
	if err != nil {
		return "", "", fmt.Errorf("OKE region not configured: %v\nRun 'austinhome install' first", err)
	}
	return clusterOCID, region, nil
}

// Register sets up OKE kubeconfig and registers the cluster with ArgoCD.
func Register() error {
	fmt.Println("\n📦 Step: OKE cluster registration")

	// 0. Load OKE config
	clusterOCID, region, err := loadOKEConfig()
	if err != nil {
		return err
	}

	// 1. Ensure OCI CLI is available
	if err := ensureOCICLI(); err != nil {
		return err
	}

	// 2. Ensure ArgoCD CLI is available
	if err := ensureArgoCDCLI(); err != nil {
		return err
	}

	// 3. Create OKE kubeconfig
	if err := createOKEKubeconfig(clusterOCID, region); err != nil {
		return err
	}

	// 4. Get ArgoCD admin password
	adminPW, err := getArgoCDAdminPassword()
	if err != nil {
		return err
	}

	// 5. Login to ArgoCD (with retry)
	if err := argocdLogin(adminPW); err != nil {
		return err
	}

	// 6. Register OKE cluster
	okeContextName := "context-" + clusterOCID
	if err := argocdAddCluster(okeContextName); err != nil {
		return err
	}

	// 7. Verify
	if err := common.RunCommand("argocd", "cluster", "list"); err != nil {
		fmt.Printf("Warning: could not list ArgoCD clusters: %v\n", err)
	}

	fmt.Println("✅ OKE cluster registered with ArgoCD")
	return nil
}

func ensureOCICLI() error {
	if common.IsCommandAvailable("oci") {
		fmt.Println("✅ OCI CLI already installed")
		return nil
	}
	fmt.Println("📦 Installing OCI CLI via Homebrew...")
	return common.RunCommand("brew", "install", "oci-cli")
}

func ensureArgoCDCLI() error {
	if common.IsCommandAvailable("argocd") {
		fmt.Println("✅ ArgoCD CLI already installed")
		return nil
	}
	fmt.Println("📦 Installing ArgoCD CLI via Homebrew...")
	return common.RunCommand("brew", "install", "argocd")
}

func createOKEKubeconfig(clusterOCID, region string) error {
	fmt.Println("📦 Creating OKE kubeconfig...")
	return common.RunCommand("oci", "ce", "cluster", "create-kubeconfig",
		"--cluster-id", clusterOCID,
		"--region", region,
		"--token-version", "2.0.0",
		"--kube-endpoint", "PUBLIC_ENDPOINT",
	)
}

func getArgoCDAdminPassword() (string, error) {
	fmt.Println("🔑 Retrieving ArgoCD admin password...")

	// Switch to K3s context first
	if err := common.RunCommand("kubectl", "config", "use-context", "colima-k3s-homeserver"); err != nil {
		return "", fmt.Errorf("failed to switch to K3s context: %v", err)
	}

	output, err := common.RunCommandOutput("kubectl", "get", "secret",
		"argocd-initial-admin-secret", "-n", argocdNS,
		"-o", "jsonpath={.data.password}")
	if err != nil {
		return "", fmt.Errorf("failed to get ArgoCD admin secret: %v", err)
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(output))
	if err != nil {
		return "", fmt.Errorf("failed to decode ArgoCD password: %v", err)
	}

	return string(decoded), nil
}

func argocdLogin(password string) error {
	fmt.Println("🔑 Logging in to ArgoCD...")

	maxRetries := 12 // 12 * 10s = 2 minutes
	for i := range maxRetries {
		err := common.RunCommand("argocd", "login", "argocd.haulrest.me",
			"--username", "admin",
			"--password", password,
			"--grpc-web",
			"--insecure",
		)
		if err == nil {
			fmt.Println("✅ ArgoCD login successful")
			return nil
		}

		fmt.Printf("⏳ ArgoCD not ready yet, retrying... (%d/%d)\n", i+1, maxRetries)
		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("failed to login to ArgoCD after %d retries", maxRetries)
}

func argocdAddCluster(okeContextName string) error {
	fmt.Println("📦 Registering OKE cluster with ArgoCD...")
	return common.RunCommand("argocd", "cluster", "add", okeContextName,
		"--name", "oke",
		"--yes",
	)
}
