package install

import (
	"austinhome/internal/logic/common"
	"austinhome/internal/logic/oke"
	"bufio"
	"fmt"
	"os"
	"strings"
)

const defaultEnvLabel = "dev"

// stdinReader is shared across all prompts to avoid bufio buffer conflicts.
var stdinReader = bufio.NewReader(os.Stdin)

// Execute runs the full installation pipeline for K3s cluster on macOS.
func Execute() error {
	// Step 1: Gather user input
	envLabel, err := promptEnvironmentLabel()
	if err != nil {
		return err
	}
	gitlabPAT, err := promptGitLabPAT()
	if err != nil {
		return err
	}

	// Step 1b: OKE config (optional, saved for reinstall)
	if err := oke.PromptAndSaveOKEConfig(stdinReader); err != nil {
		fmt.Printf("Warning: OKE config failed: %v\n", err)
	}

	// Step 2: Setup Colima + K3s cluster
	if err := validatePrerequisites(); err != nil {
		return err
	}
	if err := installColimaIfNeeded(); err != nil {
		return err
	}
	if err := setupK3sCluster(); err != nil {
		return err
	}
	if err := enableEssentialAddons(); err != nil {
		return err
	}
	if err := setupPostInstallation(envLabel); err != nil {
		return err
	}

	// Step 3: Install infrastructure components
	if err := installHelm(); err != nil {
		return err
	}
	if err := installMetalLB(); err != nil {
		return err
	}
	if err := installGateway(); err != nil {
		return err
	}
	if err := installESO(gitlabPAT); err != nil {
		return err
	}
	if err := installCertManager(); err != nil {
		return err
	}
	if err := installArgoCD(); err != nil {
		return err
	}

	// Step 4: Final verification
	return verifyInstallation()
}

// Infrastructure component installers with verification

func installHelm() error {
	if err := InstallHelm(); err != nil {
		return err
	}
	if err := verifyHelmInstallation(); err != nil {
		fmt.Printf("Warning: Helm verification failed: %v\n", err)
	}
	return nil
}

func installMetalLB() error {
	if err := InstallMetalLB(); err != nil {
		return err
	}
	if err := verifyMetalLBInstallation(); err != nil {
		fmt.Printf("Warning: MetalLB verification failed: %v\n", err)
	}
	return nil
}

func installGateway() error {
	if err := InstallNginxGateway(); err != nil {
		return err
	}
	if err := verifyNginxGatewayInstallation(); err != nil {
		fmt.Printf("Warning: NGINX Gateway Fabric verification failed: %v\n", err)
	}
	// Gateway connectivity is critical - fail installation if this doesn't work
	if err := VerifyGatewayConnectivity(); err != nil {
		fmt.Printf("❌ Critical: Gateway connectivity verification failed: %v\n", err)
		fmt.Println("🛑 Installation aborted due to gateway connectivity issues")
		return err
	}
	return nil
}

func installESO(gitlabPAT string) error {
	if err := InstallExternalSecretsOperator(); err != nil {
		return err
	}
	if err := verifyESOInstallation(); err != nil {
		fmt.Printf("Warning: ESO verification failed: %v\n", err)
	}
	if err := SetupESOSecretStore(gitlabPAT); err != nil {
		return err
	}
	if err := verifyESOSecretStore(); err != nil {
		fmt.Printf("Warning: ESO SecretStore verification failed: %v\n", err)
	}
	return nil
}

func installCertManager() error {
	if err := InstallCertManager(); err != nil {
		return err
	}
	if err := verifyCertManagerInstallation(); err != nil {
		fmt.Printf("Warning: Cert-Manager verification failed: %v\n", err)
	}
	return nil
}

func installArgoCD() error {
	if err := InstallArgoCD(); err != nil {
		return err
	}
	if err := verifyArgoCDInstallation(); err != nil {
		fmt.Printf("Warning: ArgoCD verification failed: %v\n", err)
	}
	return nil
}

// User input prompts

func promptEnvironmentLabel() (string, error) {
	fmt.Print("Enter environment label for this cluster (e.g., dev, staging, prod): ")

	input, err := stdinReader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %v", err)
	}

	envLabel := strings.TrimSpace(input)
	if envLabel == "" {
		envLabel = defaultEnvLabel
		fmt.Printf("Using default label: %s\n", defaultEnvLabel)
	}

	fmt.Printf("✅ Environment label set to: %s\n", envLabel)
	return envLabel, nil
}

func promptGitLabPAT() (string, error) {
	fmt.Print("Enter the GitLab PAT (Personal Access Token): ")

	input, err := stdinReader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read GitLab PAT: %v", err)
	}

	pat := strings.TrimSpace(input)
	if pat == "" {
		return "", fmt.Errorf("GitLab PAT cannot be empty")
	}

	// Save PAT to config for future non-interactive use (reinstall)
	if err := common.ConfigSave("gitlab-pat", pat); err != nil {
		fmt.Printf("Warning: failed to save GitLab PAT to config: %v\n", err)
	}

	fmt.Println("✅ GitLab PAT received")
	return pat, nil
}

// ExecuteNonInteractive runs the full installation pipeline without user prompts.
// Used by the reinstall command.
func ExecuteNonInteractive(envLabel, gitlabPAT string) error {
	fmt.Printf("📦 Non-interactive install (env=%s)\n", envLabel)

	// Step 2: Setup Colima + K3s cluster
	if err := validatePrerequisites(); err != nil {
		return err
	}
	if err := installColimaIfNeeded(); err != nil {
		return err
	}
	if err := setupK3sCluster(); err != nil {
		return err
	}
	if err := enableEssentialAddons(); err != nil {
		return err
	}
	if err := setupPostInstallation(envLabel); err != nil {
		return err
	}

	// Step 3: Install infrastructure components
	if err := installHelm(); err != nil {
		return err
	}
	if err := installMetalLB(); err != nil {
		return err
	}
	if err := installGateway(); err != nil {
		return err
	}
	if err := installESO(gitlabPAT); err != nil {
		return err
	}
	if err := installCertManager(); err != nil {
		return err
	}
	if err := installArgoCD(); err != nil {
		return err
	}

	// Step 4: Final verification
	return verifyInstallation()
}
