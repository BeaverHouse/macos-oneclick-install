package common

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func setupCommandEnvironment(cmd *exec.Cmd) {
	// Start with current environment
	env := os.Environ()

	// Find and update PATH
	pathUpdated := false
	homebrewPaths := "/usr/local/bin:/opt/homebrew/bin"

	for i, envVar := range env {
		if strings.HasPrefix(envVar, "PATH=") {
			currentPath := envVar[5:] // Remove "PATH="
			if !strings.Contains(currentPath, "/usr/local/bin") || !strings.Contains(currentPath, "/opt/homebrew/bin") {
				newPath := homebrewPaths + ":" + currentPath
				env[i] = "PATH=" + newPath
			}
			pathUpdated = true
			break
		}
	}

	// If PATH wasn't found, add it
	if !pathUpdated {
		env = append(env, "PATH="+homebrewPaths+":/usr/bin:/bin")
	}

	cmd.Env = env
}

func RunCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set up environment with enhanced PATH
	setupCommandEnvironment(cmd)

	fmt.Printf("Running: %s %s\n", name, strings.Join(args, " "))
	return cmd.Run()
}

// RunCommandOutput runs a command and returns its output as a string
func RunCommandOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)

	// Set up environment with enhanced PATH
	setupCommandEnvironment(cmd)

	fmt.Printf("Running: %s %s\n", name, strings.Join(args, " "))

	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}

func IsCommandAvailable(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// RunCommandWithTimeout runs a command with a timeout
func RunCommandWithTimeout(timeout time.Duration, name string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set up environment with enhanced PATH
	setupCommandEnvironment(cmd)

	fmt.Printf("Running: %s %s (timeout: %v)\n", name, strings.Join(args, " "), timeout)
	err := cmd.Run()

	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Errorf("command timed out after %v", timeout)
	}

	return err
}

// RunMultipassCommand runs multipass with absolute path resolution
func RunMultipassCommand(args ...string) error {
	// Try to find multipass in common locations
	multipassPaths := []string{
		"/usr/local/bin/multipass",
		"/opt/homebrew/bin/multipass",
		"multipass", // fallback to PATH
	}

	var multipassPath string
	for _, path := range multipassPaths {
		if _, err := os.Stat(path); err == nil {
			multipassPath = path
			break
		}
	}

	if multipassPath == "" {
		multipassPath = "multipass" // Use PATH as last resort
	}

	return RunCommand(multipassPath, args...)
}

// RunMultipassCommandOutput runs multipass with absolute path resolution and returns output
func RunMultipassCommandOutput(args ...string) (string, error) {
	// Try to find multipass in common locations
	multipassPaths := []string{
		"/usr/local/bin/multipass",
		"/opt/homebrew/bin/multipass",
		"multipass", // fallback to PATH
	}

	var multipassPath string
	for _, path := range multipassPaths {
		if _, err := os.Stat(path); err == nil {
			multipassPath = path
			break
		}
	}

	if multipassPath == "" {
		multipassPath = "multipass" // Use PATH as last resort
	}

	return RunCommandOutput(multipassPath, args...)
}

// WaitForPodsReady waits for pods to be ready in a given namespace with a selector
func WaitForPodsReady(namespace, selector string, maxWaitTime time.Duration) error {
	selectorText := selector
	if selectorText == "" {
		selectorText = "all pods"
	}

	fmt.Printf("⏳ Waiting for pods in namespace %s (%s) to be ready (max %v)...\n",
		namespace, selectorText, maxWaitTime)

	checkInterval := 10 * time.Second
	startTime := time.Now()

	for time.Since(startTime) < maxWaitTime {
		var err error
		if selector == "" {
			// Use --all when no selector is provided
			err = RunCommand("kubectl", "wait", "--namespace", namespace,
				"--for=condition=ready", "pod", "--all", "--timeout=0s")
		} else {
			err = RunCommand("kubectl", "wait", "--namespace", namespace,
				"--for=condition=ready", "pod", "--selector="+selector, "--timeout=0s")
		}

		if err == nil {
			fmt.Println("✅ Pods are ready!")
			return nil
		}

		fmt.Printf("⏳ Still waiting... (%v elapsed)\n", time.Since(startTime).Truncate(time.Second))
		time.Sleep(checkInterval)
	}

	return fmt.Errorf("timeout: pods not ready after %v", maxWaitTime)
}