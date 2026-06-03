package cmd

import (
	"austinhome/internal/schedule"
	"austinhome/internal/ui"
	"fmt"

	"github.com/spf13/cobra"
)

var ScheduleInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install launchd plists for auto reboot and reinstall",
	Long:  `Use ~/Downloads/austinhome as SSOT, install a verified launch copy, and set up launchd tasks.`,
	RunE:  runScheduleInstall,
}

var ScheduleTriggerCmd = &cobra.Command{
	Use:   "trigger",
	Short: "Refresh schedule, reboot, then reinstall at login",
	Long:  `Refresh the verified launch copy and launchd schedules, then reboot. After login, LaunchAgent runs reinstall automatically.`,
	RunE:  runScheduleTrigger,
}

var ScheduleRemoveCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove launchd plists (schedule cleanup)",
	Long:  `Unload and remove launchd plist files for automatic reboot and reinstall.`,
	RunE:  runScheduleRemove,
}

func runScheduleInstall(cmd *cobra.Command, args []string) error {
	ui.Log.Info("Setting up scheduled tasks...")

	if err := schedule.Execute(); err != nil {
		return fmt.Errorf("schedule setup failed: %w", err)
	}

	ui.Log.Info("Schedule setup completed successfully!")
	return nil
}

func runScheduleTrigger(cmd *cobra.Command, args []string) error {
	ui.Log.Info("Triggering full reboot cycle...")

	if err := schedule.Trigger(); err != nil {
		return fmt.Errorf("trigger failed: %w", err)
	}

	return nil
}

func runScheduleRemove(cmd *cobra.Command, args []string) error {
	ui.Log.Info("Removing scheduled tasks...")

	if err := schedule.Unschedule(); err != nil {
		return fmt.Errorf("unschedule failed: %w", err)
	}

	ui.Log.Info("Scheduled tasks removed successfully!")
	return nil
}
