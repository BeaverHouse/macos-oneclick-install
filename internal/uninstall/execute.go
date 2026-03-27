package uninstall

import (
	"austinhome/internal/colima"
	"austinhome/internal/ui"

	"github.com/BeaverHouse/go-common/logger"
)

func Execute() error {
	colima.Stop()
	colima.Delete()

	if err := UninstallHelm(); err != nil {
		ui.Log.Warn("Helm uninstall failed", logger.F("error", err))
	}

	if err := cleanupDirectories(); err != nil {
		return err
	}

	cleanupKubectlConfig()
	killRemainingProcesses()
	cleanHomebrew()

	return nil
}
