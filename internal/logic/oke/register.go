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
	okeContextAlias      = "oke"
)

// PromptAndSaveOKEConfig ensures OCI CLI is set up, then asks for cluster OCID and region.
// Accepts a shared bufio.Reader to avoid stdin buffer conflicts.
func PromptAndSaveOKEConfig(reader *bufio.Reader) error {
	// 1. Ensure OCI CLI is installed
	if err := ensureOCICLI(); err != nil {
		return err
	}

	// 2. Check if ~/.oci/config exists; if not, run oci setup config
	if err := ensureOCIConfig(reader); err != nil {
		return err
	}

	// 3. Prompt for cluster OCID
	fmt.Println("\n📋 OKE cluster OCID 확인 방법:")
	fmt.Println("   OCI Console → Developer Services → Kubernetes Clusters (OKE)")
	fmt.Println("   → 클러스터 선택 → Cluster Details 페이지의 OCID 복사")
	fmt.Print("\nEnter OKE cluster OCID (leave empty to skip OKE setup): ")
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
	fmt.Println("\n📋 Region 확인 방법:")
	fmt.Println("   OCI Console 우측 상단 리전 표시 확인 (e.g., ap-chuncheon-1, ap-seoul-1)")
	fmt.Print("\nEnter OKE region: ")
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

// ensureOCIConfig checks if ~/.oci/config exists. If not, guides user through oci setup config.
func ensureOCIConfig(reader *bufio.Reader) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	ociConfigPath := home + "/.oci/config"
	if _, err := os.Stat(ociConfigPath); err == nil {
		fmt.Println("✅ OCI config already exists (~/.oci/config)")
		return nil
	}

	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("  OCI CLI 초기 설정이 필요합니다")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("\n다음 정보를 미리 준비하세요:")
	fmt.Println("  1. Tenancy OCID  → OCI Console → Profile → Tenancy → OCID 복사")
	fmt.Println("  2. User OCID     → OCI Console → Profile → My Profile → OCID 복사")
	fmt.Println("  3. Region        → OCI Console 우측 상단 확인 (e.g., ap-chuncheon-1)")
	fmt.Println()
	fmt.Println("⚠️  설정 완료 후 생성되는 Public Key를 OCI Console에 등록해야 합니다:")
	fmt.Println("   OCI Console → Profile → My Profile → API Keys → Add API Key")
	fmt.Println("   → Paste Public Key → ~/.oci/oci_api_key_public.pem 내용 붙여넣기")

	fmt.Print("\n준비되었나요? (Y/n): ")
	input, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(input)) == "n" {
		return fmt.Errorf("OCI setup cancelled by user")
	}

	fmt.Println("\n📦 Running 'oci setup config'...")
	if err := common.RunCommandInteractive("oci", "setup", "config"); err != nil {
		return err
	}

	// After setup, remind about API key registration
	fmt.Println("\n" + strings.Repeat("=", 60))
	fmt.Println("  API Key 등록 확인")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println("\n지금 OCI Console에서 Public Key를 등록하세요:")
	fmt.Println("  1. OCI Console → Profile → My Profile → API Keys → Add API Key")
	fmt.Println("  2. Paste Public Key 선택")
	fmt.Printf("  3. 아래 명령어로 키 내용 확인: cat %s/.oci/oci_api_key_public.pem\n", home)
	fmt.Println("  4. 키 내용을 붙여넣고 Add 클릭")

	fmt.Print("\nAPI Key 등록을 완료했나요? (Y/n): ")
	input, _ = reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(input)) == "n" {
		return fmt.Errorf("API key registration not completed. Register the key and run 'austinhome install' again")
	}

	fmt.Println("✅ OCI config setup completed")
	return nil
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

	// 3. Snapshot existing contexts, then create OKE kubeconfig
	contextsBefore := getContextNames()

	if err := createOKEKubeconfig(clusterOCID, region); err != nil {
		return err
	}

	// 3b. Find newly created context and rename to "oke"
	newContext := findNewContext(contextsBefore, getContextNames())
	if newContext != "" {
		if err := renameContext(newContext, okeContextAlias); err != nil {
			fmt.Printf("Warning: failed to rename context %s → %s: %v\n", newContext, okeContextAlias, err)
		}
	} else {
		fmt.Println("Warning: could not detect new OKE context name")
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
	if err := argocdAddCluster(okeContextAlias); err != nil {
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

func getContextNames() map[string]bool {
	output, err := common.RunCommandOutput("kubectl", "config", "get-contexts", "-o", "name")
	if err != nil {
		return nil
	}
	contexts := make(map[string]bool)
	for _, name := range strings.Split(strings.TrimSpace(output), "\n") {
		name = strings.TrimSpace(name)
		if name != "" {
			contexts[name] = true
		}
	}
	return contexts
}

func findNewContext(before, after map[string]bool) string {
	for name := range after {
		if !before[name] {
			return name
		}
	}
	return ""
}

func renameContext(oldName, newName string) error {
	fmt.Printf("📝 Renaming context %s → %s\n", oldName, newName)
	return common.RunCommand("kubectl", "config", "rename-context", oldName, newName)
}

func createOKEKubeconfig(clusterOCID, region string) error {
	fmt.Println("📦 Creating OKE kubeconfig...")
	return common.RunCommand("oci", "ce", "cluster", "create-kubeconfig",
		"--cluster-id", clusterOCID,
		"--region", region,
		"--token-version", "2.0.0",
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
