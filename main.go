package main

import (
	"austinhome/internal/logic/install"
	"austinhome/internal/logic/uninstall"
	"fmt"
	"os"
)

const appName = "austinhome"

func main() {
	if len(os.Args) < 2 {
		showUsage()
		return
	}

	command := os.Args[1]
	switch command {
	case "install":
		executeInstall()
	case "uninstall":
		executeUninstall()
	default:
		handleUnknownCommand(command)
	}
}

func executeInstall() {
	fmt.Println("🚀 Starting installation...")

	if err := install.Execute(); err != nil {
		fmt.Printf("Error during installation: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Installation completed successfully!")
}

func executeUninstall() {
	fmt.Println("🗑️ Starting uninstallation...")

	if err := uninstall.Execute(); err != nil {
		fmt.Printf("Error during uninstallation: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Uninstallation completed successfully!")
}

func handleUnknownCommand(command string) {
	fmt.Printf("Unknown command: %s\n", command)
	showUsage()
	os.Exit(1)
}

func showUsage() {
	fmt.Printf(`Usage: %s <command>

Commands:
  install    Install K3s on Mac via Multipass VM
  uninstall  Uninstall K3s and clean up all files

`, appName)
}
