package install

import (
	"austinhome/internal/command"
	"austinhome/internal/ui"
	"fmt"
	"strings"
	"time"

	"github.com/BeaverHouse/go-common/logger"
)

const (
	argoCDVersion     = "9.4.7"
	argoCDRepoName    = "argo"
	argoCDRepoURL     = "https://argoproj.github.io/argo-helm"
	argoCDNamespace   = "argo-project"
	argoCDMaxWaitTime = 3 * time.Minute
	argoCDSyncWait    = 5 * time.Minute
)

const (
	oauthSecretURL          = "https://raw.githubusercontent.com/BeaverHouse/cicd/refs/heads/main/charts/oss-argocd/resources/oauth-secret.yaml"
	argoCDValuesURL         = "https://raw.githubusercontent.com/BeaverHouse/cicd/refs/heads/main/charts/oss-argocd/values.yaml"
	appProjectHomeServerURL = "https://raw.githubusercontent.com/BeaverHouse/cicd/refs/heads/main/charts/oss-argocd/resources/appproject-home-server.yaml"
	appProjectCloudURL      = "https://raw.githubusercontent.com/BeaverHouse/cicd/refs/heads/main/charts/oss-argocd/resources/appproject-cloud.yaml"
	appOfAppsURL            = "https://raw.githubusercontent.com/BeaverHouse/cicd/refs/heads/main/charts/oss-argocd/resources/app-of-apps.yaml"
	appOfApplicationSetsURL = "https://raw.githubusercontent.com/BeaverHouse/cicd/refs/heads/main/charts/oss-argocd/resources/app-of-applicationsets.yaml"
)

func InstallArgoCD() error {
	ui.Log.Info("Installing ArgoCD...")

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

	if err := command.WaitForPodsReady(argoCDNamespace, "app.kubernetes.io/name=argocd-dex-server", argoCDMaxWaitTime); err != nil {
		return err
	}

	if err := command.WaitForPodsReady(argoCDNamespace, "app.kubernetes.io/name=argocd-server", argoCDMaxWaitTime); err != nil {
		return err
	}

	ui.Log.Info("Restarting ArgoCD server to initialize SSO...")
	if err := command.RunCommand("kubectl", "rollout", "restart", "deployment", "argocd-server", "-n", argoCDNamespace); err != nil {
		return err
	}
	if err := command.WaitForPodsReady(argoCDNamespace, "app.kubernetes.io/name=argocd-server", argoCDMaxWaitTime); err != nil {
		return err
	}

	if err := bootstrapArgoCD(); err != nil {
		return err
	}
	if err := waitForArgoCDApplication("oss-cert-manager-home", argoCDSyncWait); err != nil {
		return err
	}

	ui.Log.Info("Successfully installed ArgoCD")
	return nil
}

func bootstrapArgoCD() error {
	ui.Log.Info("Bootstrapping ArgoCD with initial resources...")

	if err := applyAppProject(); err != nil {
		return err
	}

	if err := applyAppOfApps(); err != nil {
		return err
	}

	if err := applyAppOfApplicationSets(); err != nil {
		return err
	}

	ui.Log.Info("ArgoCD bootstrap completed")
	return nil
}

func applyAppProject() error {
	ui.Log.Info("Applying AppProjects...")

	if err := command.RunCommand("kubectl", "apply", "-f", appProjectHomeServerURL); err != nil {
		return err
	}
	if err := command.RunCommand("kubectl", "apply", "-f", appProjectCloudURL); err != nil {
		return err
	}

	return nil
}

func applyAppOfApps() error {
	ui.Log.Info("Applying App of Apps...")
	return command.RunCommand("kubectl", "apply", "-f", appOfAppsURL)
}

func applyAppOfApplicationSets() error {
	ui.Log.Info("Applying App of ApplicationSets...")
	return command.RunCommand("kubectl", "apply", "-f", appOfApplicationSetsURL)
}

func waitForArgoCDApplication(name string, maxWait time.Duration) error {
	ui.Log.Info("Waiting for ArgoCD application", logger.F("name", name))

	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		output, err := command.RunCommandOutput("kubectl", "get", "application", name,
			"-n", argoCDNamespace,
			"-o", `jsonpath={.status.sync.status}{"\t"}{.status.health.status}`)
		if err == nil {
			parts := strings.Split(strings.TrimSpace(output), "\t")
			if len(parts) == 2 && parts[0] == "Synced" && parts[1] == "Healthy" {
				ui.Log.Info("ArgoCD application is healthy", logger.F("name", name))
				return nil
			}
			ui.Log.Info("Still waiting for ArgoCD application",
				logger.F("name", name),
				logger.F("status", strings.TrimSpace(output)))
		}
		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("timeout waiting for ArgoCD application %s to become Synced/Healthy", name)
}

func createArgoCDNamespace() error {
	ui.Log.Info("Creating ArgoCD namespace...")
	err := command.RunCommand("kubectl", "create", "namespace", argoCDNamespace)
	if err != nil {
		checkErr := command.RunCommand("kubectl", "get", "namespace", argoCDNamespace)
		if checkErr != nil {
			return err
		}
		ui.Log.Info("Namespace already exists, continuing...", logger.F("namespace", argoCDNamespace))
	}
	return nil
}

func applyOAuthSecret() error {
	ui.Log.Info("Applying OAuth secret...")
	return command.RunCommand("kubectl", "apply", "-f", oauthSecretURL)
}

func addArgoCDRepo() error {
	ui.Log.Info("Adding ArgoCD Helm repository...")
	return command.RunCommand("helm", "repo", "add", argoCDRepoName, argoCDRepoURL)
}

func updateHelmRepoForArgoCD() error {
	ui.Log.Info("Updating Helm repositories...")
	return command.RunCommand("helm", "repo", "update")
}

func installArgoCDChart() error {
	ui.Log.Info("Installing ArgoCD chart...")
	return command.RunCommand("helm", "upgrade", "--install", "argocd",
		"argo/argo-cd",
		"--namespace", argoCDNamespace,
		"--create-namespace",
		"--values", argoCDValuesURL,
		"--version", argoCDVersion)
}

func verifyArgoCDInstallation() error {
	ui.Log.Info("Verifying ArgoCD installation...")

	ui.Log.Info("ArgoCD pods status:")
	if err := command.RunCommand("kubectl", "get", "pods", "-n", argoCDNamespace); err != nil {
		return err
	}

	ui.Log.Info("ArgoCD service status:")
	if err := command.RunCommand("kubectl", "get", "service", "-n", argoCDNamespace); err != nil {
		ui.Log.Warn("Failed to get ArgoCD service", logger.F("error", err))
	}

	ui.Log.Info("ArgoCD application status:")
	if err := command.RunCommand("kubectl", "get", "application", "-n", argoCDNamespace); err != nil {
		ui.Log.Info("No applications deployed yet")
	}

	return nil
}
