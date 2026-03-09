package reinstall

import (
	"austinhome/internal/logic/common"
	"austinhome/internal/logic/install"
	"austinhome/internal/logic/oke"
	"austinhome/internal/logic/uninstall"
	"fmt"
)

const defaultEnvLabel = "prod"

// Execute runs the full reinstall pipeline: uninstall → install → OKE register → kubeconfig export.
func Execute() error {
	fmt.Println("🔄 Starting reinstall pipeline...")

	// 1. Load GitLab PAT from config
	gitlabPAT, err := common.ConfigLoad("gitlab-pat")
	if err != nil {
		return fmt.Errorf("failed to load GitLab PAT from ~/.austinhome/gitlab-pat: %v\n"+
			"Run 'austinhome install' first to save your PAT", err)
	}
	fmt.Println("✅ GitLab PAT loaded from config")

	// 2. Uninstall (failure is non-fatal)
	fmt.Println("\n📦 Step 1: Uninstall existing cluster")
	if err := uninstall.Execute(); err != nil {
		fmt.Printf("⚠️  Warning: Uninstall failed (continuing anyway): %v\n", err)
	} else {
		fmt.Println("✅ Uninstall completed")
	}

	// 3. Install (non-interactive, fatal on failure)
	fmt.Println("\n📦 Step 2: Install K3s cluster")
	if err := install.ExecuteNonInteractive(defaultEnvLabel, gitlabPAT); err != nil {
		return fmt.Errorf("install failed: %v", err)
	}
	fmt.Println("✅ Install completed")

	// 4. OKE registration (non-fatal)
	fmt.Println("\n📦 Step 3: Register OKE cluster with ArgoCD")
	if err := oke.Register(); err != nil {
		fmt.Printf("⚠️  Warning: OKE registration failed (home server is still functional): %v\n", err)
	} else {
		fmt.Println("✅ OKE registration completed")
	}

	// 5. Export kubeconfig (non-fatal)
	fmt.Println("\n📦 Step 4: Export kubeconfig for MacBook")
	if err := oke.ExportKubeconfig(); err != nil {
		fmt.Printf("⚠️  Warning: Kubeconfig export failed: %v\n", err)
	} else {
		fmt.Println("✅ Kubeconfig exported")
	}

	fmt.Println("\n🎉 Reinstall pipeline completed!")
	return nil
}
