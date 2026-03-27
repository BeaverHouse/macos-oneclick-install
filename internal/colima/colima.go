package colima

import (
	"austinhome/internal/command"
	"austinhome/internal/ui"

	"github.com/BeaverHouse/go-common/logger"
)

const InstanceName = "k3s-homeserver"

func Stop() {
	ui.Log.Info("Stopping Colima instance...", logger.F("name", InstanceName))
	if err := command.RunCommand("colima", "stop", InstanceName); err != nil {
		ui.Log.Warn("Failed to stop Colima", logger.F("error", err))
	}
}

func Delete() {
	ui.Log.Info("Deleting Colima instance...", logger.F("name", InstanceName))
	if err := command.RunCommand("colima", "delete", InstanceName, "--force"); err != nil {
		ui.Log.Warn("Failed to delete Colima", logger.F("error", err))
	}
}

func IsRunning() bool {
	return command.RunCommand("colima", "status", InstanceName) == nil
}
