package service

import (
	"fmt"
	"time"
	"vpn-route-manager/internal/network"
)

// Status represents the current service status
type Status struct {
	Running         bool                   `json:"running"`
	VPNConnected    bool                   `json:"vpn_connected"`
	RoutesActive    bool                   `json:"routes_active"`
	ActiveRoutes    []network.Route        `json:"active_routes"`
	EnabledServices map[string]bool        `json:"enabled_services"`
	Gateway         string                 `json:"gateway"`
	LastCheck       time.Time              `json:"last_check"`
	Uptime          time.Duration          `json:"uptime"`
}

// GetStatusSummary returns a human-readable status summary
func (s *Status) GetStatusSummary() string {
	if !s.Running {
		return "Service not running"
	}

	if !s.VPNConnected {
		return "VPN disconnected"
	}

	if !s.RoutesActive {
		return "VPN connected, routes pending"
	}

	activeCount := 0
	for _, active := range s.EnabledServices {
		if active {
			activeCount++
		}
	}

	return fmt.Sprintf("VPN connected, %d services active, %d routes", activeCount, len(s.ActiveRoutes))
}