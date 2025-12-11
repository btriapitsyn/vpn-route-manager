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

	// Method 2: Check for GlobalProtect-specific VPN interface
	if d.hasGlobalProtectInterface() {
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

// hasGlobalProtectInterface checks for GlobalProtect VPN interface
func (d *VPNDetector) hasGlobalProtectInterface() bool {
	// GlobalProtect specific check - look for routes to 10.x.x.x networks through utun
	cmd := exec.Command("netstat", "-rn")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		// Look for routes to 10.x.x.x networks (common for corporate VPNs)
		// that go through utun interfaces
		if strings.Contains(line, "10.") && strings.Contains(line, "utun") {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				dest := fields[0]
				iface := fields[3]
				// Check for 10.0.0.0/8 or similar routes through utun
				if strings.HasPrefix(dest, "10.") && strings.HasPrefix(iface, "utun") {
					return true
				}
			}
		}
		// Also check for specific GlobalProtect routes
		if strings.Contains(line, "10.101.") && strings.Contains(line, "utun") && strings.Contains(line, "UGSc") {
			return true
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