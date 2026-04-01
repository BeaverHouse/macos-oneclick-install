package schedule

import (
	"austinhome/internal/command"
	"austinhome/internal/ui"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BeaverHouse/go-common/logger"
)

//go:embed plist/me.haulrest.austinhome-reboot.plist
var rebootPlist []byte

//go:embed plist/me.haulrest.austinhome-reinstall.plist
var reinstallPlist []byte

//go:embed plist/me.haulrest.austinhome-ipforward.plist
var ipforwardPlist []byte

const (
	rebootLabel    = "me.haulrest.austinhome-reboot"
	reinstallLabel = "me.haulrest.austinhome-reinstall"
)

const binaryInstallPath = "/usr/local/bin/austinhome"

func Execute() error {
	ui.Log.Info("Setting up scheduled tasks...")

	ui.Log.Info("Step 0: Install binary")
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to find current binary: %v", err)
	}
	if err := command.RunCommand("sudo", "cp", exe, binaryInstallPath); err != nil {
		return fmt.Errorf("failed to install binary to %s: %v", binaryInstallPath, err)
	}
	if err := command.RunCommand("sudo", "chmod", "755", binaryInstallPath); err != nil {
		return fmt.Errorf("failed to set binary permissions: %v", err)
	}
	ui.Log.Info("Binary installed", logger.F("path", binaryInstallPath))

	ui.Log.Info("Step 1: Monthly reboot schedule (requires sudo)")
	rebootDst := filepath.Join("/Library/LaunchDaemons", rebootLabel+".plist")

	if err := os.WriteFile("/tmp/"+rebootLabel+".plist", rebootPlist, 0644); err != nil {
		return fmt.Errorf("failed to write reboot plist to /tmp: %v", err)
	}
	if err := command.RunCommand("sudo", "cp", "/tmp/"+rebootLabel+".plist", rebootDst); err != nil {
		return fmt.Errorf("failed to copy reboot plist: %v", err)
	}
	if err := command.RunCommand("sudo", "chown", "root:wheel", rebootDst); err != nil {
		return fmt.Errorf("failed to set ownership on reboot plist: %v", err)
	}
	if err := command.RunCommand("sudo", "launchctl", "bootout", "system/"+rebootLabel); err != nil {
		ui.Log.Info("  (no existing job to remove, continuing)")
	}
	if err := command.RunCommand("sudo", "launchctl", "bootstrap", "system", rebootDst); err != nil {
		return fmt.Errorf("failed to load reboot plist: %v", err)
	}
	os.Remove("/tmp/" + rebootLabel + ".plist")
	ui.Log.Info("Monthly reboot scheduled (1st of month, 4:00 AM)")

	ui.Log.Info("Step 2: Boot-time reinstall agent")
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	agentsDir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %v", err)
	}

	reinstallDst := filepath.Join(agentsDir, reinstallLabel+".plist")
	if err := os.WriteFile(reinstallDst, reinstallPlist, 0644); err != nil {
		return fmt.Errorf("failed to write reinstall plist: %v", err)
	}

	uid := fmt.Sprintf("%d", os.Getuid())
	domain := "gui/" + uid
	if err := command.RunCommand("launchctl", "bootout", domain+"/"+reinstallLabel); err != nil {
		ui.Log.Info("  (no existing job to remove, continuing)")
	}
	if err := command.RunCommand("launchctl", "bootstrap", domain, reinstallDst); err != nil {
		return fmt.Errorf("failed to load reinstall plist: %v", err)
	}
	ui.Log.Info("Boot-time reinstall agent installed")

	ui.Log.Info("Step 3: pf port forwarding (Mac Mini → MetalLB VIP)")
	if err := setupPF(); err != nil {
		return fmt.Errorf("failed to setup pf: %v", err)
	}

	ui.Log.Info("Schedule setup complete!")
	ui.Log.Info("   Reboot: every 1st of month at 4:00 AM")
	ui.Log.Info("   On every boot: full reinstall (uninstall -> install -> OKE)")
	ui.Log.Info("   Port forwarding: 192.168.0.34:443 → 192.168.0.180:443")
	ui.Log.Info("   Log: /tmp/austinhome-reinstall.log")

	ui.Log.Warn("Prerequisite:")
	ui.Log.Warn("   - Auto-login must be enabled (System Settings -> Users & Groups)")
	ui.Log.Warn("   - Router port forwarding: 443 → 192.168.0.34")
	return nil
}
