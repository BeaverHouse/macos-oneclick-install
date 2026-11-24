package install

import (
	"austinhome/internal/config"
	"austinhome/internal/constants"
	"austinhome/internal/oke"
	"austinhome/internal/ui"
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/BeaverHouse/go-common/errorhandle"
	"github.com/BeaverHouse/go-common/logger"
)

const defaultEnvLabel = "dev"

var stdinReader = bufio.NewReader(os.Stdin)

func Execute() error {
	envLabel, err := promptEnvironmentLabel()
	if err != nil {
		return err
	}
	gitlabPAT, err := promptGitLabPAT()
	if err != nil {
		return err
	}

	if err := oke.PromptAndSaveOKEConfig(stdinReader); err != nil {
		ui.Log.Warn("OKE config failed", logger.Field{Key: "error", Value: err})
	}

	if err := validatePrerequisites(); err != nil {
		return err
	}
	if err := installColimaIfNeeded(); err != nil {
		return err
	}
	if err := setupK3sCluster(); err != nil {
		return err
	}
	if err := setupPostInstallation(envLabel); err != nil {
		return err
	}

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

	if err := oke.Register(); err != nil {
		ui.Log.Warn("OKE registration failed", logger.Field{Key: "error", Value: err})
	}
	if err := oke.ExportKubeconfig(); err != nil {
		ui.Log.Warn("Kubeconfig export failed", logger.Field{Key: "error", Value: err})
	}

	return verifyInstallation()
}

func installHelm() error {
	if err := InstallHelm(); err != nil {
		return err
	}
	if err := verifyHelmInstallation(); err != nil {
		ui.Log.Warn("Helm verification failed", logger.Field{Key: "error", Value: err})
	}
	return nil
}

func installMetalLB() error {
	if err := InstallMetalLB(); err != nil {
		return err
	}
	if err := verifyMetalLBInstallation(); err != nil {
		ui.Log.Warn("MetalLB verification failed", logger.Field{Key: "error", Value: err})
	}
	return nil
}

func installGateway() error {
	if err := InstallNginxGateway(); err != nil {
		return err
	}
	if err := verifyNginxGatewayInstallation(); err != nil {
		ui.Log.Warn("NGINX Gateway Fabric verification failed", logger.Field{Key: "error", Value: err})
	}
	if err := VerifyGatewayConnectivity(); err != nil {
		ui.Log.Error("Gateway connectivity verification failed", logger.Field{Key: "error", Value: err})
		ui.Log.Error("Installation aborted due to gateway connectivity issues")
		return err
	}
	return nil
}

func installESO(gitlabPAT string) error {
	if err := InstallExternalSecretsOperator(); err != nil {
		return err
	}
	if err := verifyESOInstallation(); err != nil {
		ui.Log.Warn("ESO verification failed", logger.Field{Key: "error", Value: err})
	}
	if err := SetupESOSecretStore(gitlabPAT); err != nil {
		return err
	}
	if err := verifyESOSecretStore(); err != nil {
		ui.Log.Warn("ESO SecretStore verification failed", logger.Field{Key: "error", Value: err})
	}
	return nil
}

func installCertManager() error {
	if err := InstallCertManager(); err != nil {
		return err
	}
	if err := verifyCertManagerInstallation(); err != nil {
		ui.Log.Warn("Cert-Manager verification failed", logger.Field{Key: "error", Value: err})
	}
	return nil
}

func installArgoCD() error {
	if err := InstallArgoCD(); err != nil {
		return err
	}
	if err := verifyArgoCDInstallation(); err != nil {
		ui.Log.Warn("ArgoCD verification failed", logger.Field{Key: "error", Value: err})
	}
	return nil
}

func promptEnvironmentLabel() (string, error) {
	fmt.Print("Enter environment label for this cluster (e.g., dev, staging, prod): ")

	input, err := stdinReader.ReadString('\n')
	if err != nil {
		return "", constants.ErrOperationFailed("read input", err)
	}

	envLabel := strings.TrimSpace(input)
	if envLabel == "" {
		envLabel = defaultEnvLabel
		ui.Log.Info("Using default label", logger.Field{Key: "label", Value: defaultEnvLabel})
	}

	ui.Log.Info("Environment label set", logger.Field{Key: "label", Value: envLabel})
	return envLabel, nil
}

func promptGitLabPAT() (string, error) {
	fmt.Print("Enter the GitLab PAT (Personal Access Token): ")

	input, err := stdinReader.ReadString('\n')
	if err != nil {
		return "", constants.ErrOperationFailed("read GitLab PAT", err)
	}

	pat := strings.TrimSpace(input)
	if pat == "" {
		return "", errorhandle.ErrValidationFailed("GitLab PAT cannot be empty")
	}

	if err := config.Save("gitlab-pat", pat); err != nil {
		ui.Log.Warn("Failed to save GitLab PAT to config", logger.Field{Key: "error", Value: err})
	}

	ui.Log.Info("GitLab PAT received")
	return pat, nil
}

func ExecuteNonInteractive(envLabel, gitlabPAT string) error {
	ui.Log.Info("Non-interactive install", logger.Field{Key: "env", Value: envLabel})

	if err := validatePrerequisites(); err != nil {
		return err
	}
	if err := installColimaIfNeeded(); err != nil {
		return err
	}
	if err := setupK3sCluster(); err != nil {
		return err
	}
	if err := setupPostInstallation(envLabel); err != nil {
		return err
	}

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

	return verifyInstallation()
}
