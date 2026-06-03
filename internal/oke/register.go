package oke

import (
	"austinhome/internal/command"
	"austinhome/internal/config"
	"austinhome/internal/ui"
	"bufio"
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
	k3sContextAlias      = "colima-k3s-homeserver"

	argocdManagerSAName        = "argocd-manager"
	argocdManagerSANamespace   = "kube-system"
	argocdManagerCRBName       = "argocd-manager-role-binding"
	argocdManagerSecretName    = "argocd-manager-token"
	argocdClusterSecretName    = "cluster-oke"
	tokenPopulationMaxWaitTime = 2 * time.Minute
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

	if err := useContext(okeContextAlias); err != nil {
		return err
	}

	if err := recreateArgoCDManagerOnOKE(); err != nil {
		return err
	}

	token, caData, server, err := extractClusterCredentials()
	if err != nil {
		return err
	}

	if err := useContext(k3sContextAlias); err != nil {
		return err
	}

	if err := applyArgoCDClusterSecret(token, caData, server); err != nil {
		return err
	}
	if err := refreshApplicationSetController(); err != nil {
		return err
	}

	ui.Log.Info("OKE cluster registered with ArgoCD")
	return nil
}

func useContext(name string) error {
	ui.Log.Info("Switching kubectl context", logger.F("context", name))
	if err := command.RunCommand("kubectl", "config", "use-context", name); err != nil {
		return fmt.Errorf("failed to switch context to %s: %v", name, err)
	}
	return nil
}

// recreateArgoCDManagerOnOKE deletes any existing argocd-manager SA/CRB/Secret on OKE
// and creates fresh ones, which invalidates the previous bearer token. The Secret type
// kubernetes.io/service-account-token still issues long-lived tokens on K8s 1.24+.
// Must be called with kubectl context pointing at OKE.
// See https://kubernetes.io/docs/reference/access-authn-authz/service-accounts-admin/#manual-secret-management-for-serviceaccounts
func recreateArgoCDManagerOnOKE() error {
	ui.Log.Info("Recreating argocd-manager on OKE...")

	if err := command.RunCommand("kubectl", "delete", "clusterrolebinding", argocdManagerCRBName, "--ignore-not-found"); err != nil {
		return fmt.Errorf("failed to delete argocd-manager clusterrolebinding: %v", err)
	}
	if err := command.RunCommand("kubectl", "delete", "secret", argocdManagerSecretName, "-n", argocdManagerSANamespace, "--ignore-not-found"); err != nil {
		return fmt.Errorf("failed to delete argocd-manager secret: %v", err)
	}
	if err := command.RunCommand("kubectl", "delete", "serviceaccount", argocdManagerSAName, "-n", argocdManagerSANamespace, "--ignore-not-found"); err != nil {
		return fmt.Errorf("failed to delete argocd-manager serviceaccount: %v", err)
	}

	manifest := fmt.Sprintf(`---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: %s
  namespace: %s
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: %s
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: cluster-admin
subjects:
- kind: ServiceAccount
  name: %s
  namespace: %s
---
apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
  annotations:
    kubernetes.io/service-account.name: %s
type: kubernetes.io/service-account-token
`, argocdManagerSAName, argocdManagerSANamespace,
		argocdManagerCRBName,
		argocdManagerSAName, argocdManagerSANamespace,
		argocdManagerSecretName, argocdManagerSANamespace, argocdManagerSAName)

	if err := kubectlApplyStdin(manifest); err != nil {
		return fmt.Errorf("failed to apply argocd-manager manifests: %v", err)
	}
	return nil
}

func kubectlApplyStdin(manifest string) error {
	cmd := exec.Command("kubectl", "apply", "-f", "-")
	cmd.Stdin = strings.NewReader(manifest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// extractClusterCredentials waits for the SA token Secret to be populated,
// then returns (token, caData (base64), server URL).
// Must be called with kubectl context pointing at OKE.
func extractClusterCredentials() (string, string, string, error) {
	ui.Log.Info("Waiting for argocd-manager token to populate...")

	deadline := time.Now().Add(tokenPopulationMaxWaitTime)
	var tokenB64, caData string
	for time.Now().Before(deadline) {
		t, err := command.RunCommandOutput("kubectl", "get", "secret", argocdManagerSecretName,
			"-n", argocdManagerSANamespace, "-o", "jsonpath={.data.token}")
		if err == nil && strings.TrimSpace(t) != "" {
			tokenB64 = strings.TrimSpace(t)
			c, err := command.RunCommandOutput("kubectl", "get", "secret", argocdManagerSecretName,
				"-n", argocdManagerSANamespace, "-o", "jsonpath={.data.ca\\.crt}")
			if err == nil && strings.TrimSpace(c) != "" {
				caData = strings.TrimSpace(c)
				break
			}
		}
		time.Sleep(2 * time.Second)
	}

	if tokenB64 == "" || caData == "" {
		return "", "", "", fmt.Errorf("token/ca did not populate within %v", tokenPopulationMaxWaitTime)
	}

	tokenBytes, err := base64.StdEncoding.DecodeString(tokenB64)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to decode token: %v", err)
	}

	server, err := command.RunCommandOutput("kubectl", "config", "view",
		"-o", "jsonpath={.clusters[?(@.name==\""+getCurrentClusterName()+"\")].cluster.server}")
	if err != nil || strings.TrimSpace(server) == "" {
		return "", "", "", fmt.Errorf("failed to get OKE server URL: %v", err)
	}

	return string(tokenBytes), caData, strings.TrimSpace(server), nil
}

func getCurrentClusterName() string {
	out, err := command.RunCommandOutput("kubectl", "config", "view",
		"-o", "jsonpath={.contexts[?(@.name==\""+okeContextAlias+"\")].context.cluster}")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

// applyArgoCDClusterSecret creates the ArgoCD cluster secret on the K3s cluster.
// bearerToken is the raw token string; caData stays base64-encoded as it appears
// in the source SA Secret.
// Must be called with kubectl context pointing at K3s.
// See https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#clusters
// Reference: https://medium.com/pickme-engineering-blog/how-to-connect-an-external-kubernetes-cluster-to-argo-cd-using-bearer-token-authentication-d9ab093f081d
func applyArgoCDClusterSecret(token, caDataB64, server string) error {
	ui.Log.Info("Applying ArgoCD cluster secret on K3s...", logger.F("server", server))

	configJSON := fmt.Sprintf(`{"bearerToken":%q,"tlsClientConfig":{"caData":%q}}`, token, caDataB64)

	manifest := fmt.Sprintf(`---
apiVersion: v1
kind: Secret
metadata:
  name: %s
  namespace: %s
  labels:
    argocd.argoproj.io/secret-type: cluster
type: Opaque
stringData:
  name: %s
  server: %s
  config: |
    %s
`, argocdClusterSecretName, argocdNS, okeContextAlias, server, configJSON)

	if err := kubectlApplyStdin(manifest); err != nil {
		return fmt.Errorf("failed to apply ArgoCD cluster secret: %v", err)
	}
	return nil
}

func refreshApplicationSetController() error {
	ui.Log.Info("Refreshing ArgoCD ApplicationSet controller...")
	if err := command.RunCommand("kubectl", "rollout", "restart", "deployment", "argocd-applicationset-controller", "-n", argocdNS); err != nil {
		return fmt.Errorf("failed to restart ApplicationSet controller: %v", err)
	}
	if err := command.RunCommand("kubectl", "rollout", "status", "deployment", "argocd-applicationset-controller", "-n", argocdNS, "--timeout=120s"); err != nil {
		return fmt.Errorf("ApplicationSet controller did not become ready: %v", err)
	}
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
