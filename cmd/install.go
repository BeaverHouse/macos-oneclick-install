package cmd

import (
	"austinhome/internal/constants"
	"austinhome/internal/install"
	"austinhome/internal/ui"

	"github.com/spf13/cobra"
)

var InstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install K3s cluster on macOS via Colima",
	Long:  `Interactive full setup: K3s cluster + infrastructure components (Helm, MetalLB, Gateway, ESO, Cert-Manager, ArgoCD) + OKE registration.`,
	RunE:  runInstall,
}

func runInstall(cmd *cobra.Command, args []string) error {
	ui.Log.Info("Starting installation...")

	if err := install.Execute(); err != nil {
		return constants.ErrOperationFailed("installation", err)
	}

	ui.Log.Info("Installation completed successfully!")
	return nil
}
