package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "austinhome",
	Short: "macOS K3s cluster automation",
	Long:  `Automate the setup of a K3s Kubernetes cluster on macOS via Colima, with enterprise-grade infrastructure components for GitOps-based deployments.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(InstallCmd)
	rootCmd.AddCommand(UninstallCmd)
	rootCmd.AddCommand(ReinstallCmd)
	rootCmd.AddCommand(ExportKubeconfigCmd)

	scheduleCmd := &cobra.Command{
		Use:   "schedule",
		Short: "Manage launchd scheduled tasks",
		Long:  `Install or remove launchd plist files for automatic reboot and reinstall.`,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}
	scheduleCmd.AddCommand(ScheduleInstallCmd)
	scheduleCmd.AddCommand(ScheduleRemoveCmd)
	scheduleCmd.AddCommand(ScheduleTriggerCmd)
	rootCmd.AddCommand(scheduleCmd)
}
