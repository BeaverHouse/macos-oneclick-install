package schedule

import (
	"austinhome/internal/command"
	"austinhome/internal/ui"
	"fmt"
)

func Trigger() error {
	ui.Log.Info("Triggering full cycle: refresh schedule, reboot now, reinstall will run automatically after boot")
	ui.Log.Info("Check /tmp/austinhome-reinstall.log after reboot for results")

	if err := Update(); err != nil {
		return fmt.Errorf("schedule refresh failed: %v", err)
	}

	if err := command.RunCommand("sudo", "shutdown", "-r", "now"); err != nil {
		return fmt.Errorf("reboot failed: %v", err)
	}

	return nil
}
