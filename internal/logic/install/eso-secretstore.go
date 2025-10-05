package install

import (
	"austinhome/internal/logic/common"
	"fmt"
)

const (
	gitlabClusterSecretStoreURL = "https://raw.githubusercontent.com/BeaverHouse/hybrid-cicd/refs/heads/main/charts/app-clustersecrets/resources/gitlab-clustersecretstore.yaml"
)

func SetupESOSecretStore(gitlabPAT string) error {
	fmt.Println("🔑 Setting up ESO SecretStore...")

	if err := createGitLabSecret(gitlabPAT); err != nil {
		return err
	}

	if err := applyClusterSecretStore(); err != nil {
		return err
	}

	fmt.Println("✅ Successfully set up ESO SecretStore")
	return nil
}

func createGitLabSecret(pat string) error {
	fmt.Println("🔐 Creating GitLab ESO secret...")
	return common.RunCommand("kubectl", "create", "secret", "generic", "gitlab-eso-secret",
		"--namespace", esoNamespace,
		"--from-literal=token="+pat)
}

func applyClusterSecretStore() error {
	fmt.Println("📋 Applying GitLab ClusterSecretStore...")
	return common.RunCommand("kubectl", "apply", "-f", gitlabClusterSecretStoreURL)
}

func verifyESOSecretStore() error {
	fmt.Println("🔍 Verifying ESO SecretStore setup...")

	fmt.Println("\n🔑 GitLab secret status:")
	if err := common.RunCommand("kubectl", "get", "secret", "gitlab-eso-secret", "-n", esoNamespace); err != nil {
		return err
	}

	fmt.Println("\n📋 ClusterSecretStore status:")
	if err := common.RunCommand("kubectl", "get", "clustersecretstore"); err != nil {
		fmt.Printf("Warning: failed to get ClusterSecretStore: %v\n", err)
	}

	return nil
}