package install

import (
	"austinhome/internal/logic/common"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Gateway API and NGINX Gateway Fabric configuration
const (
	gatewayAPIVersion       = "v1.4.0"
	nginxGatewayVersion     = "2.2.1"
	nginxGatewayNamespace   = "nginx-gateway"
	nginxGatewayOCIRegistry = "oci://ghcr.io/nginx/charts/nginx-gateway-fabric"
	nginxGatewayValuesURL   = "https://raw.githubusercontent.com/BeaverHouse/cicd/refs/heads/main/charts/oss-nginx-gateway/values-home.yaml"
	homeGatewayResourceURL  = "https://raw.githubusercontent.com/BeaverHouse/cicd/refs/heads/main/charts/oss-nginx-gateway/resources/gateway-home.yaml"
)

// gatewayAPICRDURL returns the URL for Gateway API CRDs
func gatewayAPICRDURL() string {
	return fmt.Sprintf("https://github.com/kubernetes-sigs/gateway-api/releases/download/%s/standard-install.yaml", gatewayAPIVersion)
}

func InstallNginxGateway() error {
	fmt.Println("🌐 Installing NGINX Gateway Fabric...")

	if err := installGatewayAPICRDs(); err != nil {
		return err
	}

	if err := installNginxGatewayFabric(); err != nil {
		return err
	}

	if err := createHomeGateway(); err != nil {
		return err
	}

	fmt.Println("✅ Successfully installed NGINX Gateway Fabric")
	return nil
}

func createHomeGateway() error {
	fmt.Println("🚪 Creating home-gateway resource...")
	return common.RunCommand("kubectl", "apply", "-f", homeGatewayResourceURL)
}

func installGatewayAPICRDs() error {
	fmt.Printf("📋 Installing Gateway API CRDs (%s)...\n", gatewayAPIVersion)
	return common.RunCommand("kubectl", "apply", "-f", gatewayAPICRDURL())
}

func installNginxGatewayFabric() error {
	fmt.Println("🚀 Installing NGINX Gateway Fabric chart...")
	return common.RunCommand("helm", "upgrade", "--install", "nginx-gateway",
		nginxGatewayOCIRegistry,
		"--namespace", nginxGatewayNamespace,
		"--version", nginxGatewayVersion,
		"--values", nginxGatewayValuesURL,
		"--create-namespace")
}

func verifyNginxGatewayInstallation() error {
	fmt.Println("🔍 Verifying NGINX Gateway Fabric installation...")

	fmt.Println("\n📋 NGINX Gateway pods status:")
	if err := common.RunCommand("kubectl", "get", "pods", "-n", nginxGatewayNamespace); err != nil {
		return err
	}

	fmt.Println("\n🌐 NGINX Gateway service status:")
	if err := common.RunCommand("kubectl", "get", "service", "-n", nginxGatewayNamespace); err != nil {
		fmt.Printf("Warning: failed to get gateway service: %v\n", err)
	}

	fmt.Println("\n⚙️ Gateway classes:")
	if err := common.RunCommand("kubectl", "get", "gatewayclass"); err != nil {
		fmt.Printf("Warning: failed to get gateway classes: %v\n", err)
	}

	return nil
}

func getGatewayIP() (string, error) {
	fmt.Println("🔍 Discovering Gateway IP address...")

	// Wait for LoadBalancer to get an external IP
	maxWaitTime := 5 * time.Minute
	checkInterval := 10 * time.Second
	startTime := time.Now()

	for time.Since(startTime) < maxWaitTime {
		output, err := common.RunCommandOutput("kubectl", "get", "service", "nginx-gateway-nginx-gateway-fabric", "-n", nginxGatewayNamespace, "-o", "jsonpath={.status.loadBalancer.ingress[0].ip}")
		if err != nil {
			return "", fmt.Errorf("failed to get gateway service info: %v", err)
		}

		ip := strings.TrimSpace(output)
		if ip != "" && ip != "<nil>" {
			fmt.Printf("✅ Found Gateway IP: %s\n", ip)
			return ip, nil
		}

		fmt.Printf("⏳ Waiting for LoadBalancer IP... (%v elapsed)\n", time.Since(startTime).Truncate(time.Second))
		time.Sleep(checkInterval)
	}

	return "", fmt.Errorf("timeout: LoadBalancer IP not assigned after %v", maxWaitTime)
}

func testGatewayConnectivity(ip string) error {
	fmt.Printf("🧪 Testing Gateway connectivity at %s...\n", ip)

	// Test HTTP connection to the gateway
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	testURL := fmt.Sprintf("http://%s", ip)
	fmt.Printf("📡 Making HTTP request to %s\n", testURL)

	resp, err := client.Get(testURL)
	if err != nil {
		return fmt.Errorf("failed to connect to gateway at %s: %v", ip, err)
	}
	defer resp.Body.Close()

	fmt.Printf("✅ HTTP Response: %s (Status: %d)\n", resp.Status, resp.StatusCode)

	// NGINX Gateway Fabric returns 404 when no routes are configured
	if resp.StatusCode == 404 {
		fmt.Println("✅ Got 404 Not Found - NGINX Gateway Fabric is working!")
		return nil
	}

	// Check if it's nginx
	serverHeader := resp.Header.Get("Server")
	if strings.Contains(strings.ToLower(serverHeader), "nginx") {
		fmt.Println("✅ NGINX is responding correctly!")
		return nil
	}

	return fmt.Errorf("unexpected response from gateway - expected nginx or 404, got: %s", resp.Status)
}

func testGatewayFromHost(ip string) error {
	fmt.Println("🖥️ Testing Gateway connectivity from host machine...")

	// Test from host using curl (which should work from macOS)
	curlCmd := fmt.Sprintf("curl -s -o /dev/null -w '%%{http_code}' --connect-timeout 10 http://%s", ip)
	output, err := common.RunCommandOutput("bash", "-c", curlCmd)
	if err != nil {
		return fmt.Errorf("failed to test connectivity from host: %v", err)
	}

	statusCode := strings.TrimSpace(output)
	fmt.Printf("📡 Host curl response code: %s\n", statusCode)

	// Accept both 200 (if there's a default backend) and 404 (normal for gateway without routes)
	if statusCode == "200" || statusCode == "404" {
		fmt.Println("✅ Host can reach Gateway successfully!")
		return nil
	}

	return fmt.Errorf("unexpected HTTP status from host: %s", statusCode)
}

func performGatewayNetworkAnalysis(ip string) error {
	fmt.Println("🔍 Performing network analysis...")

	// Check if IP is reachable via ping
	fmt.Printf("📡 Testing ping to %s...\n", ip)
	if err := common.RunCommand("ping", "-c", "3", ip); err != nil {
		fmt.Printf("❌ Ping failed: %v\n", err)
	} else {
		fmt.Println("✅ Ping successful")
	}

	// Check if port 80 is open
	fmt.Printf("🔌 Testing port 80 connectivity to %s...\n", ip)
	if err := common.RunCommand("nc", "-z", "-w", "5", ip, "80"); err != nil {
		fmt.Printf("❌ Port 80 not accessible: %v\n", err)
	} else {
		fmt.Println("✅ Port 80 is accessible")
	}

	// Show network routing
	fmt.Println("🛣️ Network routing information:")
	common.RunCommand("route", "get", ip)

	// Show gateway service details
	fmt.Println("🌐 Gateway service details:")
	common.RunCommand("kubectl", "get", "service", "nginx-gateway-nginx-gateway-fabric", "-n", nginxGatewayNamespace, "-o", "wide")

	// Show gateway controller logs
	fmt.Println("📋 Recent gateway controller logs:")
	common.RunCommand("kubectl", "logs", "-n", nginxGatewayNamespace, "-l", "app.kubernetes.io/name=nginx-gateway-fabric", "--tail=20")

	return fmt.Errorf("network analysis completed - please check the output above for connectivity issues")
}

func VerifyGatewayConnectivity() error {
	fmt.Println("🌐 Verifying Gateway connectivity...")

	// Wait for gateway controller pods to be ready
	maxWaitTime := 3 * time.Minute
	err := common.WaitForPodsReady(nginxGatewayNamespace, "app.kubernetes.io/name=nginx-gateway-fabric", maxWaitTime)
	if err != nil {
		fmt.Printf("⚠️ Warning: %v, proceeding anyway\n", err)
	}

	// Get the gateway IP
	ip, err := getGatewayIP()
	if err != nil {
		return err
	}

	// Test connectivity from cluster perspective
	if err := testGatewayConnectivity(ip); err != nil {
		fmt.Printf("❌ Cluster connectivity test failed: %v\n", err)
		return performGatewayNetworkAnalysis(ip)
	}

	// Test connectivity from host
	if err := testGatewayFromHost(ip); err != nil {
		fmt.Printf("❌ Host connectivity test failed: %v\n", err)
		return performGatewayNetworkAnalysis(ip)
	}

	fmt.Println("✅ All Gateway connectivity tests passed!")
	return nil
}
