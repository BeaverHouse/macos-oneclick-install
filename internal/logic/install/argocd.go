package install

import (
	"austinhome/internal/logic/common"
	"fmt"
	"time"
)

const (
	argoCDVersion     = "8.5.8"
	argoCDRepoName    = "argo"
	argoCDRepoURL     = "https://argoproj.github.io/argo-helm"
	argoCDNamespace   = "argo-project"
	argoCDMaxWaitTime = 3 * time.Minute
	oauthSecretURL    = "https://raw.githubusercontent.com/BeaverHouse/hybrid-cicd/refs/heads/main/charts/oss-argocd/resources/oauth-secret.yaml"
	argoCDValuesURL   = "https://raw.githubusercontent.com/BeaverHouse/hybrid-cicd/refs/heads/main/charts/oss-argocd/values.yaml"
)

func InstallArgoCD() error {
	fmt.Println("🚀 Installing ArgoCD...")

	if err := createArgoCDNamespace(); err != nil {
		return err
	}

	if err := applyOAuthSecret(); err != nil {
		return err
	}

	if err := addArgoCDRepo(); err != nil {
		return err
	}

	if err := updateHelmRepoForArgoCD(); err != nil {
		return err
	}

	if err := installArgoCDChart(); err != nil {
		return err
	}

	if err := common.WaitForPodsReady(argoCDNamespace, "app.kubernetes.io/name=argocd-server", argoCDMaxWaitTime); err != nil {
		return err
	}

	fmt.Println("✅ Successfully installed ArgoCD")
	return nil
}

func createArgoCDNamespace() error {
	fmt.Println("📋 Creating ArgoCD namespace...")
	// Using apply with a simple namespace manifest approach
	err := common.RunCommand("kubectl", "create", "namespace", argoCDNamespace)
	if err != nil {
		// Namespace might already exist, check if it exists
		checkErr := common.RunCommand("kubectl", "get", "namespace", argoCDNamespace)
		if checkErr != nil {
			return err // Return original error if namespace doesn't exist
		}
		fmt.Printf("Namespace %s already exists, continuing...\n", argoCDNamespace)
	}
	return nil
}

func applyOAuthSecret() error {
	fmt.Println("🔐 Applying OAuth secret...")
	return common.RunCommand("kubectl", "apply", "-f", oauthSecretURL)
}

func addArgoCDRepo() error {
	fmt.Println("📦 Adding ArgoCD Helm repository...")
	return common.RunCommand("helm", "repo", "add", argoCDRepoName, argoCDRepoURL)
}

func updateHelmRepoForArgoCD() error {
	fmt.Println("🔄 Updating Helm repositories...")
	return common.RunCommand("helm", "repo", "update")
}

func installArgoCDChart() error {
	fmt.Println("🚀 Installing ArgoCD chart...")
	return common.RunCommand("helm", "upgrade", "--install", "argocd",
		"argo/argo-cd",
		"--namespace", argoCDNamespace,
		"--create-namespace",
		"--values", argoCDValuesURL,
		"--version", argoCDVersion)
}

func verifyArgoCDInstallation() error {
	fmt.Println("🔍 Verifying ArgoCD installation...")

	fmt.Println("\n📋 ArgoCD pods status:")
	if err := common.RunCommand("kubectl", "get", "pods", "-n", argoCDNamespace); err != nil {
		return err
	}

	fmt.Println("\n🌐 ArgoCD service status:")
	if err := common.RunCommand("kubectl", "get", "service", "-n", argoCDNamespace); err != nil {
		fmt.Printf("Warning: failed to get ArgoCD service: %v\n", err)
	}

	fmt.Println("\n🚀 ArgoCD application status:")
	if err := common.RunCommand("kubectl", "get", "application", "-n", argoCDNamespace); err != nil {
		fmt.Printf("Info: No applications deployed yet\n")
	}

	return nil
}
