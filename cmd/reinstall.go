package cmd

import (
	"austinhome/internal/constants"
	"austinhome/internal/reinstall"
	"austinhome/internal/ui"

	"github.com/spf13/cobra"
)

var ReinstallCmd = &cobra.Command{
	Use:   "reinstall",
	Short: "Uninstall, reinstall, and register OKE (non-interactive)",
	Long:  `Non-interactive pipeline: uninstall existing cluster → install fresh → register OKE → export kubeconfig. Requires prior 'austinhome install' for saved config.`,
	RunE:  runReinstall,
}

func runReinstall(cmd *cobra.Command, args []string) error {
	ui.Log.Info("Starting reinstall...")

	if err := reinstall.Execute(); err != nil {
		return constants.ErrOperationFailed("reinstall", err)
	}

	ui.Log.Info("Reinstall completed successfully!")
	return nil
}
