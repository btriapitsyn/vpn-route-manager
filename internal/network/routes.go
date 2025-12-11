package network

import (
	"fmt"
	"net"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// Route represents a network route
type Route struct {
	Network   string
	Gateway   string
	Interface string
	AddedAt   time.Time
	Service   string
}

// RouteManager handles route manipulation
type RouteManager struct {
	mu           sync.Mutex
	activeRoutes map[string]*Route
	logger       Logger
}

// Logger interface for logging
type Logger interface {
	Info(string, ...interface{})
	Error(string, ...interface{})
	Debug(string, ...interface{})
}

// NewRouteManager creates a new route manager
func NewRouteManager(logger Logger) *RouteManager {
	return &RouteManager{
		activeRoutes: make(map[string]*Route),
		logger:       logger,
	}
}

// AddRoute adds a network route
func (m *RouteManager) AddRoute(network, gateway, service string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate network format
	_, _, err := net.ParseCIDR(network)
	if err != nil {
		return fmt.Errorf("invalid network format %s: %w", network, err)
	}

	// Check if route already exists
	if existing, exists := m.activeRoutes[network]; exists {
		if existing.Gateway == gateway {
			m.logger.Debug("Route for %s already exists with gateway %s", network, gateway)
			return nil
		}
		// Remove existing route first
		if err := m.removeRouteCommand(network); err != nil {
			m.logger.Error("Failed to remove existing route for %s: %v", network, err)
		}
	}

	// Add the route
	cmd := exec.Command("sudo", "route", "add", "-net", network, gateway)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to add route: %s: %w", string(output), err)
	}

	// Store route information
	m.activeRoutes[network] = &Route{
		Network: network,
		Gateway: gateway,
		AddedAt: time.Now(),
		Service: service,
	}

	m.logger.Info("Added route: %s -> %s (service: %s)", network, gateway, service)
	return nil
}

// RemoveRoute removes a network route
func (m *RouteManager) RemoveRoute(network string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	route, exists := m.activeRoutes[network]
	if !exists {
		m.logger.Debug("Route for %s not in active routes", network)
		return nil
	}

	if err := m.removeRouteCommand(network); err != nil {
		return err
	}

	delete(m.activeRoutes, network)
	m.logger.Info("Removed route: %s (service: %s)", network, route.Service)
	return nil
}

// removeRouteCommand executes the route delete command
func (m *RouteManager) removeRouteCommand(network string) error {
	cmd := exec.Command("sudo", "route", "delete", "-net", network)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If route doesn't exist, that's OK
		if strings.Contains(string(output), "not in table") {
			return nil
		}
		return fmt.Errorf("failed to remove route: %s: %w", string(output), err)
	}
	return nil
}

// RemoveAllRoutes removes all active routes
func (m *RouteManager) RemoveAllRoutes() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errors []string
	for network := range m.activeRoutes {
		if err := m.removeRouteCommand(network); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", network, err))
		} else {
			delete(m.activeRoutes, network)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to remove some routes: %s", strings.Join(errors, "; "))
	}

	m.logger.Info("Removed all %d active routes", len(m.activeRoutes))
	return nil
}

// GetActiveRoutes returns a copy of active routes
func (m *RouteManager) GetActiveRoutes() []Route {
	m.mu.Lock()
	defer m.mu.Unlock()

	routes := make([]Route, 0, len(m.activeRoutes))
	for _, route := range m.activeRoutes {
		routes = append(routes, *route)
	}
	return routes
}

// VerifyRoute checks if a route is actually active
func (m *RouteManager) VerifyRoute(network string) bool {
	// Check if the route exists in our active routes
	m.mu.Lock()
	route, exists := m.activeRoutes[network]
	m.mu.Unlock()

	if !exists {
		return false
	}

	// Check the actual routing table using netstat
	// This is more reliable than "route get" for broad network ranges
	cmd := exec.Command("netstat", "-rn")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Parse CIDR to get network address
	ip, ipnet, err := net.ParseCIDR(network)
	if err != nil {
		return false
	}

	// Format network for netstat matching
	// netstat on macOS shows networks without trailing zeros:
	// 172.217.0.0/16 -> "172.217/16"
	// 74.125.0.0/16 -> "74.125/16"
	// 91.108.4.0/22 -> "91.108.4/22"
	// 185.76.151.0/24 -> "185.76.151/24"
	
	ones, _ := ipnet.Mask.Size()
	ipBytes := ip.To4()
	
	// Build the netstat format by removing trailing zero octets
	var netstatFormat string
	
	// Special handling for /16 networks
	if ones == 16 && ipBytes[3] == 0 && ipBytes[2] == 0 {
		// All /16 networks with .0.0 are shown without /16 suffix
		// e.g., 172.217.0.0/16 -> "172.217"
		netstatFormat = fmt.Sprintf("%d.%d", ipBytes[0], ipBytes[1])
	} else if ipBytes[3] == 0 && ipBytes[2] == 0 && ipBytes[1] == 0 {
		// x.0.0.0/n -> x/n
		netstatFormat = fmt.Sprintf("%d/%d", ipBytes[0], ones)
	} else if ipBytes[3] == 0 && ipBytes[2] == 0 {
		// x.y.0.0/n -> x.y/n (for non-/16 networks)
		netstatFormat = fmt.Sprintf("%d.%d/%d", ipBytes[0], ipBytes[1], ones)
	} else if ipBytes[3] == 0 {
		// x.y.z.0/n -> x.y.z/n
		netstatFormat = fmt.Sprintf("%d.%d.%d/%d", ipBytes[0], ipBytes[1], ipBytes[2], ones)
	} else {
		// x.y.z.w/n -> x.y.z.w/n
		netstatFormat = fmt.Sprintf("%d.%d.%d.%d/%d", ipBytes[0], ipBytes[1], ipBytes[2], ipBytes[3], ones)
	}

	// Check if the route exists in the routing table with our gateway
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		// Skip empty lines and headers
		if line == "" || strings.Contains(line, "Destination") || strings.Contains(line, "Internet") {
			continue
		}
		
		// Split the line to check destination and gateway
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			// Check if this is our route by comparing destination and gateway
			if fields[0] == netstatFormat && fields[1] == route.Gateway {
				return true
			}
		}
	}

	// Log for debugging if we have debug enabled
	if m.logger != nil {
		m.logger.Debug("Route verification failed: network=%s, netstatFormat=%s, gateway=%s", 
			network, netstatFormat, route.Gateway)
	}

	return false
}

// VerifyAllRoutes checks all active routes
func (m *RouteManager) VerifyAllRoutes() map[string]bool {
	m.mu.Lock()
	networks := make([]string, 0, len(m.activeRoutes))
	for network := range m.activeRoutes {
		networks = append(networks, network)
	}
	m.mu.Unlock()

	results := make(map[string]bool)
	for _, network := range networks {
		results[network] = m.VerifyRoute(network)
	}

	return results
}

// RestoreRoutes re-adds all routes (useful after network changes)
func (m *RouteManager) RestoreRoutes(gateway string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errors []string
	for network, route := range m.activeRoutes {
		cmd := exec.Command("sudo", "route", "add", "-net", network, gateway)
		if output, err := cmd.CombinedOutput(); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %s", network, string(output)))
		} else {
			route.Gateway = gateway
			m.logger.Info("Restored route: %s -> %s", network, gateway)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to restore some routes: %s", strings.Join(errors, "; "))
	}

	return nil
}

// GetRouteCount returns the number of active routes
func (m *RouteManager) GetRouteCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.activeRoutes)
}

// GetServiceRouteCount returns the number of routes for a specific service
func (m *RouteManager) GetServiceRouteCount(service string) int {
	m.mu.Lock()
	defer m.mu.Unlock()

	count := 0
	for _, route := range m.activeRoutes {
		if route.Service == service {
			count++
		}
	}
	return count
}