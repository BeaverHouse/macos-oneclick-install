package schedule

import (
	"austinhome/internal/logic/common"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed plist/me.haulrest.austinhome-reboot.plist
var rebootPlist []byte

//go:embed plist/me.haulrest.austinhome-reinstall.plist
var reinstallPlist []byte

const (
	rebootLabel    = "me.haulrest.austinhome-reboot"
	reinstallLabel = "me.haulrest.austinhome-reinstall"
)

const binaryInstallPath = "/usr/local/bin/austinhome"

// Execute installs the binary and launchd plist files for automatic reboot and reinstall.
func Execute() error {
	fmt.Println("📅 Setting up scheduled tasks...")

	// 0. Install binary to /usr/local/bin/
	fmt.Println("\n📦 Step 0: Install binary")
	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to find current binary: %v", err)
	}
	if err := common.RunCommand("sudo", "cp", exe, binaryInstallPath); err != nil {
		return fmt.Errorf("failed to install binary to %s: %v", binaryInstallPath, err)
	}
	if err := common.RunCommand("sudo", "chmod", "+x", binaryInstallPath); err != nil {
		return fmt.Errorf("failed to set binary permissions: %v", err)
	}
	fmt.Printf("✅ Binary installed to %s\n", binaryInstallPath)

	// 1. Install reboot plist (requires sudo → /Library/LaunchDaemons/)
	fmt.Println("\n📦 Step 1: Monthly reboot schedule (requires sudo)")
	rebootDst := filepath.Join("/Library/LaunchDaemons", rebootLabel+".plist")

	if err := os.WriteFile("/tmp/"+rebootLabel+".plist", rebootPlist, 0644); err != nil {
		return fmt.Errorf("failed to write reboot plist to /tmp: %v", err)
	}
	if err := common.RunCommand("sudo", "cp", "/tmp/"+rebootLabel+".plist", rebootDst); err != nil {
		return fmt.Errorf("failed to copy reboot plist: %v", err)
	}
	if err := common.RunCommand("sudo", "chown", "root:wheel", rebootDst); err != nil {
		return fmt.Errorf("failed to set ownership on reboot plist: %v", err)
	}
	// Unload existing job (ignore error if not loaded)
	if err := common.RunCommand("sudo", "launchctl", "bootout", "system/"+rebootLabel); err != nil {
		fmt.Println("  (no existing job to remove, continuing)")
	}
	if err := common.RunCommand("sudo", "launchctl", "bootstrap", "system", rebootDst); err != nil {
		return fmt.Errorf("failed to load reboot plist: %v", err)
	}
	os.Remove("/tmp/" + rebootLabel + ".plist")
	fmt.Println("✅ Monthly reboot scheduled (1st of month, 4:00 AM)")

	// 2. Install reinstall plist (user agent → ~/Library/LaunchAgents/)
	fmt.Println("\n📦 Step 2: Boot-time reinstall agent")
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
	if err := common.RunCommand("launchctl", "bootout", domain+"/"+reinstallLabel); err != nil {
		fmt.Println("  (no existing job to remove, continuing)")
	}
	if err := common.RunCommand("launchctl", "bootstrap", domain, reinstallDst); err != nil {
		return fmt.Errorf("failed to load reinstall plist: %v", err)
	}
	fmt.Println("✅ Boot-time reinstall agent installed")

	fmt.Println("\n🎉 Schedule setup complete!")
	fmt.Println("   Reboot: every 1st of month at 4:00 AM")
	fmt.Println("   On every boot: full reinstall (uninstall → install → OKE)")
	fmt.Println("   Log: /tmp/austinhome-reinstall.log")

	fmt.Println("\n⚠️  Prerequisite:")
	fmt.Println("   - Auto-login must be enabled (System Settings → Users & Groups)")
	return nil
}
