package network

import (
	"os/exec"
	"strings"
)

// VPNDetector handles VPN connection detection
type VPNDetector struct{}

// NewVPNDetector creates a new VPN detector
func NewVPNDetector() *VPNDetector {
	return &VPNDetector{}
}

// IsVPNConnected checks if a VPN is currently connected
func (d *VPNDetector) IsVPNConnected() bool {
	// Method 1: Check for utun interface in default route (most reliable)
	if d.hasUTunDefaultRoute() {
		return true
	}

	// Method 2: Check for corporate VPN interface (routes to private networks via utun)
	if d.hasCorporateVPNInterface() {
		return true
	}

	return false
}

// hasUTunDefaultRoute checks if default route goes through utun interface
func (d *VPNDetector) hasUTunDefaultRoute() bool {
	// Check netstat for default routes - VPN is primary if utun appears first
	cmd := exec.Command("netstat", "-rn")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		// Look for IPv4 default routes only (exclude IPv6 fe80:: routes)
		if strings.HasPrefix(line, "default") && !strings.Contains(line, "fe80::") {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				// Check if this default route uses utun interface
				iface := fields[3]
				if strings.HasPrefix(iface, "utun") {
					// This is the first IPv4 default route and it's utun = VPN is active
					return true
				} else if iface == "en0" || strings.HasPrefix(iface, "en") {
					// This is the first IPv4 default route and it's ethernet = VPN is not primary
					return false
				}
			}
		}
	}

	return false
}

// hasCorporateVPNInterface checks for corporate VPN interfaces
// Detects VPNs like GlobalProtect, Cisco AnyConnect, FortiClient, etc.
func (d *VPNDetector) hasCorporateVPNInterface() bool {
	cmd := exec.Command("netstat", "-rn")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		// Look for routes to private networks (10.x.x.x, 172.16-31.x.x)
		// that go through utun interfaces - common pattern for corporate VPNs
		if strings.Contains(line, "utun") {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				dest := fields[0]
				iface := fields[3]
				// Check for private network routes through utun
				if strings.HasPrefix(iface, "utun") {
					if strings.HasPrefix(dest, "10.") || strings.HasPrefix(dest, "172.") {
						return true
					}
				}
			}
		}
	}

	return false
}

// hasVPNProcess checks for known VPN client processes
func (d *VPNDetector) hasVPNProcess() bool {
	vpnProcesses := []string{
		"GlobalProtect",
		"openvpn",
		"Viscosity",
		"Tunnelblick",
		"ExpressVPN",
		"NordVPN",
		"Cisco AnyConnect",
		"FortiClient",
		"PulseSecure",
	}

	for _, process := range vpnProcesses {
		cmd := exec.Command("pgrep", "-i", process)
		if err := cmd.Run(); err == nil {
			return true
		}
	}

	return false
}

// GetVPNInterface returns the active VPN interface name
func (d *VPNDetector) GetVPNInterface() string {
	cmd := exec.Command("route", "get", "default")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "interface:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				iface := parts[1]
				if strings.HasPrefix(iface, "utun") {
					return iface
				}
			}
		}
	}

	return ""
}

// GetVPNGateway returns the VPN gateway if connected
func (d *VPNDetector) GetVPNGateway() string {
	if !d.IsVPNConnected() {
		return ""
	}

	cmd := exec.Command("route", "get", "default")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "gateway:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return parts[1]
			}
		}
	}

	return ""
}