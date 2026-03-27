package reinstall

import (
	"austinhome/internal/config"
	"austinhome/internal/install"
	"austinhome/internal/oke"
	"austinhome/internal/ui"
	"austinhome/internal/uninstall"
	"fmt"

	"github.com/BeaverHouse/go-common/logger"
)

const defaultEnvLabel = "prod"

func Execute() error {
	ui.Log.Info("Starting reinstall pipeline...")

	gitlabPAT, err := config.Load("gitlab-pat")
	if err != nil {
		return fmt.Errorf("failed to load GitLab PAT from ~/.austinhome/gitlab-pat: %v\n"+
			"Run 'austinhome install' first to save your PAT", err)
	}
	ui.Log.Info("GitLab PAT loaded from config")

	ui.Log.Info("Step 1: Uninstall existing cluster")
	if err := uninstall.Execute(); err != nil {
		ui.Log.Warn("Uninstall failed (continuing anyway)", logger.F("error", err))
	} else {
		ui.Log.Info("Uninstall completed")
	}

	ui.Log.Info("Step 2: Install K3s cluster")
	if err := install.ExecuteNonInteractive(defaultEnvLabel, gitlabPAT); err != nil {
		return fmt.Errorf("install failed: %v", err)
	}
	ui.Log.Info("Install completed")

	ui.Log.Info("Step 3: Register OKE cluster with ArgoCD")
	if err := oke.Register(); err != nil {
		ui.Log.Warn("OKE registration failed (home server is still functional)", logger.F("error", err))
	} else {
		ui.Log.Info("OKE registration completed")
	}

	ui.Log.Info("Step 4: Export kubeconfig for MacBook")
	if err := oke.ExportKubeconfig(); err != nil {
		ui.Log.Warn("Kubeconfig export failed", logger.F("error", err))
	} else {
		ui.Log.Info("Kubeconfig exported")
	}

	ui.Log.Info("Reinstall pipeline completed!")
	return nil
}
