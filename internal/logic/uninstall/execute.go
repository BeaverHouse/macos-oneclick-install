package uninstall

import "fmt"

func Execute() error {
	// Stop and delete Colima instance
	stopColima()
	deleteColima()

	// Uninstall Helm if needed
	if err := UninstallHelm(); err != nil {
		fmt.Printf("Warning: Helm uninstall failed: %v\n", err)
	}

	// Cleanup remaining resources
	if err := cleanupDirectories(); err != nil {
		return err
	}

	cleanupKubectlConfig()
	killRemainingProcesses()
	cleanHomebrew()

	return nil
}