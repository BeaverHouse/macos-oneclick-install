package install

import (
	"austinhome/internal/command"
	"austinhome/internal/ui"

	"github.com/BeaverHouse/go-common/logger"
)

const (
	gitlabClusterSecretStoreURL = "https://raw.githubusercontent.com/BeaverHouse/cicd/refs/heads/main/charts/app-clustersecrets/resources/gitlab-clustersecretstore.yaml"
)

func SetupESOSecretStore(gitlabPAT string) error {
	ui.Log.Info("Setting up ESO SecretStore...")

	if err := createGitLabSecret(gitlabPAT); err != nil {
		return err
	}

	if err := applyClusterSecretStore(); err != nil {
		return err
	}

	ui.Log.Info("Successfully set up ESO SecretStore")
	return nil
}

func createGitLabSecret(pat string) error {
	ui.Log.Info("Creating GitLab ESO secret...")
	return command.RunCommand("kubectl", "create", "secret", "generic", "gitlab-eso-secret",
		"--namespace", esoNamespace,
		"--from-literal=token="+pat)
}

func applyClusterSecretStore() error {
	ui.Log.Info("Applying GitLab ClusterSecretStore...")
	return command.RunCommand("kubectl", "apply", "-f", gitlabClusterSecretStoreURL)
}

func verifyESOSecretStore() error {
	ui.Log.Info("Verifying ESO SecretStore setup...")

	ui.Log.Info("GitLab secret status:")
	if err := command.RunCommand("kubectl", "get", "secret", "gitlab-eso-secret", "-n", esoNamespace); err != nil {
		return err
	}

	ui.Log.Info("ClusterSecretStore status:")
	if err := command.RunCommand("kubectl", "get", "clustersecretstore"); err != nil {
		ui.Log.Warn("Failed to get ClusterSecretStore", logger.F("error", err))
	}

	return nil
}
