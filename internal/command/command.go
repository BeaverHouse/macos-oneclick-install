package command

import (
	"austinhome/internal/ui"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/BeaverHouse/go-common/logger"
)

func setupCommandEnvironment(cmd *exec.Cmd) {
	env := os.Environ()

	pathUpdated := false
	homebrewPaths := "/usr/local/bin:/opt/homebrew/bin"

	for i, envVar := range env {
		if strings.HasPrefix(envVar, "PATH=") {
			currentPath := envVar[5:]
			if !strings.Contains(currentPath, "/usr/local/bin") || !strings.Contains(currentPath, "/opt/homebrew/bin") {
				newPath := homebrewPaths + ":" + currentPath
				env[i] = "PATH=" + newPath
			}
			pathUpdated = true
			break
		}
	}

	if !pathUpdated {
		env = append(env, "PATH="+homebrewPaths+":/usr/bin:/bin")
	}

	cmd.Env = env
}

func RunCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	setupCommandEnvironment(cmd)

	ui.Log.Debug("Running command", logger.F("command", name), logger.F("args", strings.Join(args, " ")))
	return cmd.Run()
}

func RunCommandOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)

	setupCommandEnvironment(cmd)

	ui.Log.Debug("Running command", logger.F("command", name), logger.F("args", strings.Join(args, " ")))

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

func RunCommandInteractive(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	setupCommandEnvironment(cmd)

	ui.Log.Debug("Running command", logger.F("command", name), logger.F("args", strings.Join(args, " ")))
	return cmd.Run()
}

func IsCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func RunCommandWithTimeout(timeout time.Duration, name string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	setupCommandEnvironment(cmd)

	ui.Log.Debug("Running command", logger.F("command", name), logger.F("args", strings.Join(args, " ")), logger.F("timeout", timeout))
	err := cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("command timed out after %v", timeout)
	}

	return err
}

func WaitForPodsReady(namespace, selector string, maxWaitTime time.Duration) error {
	selectorText := selector
	if selectorText == "" {
		selectorText = "all pods"
	}

	ui.Log.Info("Waiting for pods to be ready", logger.F("namespace", namespace), logger.F("selector", selectorText), logger.F("maxWait", maxWaitTime))

	checkInterval := 10 * time.Second
	startTime := time.Now()

	for time.Since(startTime) < maxWaitTime {
		var err error
		if selector == "" {
			err = RunCommand("kubectl", "wait", "--namespace", namespace,
				"--for=condition=ready", "pod", "--all", "--timeout=0s")
		} else {
			err = RunCommand("kubectl", "wait", "--namespace", namespace,
				"--for=condition=ready", "pod", "--selector="+selector, "--timeout=0s")
		}

		if err == nil {
			ui.Log.Info("Pods are ready!")
			return nil
		}

		ui.Log.Info("Still waiting...", logger.F("elapsed", time.Since(startTime).Truncate(time.Second)))
		time.Sleep(checkInterval)
	}

	return fmt.Errorf("timeout: pods not ready after %v", maxWaitTime)
}
