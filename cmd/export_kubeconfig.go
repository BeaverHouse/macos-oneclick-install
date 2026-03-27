package cmd

import (
	"austinhome/internal/oke"

	"github.com/spf13/cobra"
)

var ExportKubeconfigCmd = &cobra.Command{
	Use:   "export-kubeconfig",
	Short: "Export kubeconfig with LAN IP for remote access",
	RunE:  runExportKubeconfig,
}

func runExportKubeconfig(cmd *cobra.Command, args []string) error {
	return oke.ExportKubeconfig()
}
