package install

import (
	"austinhome/internal/command"
	"austinhome/internal/ui"
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
	ui.Log.Info("Installing External Secrets Operator...")

	if err := addESORepo(); err != nil {
		return err
	}

	if err := updateHelmRepoForESO(); err != nil {
		return err
	}

	if err := installESOChart(); err != nil {
		return err
	}

	if err := command.WaitForPodsReady(esoNamespace, "", esoMaxWaitTime); err != nil {
		return err
	}

	ui.Log.Info("Successfully installed External Secrets Operator")
	return nil
}

func addESORepo() error {
	ui.Log.Info("Adding External Secrets Helm repository...")
	return command.RunCommand("helm", "repo", "add", esoRepoName, esoRepoURL)
}

func updateHelmRepoForESO() error {
	ui.Log.Info("Updating Helm repositories...")
	return command.RunCommand("helm", "repo", "update")
}

func installESOChart() error {
	ui.Log.Info("Installing External Secrets chart...")
	return command.RunCommand("helm", "install", "external-secrets",
		"external-secrets/external-secrets",
		"--namespace", esoNamespace,
		"--version", esoVersion,
		"--create-namespace")
}

func verifyESOInstallation() error {
	ui.Log.Info("Verifying External Secrets Operator installation...")

	ui.Log.Info("External Secrets pods status:")
	if err := command.RunCommand("kubectl", "get", "pods", "-n", esoNamespace); err != nil {
		return err
	}

	return nil
}
