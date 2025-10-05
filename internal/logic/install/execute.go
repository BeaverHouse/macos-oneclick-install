package install

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func Execute() error {
	envLabel, err := getEnvironmentLabel()
	if err != nil {
		return err
	}

	gitlabPAT, err := getGitLabPAT()
	if err != nil {
		return err
	}

	// Validate prerequisites
	if err := validatePrerequisites(); err != nil {
		return err
	}

	// Install Colima if needed
	if err := installColimaIfNeeded(); err != nil {
		return err
	}

	// Setup Colima K3s cluster
	if err := setupK3sCluster(); err != nil {
		return err
	}

	// Enable essential addons and post-installation setup
	if err := enableEssentialAddons(); err != nil {
		return err
	}

	if err := setupPostInstallation(envLabel); err != nil {
		return err
	}

	// Install Helm
	if err := InstallHelm(); err != nil {
		return err
	}

	if err := verifyHelmInstallation(); err != nil {
		fmt.Printf("Warning: Helm verification failed: %v\n", err)
	}

	// Install MetalLB for LoadBalancer support
	if err := InstallMetalLB(); err != nil {
		return err
	}

	if err := verifyMetalLBInstallation(); err != nil {
		fmt.Printf("Warning: MetalLB verification failed: %v\n", err)
	}

	// Install NGINX Ingress Controller
	if err := InstallIngressNginx(); err != nil {
		return err
	}

	if err := verifyIngressNginxInstallation(); err != nil {
		fmt.Printf("Warning: Ingress Nginx verification failed: %v\n", err)
	}

	// Critical: Test ingress connectivity, fail installation if this doesn't work
	if err := VerifyIngressConnectivity(); err != nil {
		fmt.Printf("❌ Critical: Ingress connectivity verification failed: %v\n", err)
		fmt.Println("🛑 Installation aborted due to ingress connectivity issues")
		return err
	}

	// Install External Secrets Operator
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

	// Install Cert-Manager
	if err := InstallCertManager(); err != nil {
		return err
	}

	if err := verifyCertManagerInstallation(); err != nil {
		fmt.Printf("Warning: Cert-Manager verification failed: %v\n", err)
	}

	// Install ArgoCD
	if err := InstallArgoCD(); err != nil {
		return err
	}

	if err := verifyArgoCDInstallation(); err != nil {
		fmt.Printf("Warning: ArgoCD verification failed: %v\n", err)
	}

	// Final verification
	if err := verifyInstallation(); err != nil {
		return err
	}

	return nil
}

func getEnvironmentLabel() (string, error) {
	fmt.Print("Enter environment label for this cluster (e.g., dev, staging, prod): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %v", err)
	}

	envLabel := strings.TrimSpace(input)
	if envLabel == "" {
		envLabel = "dev" // default value
		fmt.Println("Using default label: dev")
	}

	fmt.Printf("✅ Environment label set to: %s\n", envLabel)
	return envLabel, nil
}

func getGitLabPAT() (string, error) {
	fmt.Print("Enter the GitLab PAT (Personal Access Token): ")

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read GitLab PAT: %v", err)
	}

	pat := strings.TrimSpace(input)
	if pat == "" {
		return "", fmt.Errorf("GitLab PAT cannot be empty")
	}

	fmt.Println("✅ GitLab PAT received")
	return pat, nil
}
