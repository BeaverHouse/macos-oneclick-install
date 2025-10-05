package install

import (
	"austinhome/internal/logic/common"
	"fmt"
	"time"
)

const (
	esoVersion     = "0.20.2"
	esoRepoName    = "external-secrets"
	esoRepoURL     = "https://charts.external-secrets.io"
	esoNamespace   = "external-secrets"
	esoMaxWaitTime = 3 * time.Minute
)

func InstallExternalSecretsOperator() error {
	fmt.Println("🔐 Installing External Secrets Operator...")

	if err := addESORepo(); err != nil {
		return err
	}

	if err := updateHelmRepoForESO(); err != nil {
		return err
	}

	if err := installESOChart(); err != nil {
		return err
	}

	if err := common.WaitForPodsReady(esoNamespace, "", esoMaxWaitTime); err != nil {
		return err
	}

	fmt.Println("✅ Successfully installed External Secrets Operator")
	return nil
}

func addESORepo() error {
	fmt.Println("📦 Adding External Secrets Helm repository...")
	return common.RunCommand("helm", "repo", "add", esoRepoName, esoRepoURL)
}

func updateHelmRepoForESO() error {
	fmt.Println("🔄 Updating Helm repositories...")
	return common.RunCommand("helm", "repo", "update")
}

func installESOChart() error {
	fmt.Println("🚀 Installing External Secrets chart...")
	return common.RunCommand("helm", "install", "external-secrets",
		"external-secrets/external-secrets",
		"--namespace", esoNamespace,
		"--version", esoVersion,
		"--create-namespace")
}

func verifyESOInstallation() error {
	fmt.Println("🔍 Verifying External Secrets Operator installation...")

	fmt.Println("\n📋 External Secrets pods status:")
	if err := common.RunCommand("kubectl", "get", "pods", "-n", esoNamespace); err != nil {
		return err
	}

	return nil
}
