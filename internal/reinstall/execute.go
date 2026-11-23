package reinstall

import (
	"austinhome/internal/config"
	"austinhome/internal/constants"
	"austinhome/internal/install"
	"austinhome/internal/oke"
	"austinhome/internal/ui"
	"austinhome/internal/uninstall"

	"github.com/BeaverHouse/go-common/logger"
)

const defaultEnvLabel = "prod"

func Execute() error {
	ui.Log.Info("Starting reinstall pipeline...")

	gitlabPAT, err := config.Load("gitlab-pat")
	if err != nil {
		return constants.ErrNotConfigured("GitLab PAT in ~/.austinhome/gitlab-pat", err)
	}
	ui.Log.Info("GitLab PAT loaded from config")

	ui.Log.Info("Step 1: Uninstall existing cluster")
	if err := uninstall.Execute(); err != nil {
		ui.Log.Warn("Uninstall failed (continuing anyway)", logger.Field{Key: "error", Value: err})
	} else {
		ui.Log.Info("Uninstall completed")
	}

	ui.Log.Info("Step 2: Install K3s cluster")
	if err := install.ExecuteNonInteractive(defaultEnvLabel, gitlabPAT); err != nil {
		return constants.ErrOperationFailed("install", err)
	}
	ui.Log.Info("Install completed")

	ui.Log.Info("Step 3: Register OKE cluster with ArgoCD")
	if err := oke.Register(); err != nil {
		return constants.ErrOperationFailed("OKE registration", err)
	}
	ui.Log.Info("OKE registration completed")

	ui.Log.Info("Step 4: Export kubeconfig for MacBook")
	if err := oke.ExportKubeconfig(); err != nil {
		return constants.ErrOperationFailed("kubeconfig export", err)
	}
	ui.Log.Info("Kubeconfig exported")

	ui.Log.Info("Reinstall pipeline completed!")
	return nil
}
