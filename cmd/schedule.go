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
	Long:  `Install binary to /usr/local/bin, set up monthly reboot daemon and boot-time reinstall agent via launchd.`,
	RunE:  runScheduleInstall,
}

var ScheduleTriggerCmd = &cobra.Command{
	Use:   "trigger",
	Short: "Run full cycle: reinstall → reboot → reinstall",
	Long:  `Verify reinstall pipeline works, then reboot. After reboot, LaunchAgent runs reinstall again automatically.`,
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
