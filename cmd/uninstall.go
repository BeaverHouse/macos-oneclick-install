package cmd

import (
	"austinhome/internal/ui"
	"austinhome/internal/uninstall"
	"fmt"

	"github.com/spf13/cobra"
)

var UninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall K3s cluster and clean up all resources",
	Long:  `Complete cleanup: stop and delete Colima instance, remove Helm, clean up directories and kubectl config.`,
	RunE:  runUninstall,
}

func runUninstall(cmd *cobra.Command, args []string) error {
	ui.Log.Info("Starting uninstallation...")

	if err := uninstall.Execute(); err != nil {
		return fmt.Errorf("uninstallation failed: %w", err)
	}

	ui.Log.Info("Uninstallation completed successfully!")
	return nil
}
