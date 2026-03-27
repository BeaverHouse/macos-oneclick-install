package install

import (
	"austinhome/internal/command"
	"austinhome/internal/ui"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/BeaverHouse/go-common/logger"
)

const (
	gatewayAPIVersion       = "v1.4.0"
	nginxGatewayVersion     = "2.2.1"
	nginxGatewayNamespace   = "nginx-gateway"
	nginxGatewayOCIRegistry = "oci://ghcr.io/nginx/charts/nginx-gateway-fabric"
	nginxGatewayValuesURL   = "https://raw.githubusercontent.com/BeaverHouse/cicd/refs/heads/main/charts/oss-nginx-gateway/values-home.yaml"
	homeGatewayResourceURL  = "https://raw.githubusercontent.com/BeaverHouse/cicd/refs/heads/main/charts/oss-nginx-gateway/resources/gateway-home.yaml"
)

func gatewayAPICRDURL() string {
	return fmt.Sprintf("https://github.com/kubernetes-sigs/gateway-api/releases/download/%s/standard-install.yaml", gatewayAPIVersion)
}

func InstallNginxGateway() error {
	ui.Log.Info("Installing NGINX Gateway Fabric...")

	if err := installGatewayAPICRDs(); err != nil {
		return err
	}

	if err := installNginxGatewayFabric(); err != nil {
		return err
	}

	if err := createHomeGateway(); err != nil {
		return err
	}

	ui.Log.Info("Successfully installed NGINX Gateway Fabric")
	return nil
}

func createHomeGateway() error {
	ui.Log.Info("Creating home-gateway resource...")
	return command.RunCommand("kubectl", "apply", "-f", homeGatewayResourceURL)
}

func installGatewayAPICRDs() error {
	ui.Log.Info("Installing Gateway API CRDs", logger.F("version", gatewayAPIVersion))
	return command.RunCommand("kubectl", "apply", "-f", gatewayAPICRDURL())
}

func installNginxGatewayFabric() error {
	ui.Log.Info("Installing NGINX Gateway Fabric chart...")
	return command.RunCommand("helm", "upgrade", "--install", "nginx-gateway",
		nginxGatewayOCIRegistry,
		"--namespace", nginxGatewayNamespace,
		"--version", nginxGatewayVersion,
		"--values", nginxGatewayValuesURL,
		"--create-namespace")
}

func verifyNginxGatewayInstallation() error {
	ui.Log.Info("Verifying NGINX Gateway Fabric installation...")

	ui.Log.Info("NGINX Gateway pods status:")
	if err := command.RunCommand("kubectl", "get", "pods", "-n", nginxGatewayNamespace); err != nil {
		return err
	}

	ui.Log.Info("NGINX Gateway service status:")
	if err := command.RunCommand("kubectl", "get", "service", "-n", nginxGatewayNamespace); err != nil {
		ui.Log.Warn("Failed to get gateway service", logger.F("error", err))
	}

	ui.Log.Info("Gateway classes:")
	if err := command.RunCommand("kubectl", "get", "gatewayclass"); err != nil {
		ui.Log.Warn("Failed to get gateway classes", logger.F("error", err))
	}

	return nil
}

func getGatewayIP() (string, error) {
	ui.Log.Info("Discovering Gateway IP address...")

	maxWaitTime := 5 * time.Minute
	checkInterval := 10 * time.Second
	startTime := time.Now()

	for time.Since(startTime) < maxWaitTime {
		output, err := command.RunCommandOutput("kubectl", "get", "service", "home-gateway-nginx", "-n", nginxGatewayNamespace, "-o", "jsonpath={.status.loadBalancer.ingress[0].ip}")
		if err != nil {
			return "", fmt.Errorf("failed to get gateway service info: %v", err)
		}

		ip := strings.TrimSpace(output)
		if ip != "" && ip != "<nil>" {
			ui.Log.Info("Found Gateway IP", logger.F("ip", ip))
			return ip, nil
		}

		ui.Log.Info("Waiting for LoadBalancer IP...", logger.F("elapsed", time.Since(startTime).Truncate(time.Second)))
		time.Sleep(checkInterval)
	}

	return "", fmt.Errorf("timeout: LoadBalancer IP not assigned after %v", maxWaitTime)
}

func testGatewayConnectivity(ip string) error {
	ui.Log.Info("Testing Gateway connectivity", logger.F("ip", ip))

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	testURL := fmt.Sprintf("http://%s", ip)
	maxRetries := 6
	retryInterval := 10 * time.Second

	for attempt := 1; attempt <= maxRetries; attempt++ {
		ui.Log.Info("Making HTTP request", logger.F("url", testURL), logger.F("attempt", attempt), logger.F("maxRetries", maxRetries))

		resp, err := client.Get(testURL)
		if err != nil {
			if attempt < maxRetries {
				ui.Log.Info("Gateway not ready yet, retrying...", logger.F("retryInterval", retryInterval))
				time.Sleep(retryInterval)
				continue
			}
			return fmt.Errorf("failed to connect to gateway at %s after %d attempts: %v", ip, maxRetries, err)
		}
		defer resp.Body.Close()

		ui.Log.Info("HTTP Response", logger.F("status", resp.Status), logger.F("statusCode", resp.StatusCode))

		if resp.StatusCode == 404 {
			ui.Log.Info("Got 404 Not Found - NGINX Gateway Fabric is working!")
			return nil
		}

		serverHeader := resp.Header.Get("Server")
		if strings.Contains(strings.ToLower(serverHeader), "nginx") {
			ui.Log.Info("NGINX is responding correctly!")
			return nil
		}

		return fmt.Errorf("unexpected response from gateway - expected nginx or 404, got: %s", resp.Status)
	}

	return fmt.Errorf("failed to connect to gateway at %s after %d attempts", ip, maxRetries)
}

func testGatewayFromHost(ip string) error {
	ui.Log.Info("Testing Gateway connectivity from host machine...")

	curlCmd := fmt.Sprintf("curl -s -o /dev/null -w '%%{http_code}' --connect-timeout 10 http://%s", ip)
	output, err := command.RunCommandOutput("bash", "-c", curlCmd)
	if err != nil {
		return fmt.Errorf("failed to test connectivity from host: %v", err)
	}

	statusCode := strings.TrimSpace(output)
	ui.Log.Info("Host curl response", logger.F("statusCode", statusCode))

	if statusCode == "200" || statusCode == "404" {
		ui.Log.Info("Host can reach Gateway successfully!")
		return nil
	}

	return fmt.Errorf("unexpected HTTP status from host: %s", statusCode)
}

func performGatewayNetworkAnalysis(ip string) error {
	ui.Log.Info("Performing network analysis...")

	ui.Log.Info("Testing ping", logger.F("ip", ip))
	if err := command.RunCommand("ping", "-c", "3", ip); err != nil {
		ui.Log.Error("Ping failed", logger.F("error", err))
	} else {
		ui.Log.Info("Ping successful")
	}

	ui.Log.Info("Testing port 80 connectivity", logger.F("ip", ip))
	if err := command.RunCommand("nc", "-z", "-w", "5", ip, "80"); err != nil {
		ui.Log.Error("Port 80 not accessible", logger.F("error", err))
	} else {
		ui.Log.Info("Port 80 is accessible")
	}

	ui.Log.Info("Network routing information:")
	command.RunCommand("route", "get", ip)

	ui.Log.Info("Gateway service details:")
	command.RunCommand("kubectl", "get", "service", "home-gateway-nginx", "-n", nginxGatewayNamespace, "-o", "wide")

	ui.Log.Info("Recent gateway controller logs:")
	command.RunCommand("kubectl", "logs", "-n", nginxGatewayNamespace, "-l", "app.kubernetes.io/name=nginx-gateway-fabric", "--tail=20")

	return fmt.Errorf("network analysis completed - please check the output above for connectivity issues")
}

func VerifyGatewayConnectivity() error {
	ui.Log.Info("Verifying Gateway connectivity...")

	maxWaitTime := 3 * time.Minute
	err := command.WaitForPodsReady(nginxGatewayNamespace, "app.kubernetes.io/name=nginx-gateway-fabric", maxWaitTime)
	if err != nil {
		ui.Log.Warn("Proceeding anyway", logger.F("error", err))
	}

	ip, err := getGatewayIP()
	if err != nil {
		return err
	}

	if err := testGatewayConnectivity(ip); err != nil {
		ui.Log.Error("Cluster connectivity test failed", logger.F("error", err))
		return performGatewayNetworkAnalysis(ip)
	}

	if err := testGatewayFromHost(ip); err != nil {
		ui.Log.Error("Host connectivity test failed", logger.F("error", err))
		return performGatewayNetworkAnalysis(ip)
	}

	ui.Log.Info("All Gateway connectivity tests passed!")
	return nil
}
