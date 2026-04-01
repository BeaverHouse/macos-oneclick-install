package schedule

import (
	"austinhome/internal/command"
	"austinhome/internal/ui"
	"fmt"
	"os"
	"strings"

	"github.com/BeaverHouse/go-common/logger"
)

const (
	pfAnchorName = "austinhome"
	pfAnchorPath = "/etc/pf.anchors/austinhome"
	pfConfPath   = "/etc/pf.conf"
	pfAnchorContent = `nat proto tcp from any to 192.168.0.180 port 443 -> 192.168.0.34
rdr pass proto tcp from any to 192.168.0.34 port 443 -> 192.168.0.180
`
)

func setupPF() error {
	ui.Log.Info("Setting up pf port forwarding (Mac Mini → MetalLB VIP)...")

	if err := writeAnchorFile(); err != nil {
		return err
	}

	if err := updatePFConf(); err != nil {
		return err
	}

	if err := enableIPForwarding(); err != nil {
		return err
	}

	if err := reloadPF(); err != nil {
		return err
	}

	if err := installIPForwardDaemon(); err != nil {
		return err
	}

	ui.Log.Info("pf port forwarding configured")
	return nil
}

func writeAnchorFile() error {
	tmpPath := "/tmp/" + pfAnchorName + ".anchor"
	if err := os.WriteFile(tmpPath, []byte(pfAnchorContent), 0644); err != nil {
		return fmt.Errorf("failed to write anchor to /tmp: %v", err)
	}
	if err := command.RunCommand("sudo", "cp", tmpPath, pfAnchorPath); err != nil {
		return fmt.Errorf("failed to copy anchor file: %v", err)
	}
	os.Remove(tmpPath)
	ui.Log.Info("pf anchor file written", logger.F("path", pfAnchorPath))
	return nil
}

func updatePFConf() error {
	data, err := os.ReadFile(pfConfPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %v", pfConfPath, err)
	}

	content := string(data)
	if strings.Contains(content, `"austinhome"`) {
		ui.Log.Info("pf.conf already contains austinhome anchors")
		return nil
	}

	lines := strings.Split(content, "\n")
	var result []string
	loadAnchorInserted := false

	for _, line := range lines {
		result = append(result, line)
		trimmed := strings.TrimSpace(line)
		if trimmed == `nat-anchor "com.apple/*"` {
			result = append(result, fmt.Sprintf(`nat-anchor "%s"`, pfAnchorName))
		} else if trimmed == `rdr-anchor "com.apple/*"` {
			result = append(result, fmt.Sprintf(`rdr-anchor "%s"`, pfAnchorName))
		} else if strings.HasPrefix(trimmed, `load anchor "com.apple"`) && !loadAnchorInserted {
			result = append(result, fmt.Sprintf(`load anchor "%s" from "%s"`, pfAnchorName, pfAnchorPath))
			loadAnchorInserted = true
		}
	}

	tmpPath := "/tmp/pf.conf.tmp"
	if err := os.WriteFile(tmpPath, []byte(strings.Join(result, "\n")), 0644); err != nil {
		return fmt.Errorf("failed to write temp pf.conf: %v", err)
	}
	if err := command.RunCommand("sudo", "cp", tmpPath, pfConfPath); err != nil {
		return fmt.Errorf("failed to update pf.conf: %v", err)
	}
	os.Remove(tmpPath)
	ui.Log.Info("pf.conf updated with austinhome anchors")
	return nil
}

func enableIPForwarding() error {
	return command.RunCommand("sudo", "sysctl", "-w", "net.inet.ip.forwarding=1")
}

func reloadPF() error {
	if err := command.RunCommand("sudo", "pfctl", "-ef", pfConfPath); err != nil {
		ui.Log.Warn("pfctl enable/reload", logger.F("error", err))
	}
	return nil
}

func installIPForwardDaemon() error {
	label := "me.haulrest.austinhome-ipforward"
	dst := "/Library/LaunchDaemons/" + label + ".plist"

	tmpPath := "/tmp/" + label + ".plist"
	if err := os.WriteFile(tmpPath, ipforwardPlist, 0644); err != nil {
		return fmt.Errorf("failed to write ipforward plist: %v", err)
	}
	if err := command.RunCommand("sudo", "cp", tmpPath, dst); err != nil {
		return fmt.Errorf("failed to copy ipforward plist: %v", err)
	}
	if err := command.RunCommand("sudo", "chown", "root:wheel", dst); err != nil {
		return fmt.Errorf("failed to set ownership: %v", err)
	}
	if err := command.RunCommand("sudo", "launchctl", "bootout", "system/"+label); err != nil {
		ui.Log.Info("  (no existing job to remove, continuing)")
	}
	if err := command.RunCommand("sudo", "launchctl", "bootstrap", "system", dst); err != nil {
		return fmt.Errorf("failed to load ipforward daemon: %v", err)
	}
	os.Remove(tmpPath)
	ui.Log.Info("IP forwarding daemon installed")
	return nil
}

func removePF() error {
	ui.Log.Info("Removing pf port forwarding...")

	label := "me.haulrest.austinhome-ipforward"
	dst := "/Library/LaunchDaemons/" + label + ".plist"

	if err := command.RunCommand("sudo", "launchctl", "bootout", "system/"+label); err != nil {
		ui.Log.Info("  (ipforward job not loaded, continuing)")
	}
	if err := command.RunCommand("sudo", "rm", "-f", dst); err != nil {
		ui.Log.Warn("Failed to remove ipforward plist", logger.F("error", err))
	}
	if err := command.RunCommand("sudo", "rm", "-f", pfAnchorPath); err != nil {
		ui.Log.Warn("Failed to remove pf anchor", logger.F("error", err))
	}

	data, err := os.ReadFile(pfConfPath)
	if err == nil {
		lines := strings.Split(string(data), "\n")
		var filtered []string
		for _, line := range lines {
			if !strings.Contains(line, `"austinhome"`) {
				filtered = append(filtered, line)
			}
		}
		tmpPath := "/tmp/pf.conf.tmp"
		if err := os.WriteFile(tmpPath, []byte(strings.Join(filtered, "\n")), 0644); err == nil {
			command.RunCommand("sudo", "cp", tmpPath, pfConfPath)
			os.Remove(tmpPath)
		}
	}

	command.RunCommand("sudo", "pfctl", "-f", pfConfPath)
	ui.Log.Info("pf port forwarding removed")
	return nil
}
