package cmd

import (
	"fmt"
	"os"

	"austinhome/internal/schedule"
	"austinhome/internal/ui"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "austinhome",
	Short: "macOS K3s cluster automation",
	Long:  `Automate the setup of a K3s Kubernetes cluster on macOS via Colima, with enterprise-grade infrastructure components for GitOps-based deployments.`,
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.Log.Info("No command provided; applying Austin Home update schedule from this binary.")
		ui.Log.Info("This refreshes the launch binary and boot-time reinstall agent without changing the monthly reboot daemon.")
		if err := schedule.Update(); err != nil {
			return fmt.Errorf("update schedule failed: %w", err)
		}
		ui.Log.Info("Update schedule applied. The next monthly reboot will use this austinhome binary.")
		return nil
	},
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
