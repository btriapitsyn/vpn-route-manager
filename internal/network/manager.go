package network

import (
	"fmt"
	"time"
)

// Manager implements the NetworkManager interface
type Manager struct {
	gatewayDetector *GatewayDetector
	vpnDetector     *VPNDetector
	routeManager    *RouteManager
	logger          Logger
}

// NewManager creates a new network manager
func NewManager(logger Logger) *Manager {
	return &Manager{
		gatewayDetector: NewGatewayDetector(),
		vpnDetector:     NewVPNDetector(),
		routeManager:    NewRouteManager(logger),
		logger:          logger,
	}
}

// DetectGateway detects the local network gateway
func (m *Manager) DetectGateway() (string, error) {
	gateway, err := m.gatewayDetector.DetectGateway()
	if err != nil {
		m.logger.Error("Gateway detection failed: %v", err)
		return gateway, err
	}
	m.logger.Info("Detected gateway: %s", gateway)
	return gateway, nil
}

// IsVPNConnected checks if VPN is connected
func (m *Manager) IsVPNConnected() bool {
	connected := m.vpnDetector.IsVPNConnected()
	if connected {
		iface := m.vpnDetector.GetVPNInterface()
		gateway := m.vpnDetector.GetVPNGateway()
		m.logger.Debug("VPN connected via %s (gateway: %s)", iface, gateway)
	}
	return connected
}

// AddRoute adds a network route
func (m *Manager) AddRoute(network, gateway, service string) error {
	return m.routeManager.AddRoute(network, gateway, service)
}

// RemoveRoute removes a network route
func (m *Manager) RemoveRoute(network string) error {
	return m.routeManager.RemoveRoute(network)
}

// RemoveAllRoutes removes all active routes
func (m *Manager) RemoveAllRoutes() error {
	return m.routeManager.RemoveAllRoutes()
}

// GetActiveRoutes returns all active routes
func (m *Manager) GetActiveRoutes() []Route {
	return m.routeManager.GetActiveRoutes()
}

// AddServiceRoutes adds all routes for a service
func (m *Manager) AddServiceRoutes(serviceName string, networks []string, gateway string) error {
	var errors []string
	addedCount := 0

	for _, network := range networks {
		if err := m.AddRoute(network, gateway, serviceName); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", network, err))
		} else {
			addedCount++
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("added %d/%d routes, errors: %v", addedCount, len(networks), errors)
	}

	return nil
}

// RemoveServiceRoutes removes all routes for a service
func (m *Manager) RemoveServiceRoutes(serviceName string) error {
	routes := m.GetActiveRoutes()
	var errors []string
	removedCount := 0

	for _, route := range routes {
		if route.Service == serviceName {
			if err := m.RemoveRoute(route.Network); err != nil {
				errors = append(errors, fmt.Sprintf("%s: %v", route.Network, err))
			} else {
				removedCount++
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("removed %d routes, errors: %v", removedCount, errors)
	}

	return nil
}

// VerifyRoutes verifies all active routes are working
func (m *Manager) VerifyRoutes() map[string]bool {
	return m.routeManager.VerifyAllRoutes()
}

// GetStatus returns current network status
func (m *Manager) GetStatus() map[string]interface{} {
	status := make(map[string]interface{})
	
	// VPN status
	status["vpn_connected"] = m.IsVPNConnected()
	if m.IsVPNConnected() {
		status["vpn_interface"] = m.vpnDetector.GetVPNInterface()
		status["vpn_gateway"] = m.vpnDetector.GetVPNGateway()
	}
	
	// Gateway status
	gateway, err := m.DetectGateway()
	status["local_gateway"] = gateway
	status["gateway_detection_error"] = err
	
	// Route status
	routes := m.GetActiveRoutes()
	status["active_routes_count"] = len(routes)
	status["routes_by_service"] = m.getRoutesByService(routes)
	
	return status
}

// getRoutesByService groups routes by service
func (m *Manager) getRoutesByService(routes []Route) map[string]int {
	serviceCount := make(map[string]int)
	for _, route := range routes {
		serviceCount[route.Service]++
	}
	return serviceCount
}

// MonitorNetworkChanges monitors for network changes
func (m *Manager) MonitorNetworkChanges(callback func(bool)) {
	wasConnected := m.IsVPNConnected()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		isConnected := m.IsVPNConnected()
		if isConnected != wasConnected {
			m.logger.Info("VPN state changed: connected=%v", isConnected)
			callback(isConnected)
			wasConnected = isConnected
		}
	}
}