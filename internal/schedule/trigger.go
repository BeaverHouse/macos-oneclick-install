package schedule

import (
	"austinhome/internal/command"
	"austinhome/internal/constants"
	"austinhome/internal/ui"
)

func Trigger() error {
	ui.Log.Info("Triggering full cycle: refresh schedule, reboot now, reinstall will run automatically after boot")
	ui.Log.Info("Check /tmp/austinhome-reinstall.log after reboot for results")

	if err := Update(); err != nil {
		return constants.ErrOperationFailed("schedule refresh", err)
	}

	if err := command.RunCommand("sudo", "shutdown", "-r", "now"); err != nil {
		return constants.ErrOperationFailed("reboot", err)
	}

	return nil
}
