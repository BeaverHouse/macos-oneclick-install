package schedule

import (
	"austinhome/internal/command"
	"austinhome/internal/ui"
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BeaverHouse/go-common/logger"
)

//go:embed plist/me.haulrest.austinhome-reboot.plist
var rebootPlist []byte

//go:embed plist/me.haulrest.austinhome-reinstall.plist
var reinstallPlist []byte

//go:embed plist/me.haulrest.austinhome-network.plist
var networkPlist []byte

const (
	rebootLabel    = "me.haulrest.austinhome-reboot"
	reinstallLabel = "me.haulrest.austinhome-reinstall"
)

const binaryName = "austinhome"

func Execute() error {
	ui.Log.Info("Setting up scheduled tasks...")

	if err := Update(); err != nil {
		return err
	}

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

func Update() error {
	ui.Log.Info("Step 0: Install binary")
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}
	sourcePath, launchPath := binaryPaths(home)
	if err := installLaunchBinary(sourcePath, launchPath); err != nil {
		return err
	}
	ui.Log.Info("Launch binary installed", logger.F("source", sourcePath), logger.F("path", launchPath))

	ui.Log.Info("Step 1: Boot-time reinstall agent")
	agentsDir := filepath.Join(home, "Library", "LaunchAgents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %v", err)
	}

	reinstallDst := filepath.Join(agentsDir, reinstallLabel+".plist")
	reinstallContent := strings.ReplaceAll(string(reinstallPlist), "__AUSTINHOME_BINARY__", launchPath)
	if err := os.WriteFile(reinstallDst, []byte(reinstallContent), 0644); err != nil {
		return fmt.Errorf("failed to write reinstall plist: %v", err)
	}

	uid := fmt.Sprintf("%d", os.Getuid())
	domain := "gui/" + uid
	if err := command.RunCommand("launchctl", "bootout", domain+"/"+reinstallLabel); err != nil {
		ui.Log.Info("  (no existing job to remove, continuing)")
	}
	ui.Log.Info("Boot-time reinstall agent installed for next login")
	return nil
}

func binaryPaths(home string) (sourcePath, launchPath string) {
	return filepath.Join(home, "Downloads", binaryName), filepath.Join(home, ".local", "bin", binaryName)
}

func installLaunchBinary(sourcePath, launchPath string) error {
	sourceBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to read Downloads SSOT binary %s: %v", sourcePath, err)
	}
	if err := os.MkdirAll(filepath.Dir(launchPath), 0755); err != nil {
		return fmt.Errorf("failed to create launch binary directory: %v", err)
	}

	tmpPath := launchPath + ".tmp"
	if err := os.WriteFile(tmpPath, sourceBytes, 0755); err != nil {
		return fmt.Errorf("failed to write temp launch binary: %v", err)
	}
	defer os.Remove(tmpPath)
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("failed to chmod temp launch binary: %v", err)
	}

	if err := os.Rename(tmpPath, launchPath); err != nil {
		return fmt.Errorf("failed to install launch binary atomically: %v", err)
	}

	launchBytes, err := os.ReadFile(launchPath)
	if err != nil {
		return fmt.Errorf("failed to verify installed launch binary: %v", err)
	}
	if !bytes.Equal(sourceBytes, launchBytes) {
		return fmt.Errorf("launch binary mismatch after install")
	}

	return nil
}
