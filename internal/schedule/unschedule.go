package schedule

import (
	"austinhome/internal/command"
	"austinhome/internal/ui"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BeaverHouse/go-common/logger"
)

func Unschedule() error {
	ui.Log.Info("Removing scheduled tasks...")

	ui.Log.Info("Step 1: Remove monthly reboot schedule")
	rebootDst := filepath.Join("/Library/LaunchDaemons", rebootLabel+".plist")

	if err := command.RunCommand("sudo", "launchctl", "bootout", "system/"+rebootLabel); err != nil {
		ui.Log.Info("  (job not loaded, continuing)")
	}
	if err := command.RunCommand("sudo", "rm", "-f", rebootDst); err != nil {
		ui.Log.Warn("Failed to remove reboot plist", logger.F("error", err))
	}
	ui.Log.Info("Monthly reboot schedule removed")

	ui.Log.Info("Step 2: Remove boot-time reinstall agent")
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	reinstallDst := filepath.Join(home, "Library", "LaunchAgents", reinstallLabel+".plist")

	uid := fmt.Sprintf("%d", os.Getuid())
	domain := "gui/" + uid
	if err := command.RunCommand("launchctl", "bootout", domain+"/"+reinstallLabel); err != nil {
		ui.Log.Info("  (job not loaded, continuing)")
	}
	if err := os.Remove(reinstallDst); err != nil {
		if !os.IsNotExist(err) {
			ui.Log.Warn("Failed to remove reinstall plist", logger.F("error", err))
		}
	}
	ui.Log.Info("Boot-time reinstall agent removed")

	ui.Log.Info("All scheduled tasks removed!")
	return nil
}
