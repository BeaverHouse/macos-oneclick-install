package schedule

import (
	"austinhome/internal/logic/common"
	"fmt"
	"os"
	"path/filepath"
)

// Unschedule removes the launchd plist files for automatic reboot and reinstall.
func Unschedule() error {
	fmt.Println("📅 Removing scheduled tasks...")

	// 1. Remove reboot daemon
	fmt.Println("\n📦 Step 1: Remove monthly reboot schedule")
	rebootDst := filepath.Join("/Library/LaunchDaemons", rebootLabel+".plist")

	if err := common.RunCommand("sudo", "launchctl", "bootout", "system/"+rebootLabel); err != nil {
		fmt.Println("  (job not loaded, continuing)")
	}
	if err := common.RunCommand("sudo", "rm", "-f", rebootDst); err != nil {
		fmt.Printf("Warning: failed to remove reboot plist: %v\n", err)
	}
	fmt.Println("✅ Monthly reboot schedule removed")

	// 2. Remove reinstall agent
	fmt.Println("\n📦 Step 2: Remove boot-time reinstall agent")
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %v", err)
	}

	reinstallDst := filepath.Join(home, "Library", "LaunchAgents", reinstallLabel+".plist")

	uid := fmt.Sprintf("%d", os.Getuid())
	domain := "gui/" + uid
	if err := common.RunCommand("launchctl", "bootout", domain+"/"+reinstallLabel); err != nil {
		fmt.Println("  (job not loaded, continuing)")
	}
	if err := os.Remove(reinstallDst); err != nil {
		if !os.IsNotExist(err) {
			fmt.Printf("Warning: failed to remove reinstall plist: %v\n", err)
		}
	}
	fmt.Println("✅ Boot-time reinstall agent removed")

	fmt.Println("\n🎉 All scheduled tasks removed!")
	return nil
}
