package schedule

import (
	"austinhome/internal/command"
	"austinhome/internal/ui"
	"fmt"
)

func Trigger() error {
	ui.Log.Info("Triggering full cycle: reboot now, reinstall will run automatically after boot")
	ui.Log.Info("Check /tmp/austinhome-reinstall.log after reboot for results")

	if err := command.RunCommand("sudo", "shutdown", "-r", "now"); err != nil {
		return fmt.Errorf("reboot failed: %v", err)
	}

	return nil
}
