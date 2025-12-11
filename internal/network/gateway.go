package network

import (
	"bufio"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strings"
	"time"
)

// GatewayDetector handles gateway detection
type GatewayDetector struct {
	cache        string
	cacheTime    time.Time
	cacheDuration time.Duration
}

// NewGatewayDetector creates a new gateway detector
func NewGatewayDetector() *GatewayDetector {
	return &GatewayDetector{
		cacheDuration: 5 * time.Minute,
	}
}

// DetectGateway detects the local network gateway
func (d *GatewayDetector) DetectGateway() (string, error) {
	// Check cache first
	if d.cache != "" && time.Since(d.cacheTime) < d.cacheDuration {
		return d.cache, nil
	}

	// Try multiple detection methods
	methods := []func() (string, error){
		d.detectFromNetstat,
		d.detectFromRoute,
		d.detectFromNetworksetup,
		d.detectFromIPConfig,
		d.detectCommonGateways,
	}

	for _, method := range methods {
		if gateway, err := method(); err == nil && gateway != "" {
			// Validate it's not a VPN gateway
			if !d.isVPNGateway(gateway) {
				d.cache = gateway
				d.cacheTime = time.Now()
				return gateway, nil
			}
		}
	}

	// Default fallback
	return "192.168.1.1", fmt.Errorf("could not detect gateway reliably")
}

// detectFromNetstat uses netstat to find the gateway
func (d *GatewayDetector) detectFromNetstat() (string, error) {
	cmd := exec.Command("netstat", "-rn")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(strings.NewReader(string(output)))
	for scanner.Scan() {
		line := scanner.Text()
		// Look for default route through physical interface (en0)
		if strings.HasPrefix(line, "default") && strings.Contains(line, "en0") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				gateway := fields[1]
				if net.ParseIP(gateway) != nil {
					return gateway, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no gateway found in netstat output")
}

// detectFromRoute uses route command to find gateway
func (d *GatewayDetector) detectFromRoute() (string, error) {
	// Check routes to private networks
	privateNets := []string{"192.168.0.0/16", "10.0.0.0/8", "172.16.0.0/12"}
	
	for _, network := range privateNets {
		cmd := exec.Command("route", "-n", "get", network)
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		// Parse output for gateway
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.Contains(line, "gateway:") && !strings.Contains(line, "utun") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					gateway := parts[1]
					if net.ParseIP(gateway) != nil && d.isPrivateIP(gateway) {
						return gateway, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("no gateway found via route command")
}

// detectFromNetworksetup uses networksetup to find gateway
func (d *GatewayDetector) detectFromNetworksetup() (string, error) {
	// Try WiFi first, then Ethernet
	interfaces := []string{"Wi-Fi", "Ethernet"}
	
	for _, iface := range interfaces {
		cmd := exec.Command("networksetup", "-getinfo", iface)
		output, err := cmd.Output()
		if err != nil {
			continue
		}

		// Parse output for router
		lines := strings.Split(string(output), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Router:") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					gateway := parts[1]
					if net.ParseIP(gateway) != nil {
						return gateway, nil
					}
				}
			}
		}
	}

	return "", fmt.Errorf("no gateway found via networksetup")
}

// detectFromIPConfig uses IP configuration to infer gateway
func (d *GatewayDetector) detectFromIPConfig() (string, error) {
	cmd := exec.Command("ifconfig", "en0")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	// Extract IP address
	ipRegex := regexp.MustCompile(`inet\s+(\d+\.\d+\.\d+\.\d+)`)
	matches := ipRegex.FindStringSubmatch(string(output))
	if len(matches) < 2 {
		return "", fmt.Errorf("no IP found on en0")
	}

	ip := net.ParseIP(matches[1])
	if ip == nil {
		return "", fmt.Errorf("invalid IP address")
	}

	// Infer gateway (usually .1 in the subnet)
	if ip4 := ip.To4(); ip4 != nil {
		// Try common patterns
		gateway := fmt.Sprintf("%d.%d.%d.1", ip4[0], ip4[1], ip4[2])
		if d.pingGateway(gateway) {
			return gateway, nil
		}
		
		// Try .254 as well (some routers use this)
		gateway = fmt.Sprintf("%d.%d.%d.254", ip4[0], ip4[1], ip4[2])
		if d.pingGateway(gateway) {
			return gateway, nil
		}
	}

	return "", fmt.Errorf("could not infer gateway from IP")
}

// detectCommonGateways tries common gateway IPs
func (d *GatewayDetector) detectCommonGateways() (string, error) {
	commonGateways := []string{
		"192.168.1.1",
		"192.168.0.1",
		"10.0.0.1",
		"192.168.2.1",
		"10.1.1.1",
		"172.16.0.1",
	}

	for _, gateway := range commonGateways {
		if d.pingGateway(gateway) {
			return gateway, nil
		}
	}

	return "", fmt.Errorf("no common gateways responding")
}

// isVPNGateway checks if the gateway looks like a VPN gateway
func (d *GatewayDetector) isVPNGateway(gateway string) bool {
	// Common corporate VPN gateway patterns (used by GlobalProtect, Cisco AnyConnect, etc.)
	vpnPatterns := []string{
		"10.10",   // Common corporate VPN ranges (10.10x.x.x)
		"172.29.", // Docker/VM/VPN patterns
		"172.30.",
		"172.31.",
	}

	for _, pattern := range vpnPatterns {
		if strings.HasPrefix(gateway, pattern) {
			return true
		}
	}

	return false
}

// isPrivateIP checks if an IP is in private address space
func (d *GatewayDetector) isPrivateIP(ip string) bool {
	privateRanges := []string{
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",
	}

	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return false
	}

	for _, cidr := range privateRanges {
		_, network, _ := net.ParseCIDR(cidr)
		if network.Contains(ipAddr) {
			return true
		}
	}

	return false
}

// pingGateway checks if a gateway responds to ping
func (d *GatewayDetector) pingGateway(gateway string) bool {
	cmd := exec.Command("ping", "-c", "1", "-W", "1000", gateway)
	err := cmd.Run()
	return err == nil
}