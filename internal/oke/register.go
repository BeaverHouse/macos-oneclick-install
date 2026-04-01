package oke

import (
	"austinhome/internal/command"
	"austinhome/internal/config"
	"austinhome/internal/ui"
	"bufio"
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/BeaverHouse/go-common/logger"
)

const (
	configKeyClusterOCID = "oke-cluster-ocid"
	configKeyRegion      = "oke-region"
	argocdNS             = "argo-project"
	okeContextAlias      = "oke"
)

func PromptAndSaveOKEConfig(reader *bufio.Reader) error {
	if err := ensureOCICLI(); err != nil {
		return err
	}

	if err := ensureOCIConfig(reader); err != nil {
		return err
	}

	ui.Log.Info("OKE cluster OCID 확인 방법:")
	ui.Log.Info("   OCI Console -> Developer Services -> Kubernetes Clusters (OKE)")
	ui.Log.Info("   -> 클러스터 선택 -> Cluster Details 페이지의 OCID 복사")
	fmt.Print("\nEnter OKE cluster OCID (leave empty to skip OKE setup): ")
	ocidInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %v", err)
	}
	ocid := strings.TrimSpace(ocidInput)
	if ocid == "" {
		ui.Log.Info("OKE setup skipped")
		return nil
	}

	ui.Log.Info("Region 확인 방법:")
	ui.Log.Info("   OCI Console 우측 상단 리전 표시 확인 (e.g., ap-chuncheon-1, ap-seoul-1)")
	fmt.Print("\nEnter OKE region: ")
	regionInput, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read input: %v", err)
	}
	region := strings.TrimSpace(regionInput)
	if region == "" {
		return fmt.Errorf("OKE region cannot be empty")
	}

	if err := config.Save(configKeyClusterOCID, ocid); err != nil {
		return err
	}
	if err := config.Save(configKeyRegion, region); err != nil {
		return err
	}

	ui.Log.Info("OKE config saved")
	return nil
}

func ensureOCIConfig(reader *bufio.Reader) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	ociConfigPath := home + "/.oci/config"
	if _, err := os.Stat(ociConfigPath); err == nil {
		ui.Log.Info("OCI config already exists (~/.oci/config)")
		return nil
	}

	ui.Log.Info(strings.Repeat("=", 60))
	ui.Log.Info("  OCI CLI 초기 설정이 필요합니다")
	ui.Log.Info(strings.Repeat("=", 60))
	ui.Log.Info("다음 정보를 미리 준비하세요:")
	ui.Log.Info("  1. Tenancy OCID  -> OCI Console -> Profile -> Tenancy -> OCID 복사")
	ui.Log.Info("  2. User OCID     -> OCI Console -> Profile -> My Profile -> OCID 복사")
	ui.Log.Info("  3. Region        -> OCI Console 우측 상단 확인 (e.g., ap-chuncheon-1)")
	ui.Log.Warn("설정 완료 후 생성되는 Public Key를 OCI Console에 등록해야 합니다:")
	ui.Log.Warn("   OCI Console -> Profile -> My Profile -> API Keys -> Add API Key")
	ui.Log.Warn("   -> Paste Public Key -> ~/.oci/oci_api_key_public.pem 내용 붙여넣기")

	fmt.Print("\n준비되었나요? (Y/n): ")
	input, _ := reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(input)) == "n" {
		return fmt.Errorf("OCI setup cancelled by user")
	}

	ui.Log.Info("Running 'oci setup config'...")
	if err := command.RunCommandInteractive("oci", "setup", "config"); err != nil {
		return err
	}

	ui.Log.Info(strings.Repeat("=", 60))
	ui.Log.Info("  API Key 등록 확인")
	ui.Log.Info(strings.Repeat("=", 60))
	ui.Log.Info("지금 OCI Console에서 Public Key를 등록하세요:")
	ui.Log.Info("  1. OCI Console -> Profile -> My Profile -> API Keys -> Add API Key")
	ui.Log.Info("  2. Paste Public Key 선택")
	ui.Log.Info("  3. 아래 명령어로 키 내용 확인", logger.F("command", fmt.Sprintf("cat %s/.oci/oci_api_key_public.pem", home)))
	ui.Log.Info("  4. 키 내용을 붙여넣고 Add 클릭")

	fmt.Print("\nAPI Key 등록을 완료했나요? (Y/n): ")
	input, _ = reader.ReadString('\n')
	if strings.TrimSpace(strings.ToLower(input)) == "n" {
		return fmt.Errorf("API key registration not completed. Register the key and run 'austinhome install' again")
	}

	ui.Log.Info("OCI config setup completed")
	return nil
}

func loadOKEConfig() (clusterOCID, region string, err error) {
	clusterOCID, err = config.Load(configKeyClusterOCID)
	if err != nil {
		return "", "", fmt.Errorf("OKE cluster OCID not configured: %v\nRun 'austinhome install' first", err)
	}
	region, err = config.Load(configKeyRegion)
	if err != nil {
		return "", "", fmt.Errorf("OKE region not configured: %v\nRun 'austinhome install' first", err)
	}
	return clusterOCID, region, nil
}

func Register() error {
	ui.Log.Info("Step: OKE cluster registration")

	clusterOCID, region, err := loadOKEConfig()
	if err != nil {
		return err
	}

	if err := ensureOCICLI(); err != nil {
		return err
	}

	if err := ensureArgoCDCLI(); err != nil {
		return err
	}

	contextsBefore := getContextNames()

	if err := createOKEKubeconfig(clusterOCID, region); err != nil {
		return err
	}

	newContext := findNewContext(contextsBefore, getContextNames())
	if newContext != "" {
		if err := renameContext(newContext, okeContextAlias); err != nil {
			ui.Log.Warn("Failed to rename context", logger.F("from", newContext), logger.F("to", okeContextAlias), logger.F("error", err))
		}
	} else {
		ui.Log.Warn("Could not detect new OKE context name")
	}

	adminPW, err := getArgoCDAdminPassword()
	if err != nil {
		return err
	}

	if err := argocdLogin(adminPW); err != nil {
		return err
	}

	if err := argocdAddCluster(okeContextAlias); err != nil {
		return err
	}

	if err := command.RunCommand("argocd", "cluster", "list"); err != nil {
		ui.Log.Warn("Could not list ArgoCD clusters", logger.F("error", err))
	}

	ui.Log.Info("OKE cluster registered with ArgoCD")
	return nil
}

func ensureOCICLI() error {
	if command.IsCommandAvailable("oci") {
		ui.Log.Info("OCI CLI already installed")
		return nil
	}
	ui.Log.Info("Installing OCI CLI via Homebrew...")
	return command.RunCommand("brew", "install", "oci-cli")
}

func ensureArgoCDCLI() error {
	if command.IsCommandAvailable("argocd") {
		ui.Log.Info("ArgoCD CLI already installed")
		return nil
	}
	ui.Log.Info("Installing ArgoCD CLI via Homebrew...")
	return command.RunCommand("brew", "install", "argocd")
}

func getContextNames() map[string]bool {
	output, err := command.RunCommandOutput("kubectl", "config", "get-contexts", "-o", "name")
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
	ui.Log.Info("Renaming context", logger.F("from", oldName), logger.F("to", newName))
	return command.RunCommand("kubectl", "config", "rename-context", oldName, newName)
}

func createOKEKubeconfig(clusterOCID, region string) error {
	ui.Log.Info("Creating OKE kubeconfig...")
	return command.RunCommand("oci", "ce", "cluster", "create-kubeconfig",
		"--cluster-id", clusterOCID,
		"--region", region,
		"--token-version", "2.0.0",
	)
}

func getArgoCDAdminPassword() (string, error) {
	ui.Log.Info("Retrieving ArgoCD admin password...")

	if err := command.RunCommand("kubectl", "config", "use-context", "colima-k3s-homeserver"); err != nil {
		return "", fmt.Errorf("failed to switch to K3s context: %v", err)
	}

	output, err := command.RunCommandOutput("kubectl", "get", "secret",
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
	ui.Log.Info("Logging in to ArgoCD via port-forward...")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pfCmd := exec.CommandContext(ctx, "kubectl", "port-forward", "svc/argocd-server", "-n", argocdNS, "18443:443")
	pfCmd.Stdout = nil
	pfCmd.Stderr = nil
	if err := pfCmd.Start(); err != nil {
		return fmt.Errorf("failed to start port-forward: %v", err)
	}
	defer func() {
		cancel()
		pfCmd.Wait()
	}()

	time.Sleep(3 * time.Second)

	maxRetries := 12
	for i := range maxRetries {
		err := command.RunCommand("argocd", "login", "localhost:18443",
			"--username", "admin",
			"--password", password,
			"--insecure",
		)
		if err == nil {
			ui.Log.Info("ArgoCD login successful")
			return nil
		}

		ui.Log.Info("ArgoCD not ready yet, retrying...", logger.F("attempt", i+1), logger.F("maxRetries", maxRetries))
		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("failed to login to ArgoCD after %d retries", maxRetries)
}

func argocdAddCluster(okeContextName string) error {
	ui.Log.Info("Registering OKE cluster with ArgoCD...")
	return command.RunCommand("argocd", "cluster", "add", okeContextName,
		"--name", "oke",
		"--yes",
	)
}
