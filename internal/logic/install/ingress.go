package install

import (
	"austinhome/internal/logic/common"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	ingressNginxVersion = "4.13.3"
	ingressRepoName     = "ingress-nginx"
	ingressRepoURL      = "https://kubernetes.github.io/ingress-nginx"
	ingressNamespace    = "ingress-nginx"
	loadBalancerIP      = "192.168.0.180"
)

func InstallIngressNginx() error {
	fmt.Println("🌐 Installing Ingress Nginx...")

	if err := addIngressRepo(); err != nil {
		return err
	}

	if err := updateHelmRepo(); err != nil {
		return err
	}

	if err := installIngressChart(); err != nil {
		return err
	}

	fmt.Println("✅ Successfully installed ingress-nginx")
	return nil
}

func addIngressRepo() error {
	fmt.Println("📦 Adding ingress-nginx Helm repository...")
	return common.RunCommand("helm", "repo", "add", ingressRepoName, ingressRepoURL)
}

func updateHelmRepo() error {
	fmt.Println("🔄 Updating Helm repositories...")
	return common.RunCommand("helm", "repo", "update")
}

func installIngressChart() error {
	fmt.Println("🚀 Installing ingress-nginx chart...")
	return common.RunCommand("helm", "upgrade", "--install", "ingress-nginx",
		"ingress-nginx/ingress-nginx",
		"--namespace", ingressNamespace,
		"--version", ingressNginxVersion,
		"--set", "controller.kind=DaemonSet",
		"--set", fmt.Sprintf("controller.service.loadBalancerIP=%s", loadBalancerIP),
		"--set", "controller.progressDeadlineSeconds=null",
		"--create-namespace")
}

func verifyIngressNginxInstallation() error {
	fmt.Println("🔍 Verifying Ingress Nginx installation...")

	fmt.Println("\n📋 Ingress Nginx pods status:")
	if err := common.RunCommand("kubectl", "get", "pods", "-n", ingressNamespace); err != nil {
		return err
	}

	fmt.Println("\n🌐 Ingress Nginx service status:")
	if err := common.RunCommand("kubectl", "get", "service", "-n", ingressNamespace); err != nil {
		fmt.Printf("Warning: failed to get ingress service: %v\n", err)
	}

	fmt.Println("\n⚙️ Ingress classes:")
	if err := common.RunCommand("kubectl", "get", "ingressclass"); err != nil {
		fmt.Printf("Warning: failed to get ingress classes: %v\n", err)
	}

	return nil
}

func getIngressIP() (string, error) {
	fmt.Println("🔍 Discovering Ingress IP address...")

	// Wait for LoadBalancer to get an external IP
	maxWaitTime := 5 * time.Minute
	checkInterval := 10 * time.Second
	startTime := time.Now()

	for time.Since(startTime) < maxWaitTime {
		output, err := common.RunCommandOutput("kubectl", "get", "service", "ingress-nginx-controller", "-n", ingressNamespace, "-o", "jsonpath={.status.loadBalancer.ingress[0].ip}")
		if err != nil {
			return "", fmt.Errorf("failed to get ingress service info: %v", err)
		}

		ip := strings.TrimSpace(output)
		if ip != "" && ip != "<nil>" {
			fmt.Printf("✅ Found Ingress IP: %s\n", ip)
			return ip, nil
		}

		fmt.Printf("⏳ Waiting for LoadBalancer IP... (%v elapsed)\n", time.Since(startTime).Truncate(time.Second))
		time.Sleep(checkInterval)
	}

	return "", fmt.Errorf("timeout: LoadBalancer IP not assigned after %v", maxWaitTime)
}

func testIngressConnectivity(ip string) error {
	fmt.Printf("🧪 Testing Ingress connectivity at %s...\n", ip)

	// Test HTTP connection to the ingress
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	testURL := fmt.Sprintf("http://%s", ip)
	fmt.Printf("📡 Making HTTP request to %s\n", testURL)

	resp, err := client.Get(testURL)
	if err != nil {
		return fmt.Errorf("failed to connect to ingress at %s: %v", ip, err)
	}
	defer resp.Body.Close()

	fmt.Printf("✅ HTTP Response: %s (Status: %d)\n", resp.Status, resp.StatusCode)

	// Check if it's nginx (even if 404, it should be nginx)
	serverHeader := resp.Header.Get("Server")
	if strings.Contains(strings.ToLower(serverHeader), "nginx") {
		fmt.Println("✅ Nginx is responding correctly!")
		return nil
	}

	// Even without nginx in header, 404 from ingress controller is expected
	if resp.StatusCode == 404 {
		fmt.Println("✅ Got 404 Not Found - Nginx ingress controller is working!")
		return nil
	}

	return fmt.Errorf("unexpected response from ingress - expected nginx or 404, got: %s", resp.Status)
}

func testIngressFromHost(ip string) error {
	fmt.Println("🖥️ Testing Ingress connectivity from host machine...")

	// Test from host using curl (which should work from macOS)
	curlCmd := fmt.Sprintf("curl -s -o /dev/null -w '%%{http_code}' --connect-timeout 10 http://%s", ip)
	output, err := common.RunCommandOutput("bash", "-c", curlCmd)
	if err != nil {
		return fmt.Errorf("failed to test connectivity from host: %v", err)
	}

	statusCode := strings.TrimSpace(output)
	fmt.Printf("📡 Host curl response code: %s\n", statusCode)

	// Accept both 200 (if there's a default backend) and 404 (normal for ingress without default)
	if statusCode == "200" || statusCode == "404" {
		fmt.Println("✅ Host can reach Ingress successfully!")
		return nil
	}

	return fmt.Errorf("unexpected HTTP status from host: %s", statusCode)
}

func performNetworkAnalysis(ip string) error {
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

	// Show ingress service details
	fmt.Println("🌐 Ingress service details:")
	common.RunCommand("kubectl", "get", "service", "ingress-nginx-controller", "-n", ingressNamespace, "-o", "wide")

	// Show ingress controller logs
	fmt.Println("📋 Recent ingress controller logs:")
	common.RunCommand("kubectl", "logs", "-n", ingressNamespace, "-l", "app.kubernetes.io/name=ingress-nginx", "--tail=20")

	return fmt.Errorf("network analysis completed - please check the output above for connectivity issues")
}

func VerifyIngressConnectivity() error {
	fmt.Println("🌐 Verifying Ingress connectivity...")

	// Wait for ingress controller pods to be ready
	maxWaitTime := 3 * time.Minute
	err := common.WaitForPodsReady(ingressNamespace, "app.kubernetes.io/name=ingress-nginx", maxWaitTime)
	if err != nil {
		fmt.Printf("⚠️ Warning: %v, proceeding anyway\n", err)
	}

	// Get the ingress IP
	ip, err := getIngressIP()
	if err != nil {
		return err
	}

	// Test connectivity from cluster perspective
	if err := testIngressConnectivity(ip); err != nil {
		fmt.Printf("❌ Cluster connectivity test failed: %v\n", err)
		return performNetworkAnalysis(ip)
	}

	// Test connectivity from host
	if err := testIngressFromHost(ip); err != nil {
		fmt.Printf("❌ Host connectivity test failed: %v\n", err)
		return performNetworkAnalysis(ip)
	}

	fmt.Println("✅ All Ingress connectivity tests passed!")
	return nil
}