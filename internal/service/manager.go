package service

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"vpn-route-manager/internal/config"
	"vpn-route-manager/internal/logger"
	"vpn-route-manager/internal/network"
)

// Manager handles the main service loop
type Manager struct {
	config         *config.Manager
	network        *network.Manager
	state          *StateManager
	logger         *logger.Logger
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	mu             sync.Mutex
	isRunning      bool
	lastVPNState   bool
	checkInterval  time.Duration
}

// NewManager creates a new service manager
func NewManager(cfg *config.Manager, net *network.Manager, log *logger.Logger) (*Manager, error) {
	stateManager, err := NewStateManager(cfg.Get().StateDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create state manager: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Manager{
		config:        cfg,
		network:       net,
		state:         stateManager,
		logger:        log,
		ctx:           ctx,
		cancel:        cancel,
		checkInterval: time.Duration(cfg.Get().CheckInterval) * time.Second,
	}, nil
}

// Start starts the service
func (m *Manager) Start() error {
	m.mu.Lock()
	if m.isRunning {
		m.mu.Unlock()
		return fmt.Errorf("service is already running")
	}
	m.isRunning = true
	m.mu.Unlock()

	m.logger.Info("Starting VPN Route Manager service")

	// Load state
	if err := m.state.Load(); err != nil {
		m.logger.Warn("Failed to load state: %v", err)
	}

	// Setup signal handling
	m.setupSignalHandling()

	// Start monitoring
	m.wg.Add(1)
	go m.monitorLoop()

	m.logger.Info("Service started successfully")
	return nil
}

// Stop stops the service
func (m *Manager) Stop() error {
	m.mu.Lock()
	if !m.isRunning {
		m.mu.Unlock()
		return fmt.Errorf("service is not running")
	}
	m.isRunning = false
	m.mu.Unlock()

	m.logger.Info("Stopping VPN Route Manager service")

	// Cancel context to stop all goroutines
	m.cancel()

	// Wait for goroutines to finish
	done := make(chan struct{})
	go func() {
		m.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		m.logger.Info("Service stopped gracefully")
	case <-time.After(30 * time.Second):
		m.logger.Warn("Service stop timeout - some operations may not have completed")
	}

	// Remove all routes
	if err := m.removeAllRoutes(); err != nil {
		m.logger.Error("Failed to remove routes during shutdown: %v", err)
	}

	// Save state
	if err := m.state.Save(); err != nil {
		m.logger.Error("Failed to save state: %v", err)
	}

	return nil
}

// monitorLoop is the main monitoring loop
func (m *Manager) monitorLoop() {
	defer m.wg.Done()

	m.logger.Info("Starting VPN monitoring loop (interval: %v)", m.checkInterval)

	// Initial check
	m.checkAndUpdateRoutes()

	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			m.logger.Info("Monitoring loop stopped")
			return
		case <-ticker.C:
			m.checkAndUpdateRoutes()
		}
	}
}

// checkAndUpdateRoutes checks VPN status and updates routes accordingly
func (m *Manager) checkAndUpdateRoutes() {
	isVPNConnected := m.network.IsVPNConnected()
	
	// Always update the last check time
	m.state.UpdateLastCheck()
	
	// Log periodic check
	if m.logger != nil {
		m.logger.Debug("Monitoring: VPN=%v, Routes=%v, Check=%v", 
			isVPNConnected, 
			m.state.HasActiveRoutes(),
			m.state.GetLastCheck().Format("15:04:05"))
	}

	// Check if state changed
	if isVPNConnected != m.lastVPNState {
		m.logger.Info("VPN state changed: connected=%v", isVPNConnected)
		
		if isVPNConnected {
			m.handleVPNConnected()
		} else {
			m.handleVPNDisconnected()
		}
		
		m.lastVPNState = isVPNConnected
		m.state.SetVPNConnected(isVPNConnected)
		
		// Save state
		if err := m.state.Save(); err != nil {
			m.logger.Error("Failed to save state: %v", err)
		}
	}

	// Verify routes periodically
	// Disabled for now - netstat format inconsistencies with /16 networks
	// if isVPNConnected && m.state.HasActiveRoutes() {
	// 	m.verifyRoutes()
	// }
}

// handleVPNConnected handles VPN connection event
func (m *Manager) handleVPNConnected() {
	m.logger.Info("VPN connected - adding bypass routes")

	// Detect gateway
	gateway, err := m.network.DetectGateway()
	if err != nil {
		m.logger.Error("Failed to detect gateway: %v", err)
		return
	}

	// Get enabled services
	services := m.config.GetEnabledServices()
	if len(services) == 0 {
		m.logger.Warn("No services enabled for bypass")
		return
	}

	// Add routes for each service
	totalRoutes := 0
	for name, service := range services {
		m.logger.Info("Adding routes for service: %s", name)
		
		if err := m.network.AddServiceRoutes(name, service.Networks, gateway); err != nil {
			m.logger.Error("Failed to add routes for %s: %v", name, err)
			continue
		}
		
		routeCount := len(service.Networks)
		totalRoutes += routeCount
		m.state.SetServiceActive(name, true)
		m.logger.Info("Added %d routes for %s", routeCount, name)
	}

	m.state.SetRoutesActive(true)
	m.logger.Info("Successfully added %d total routes", totalRoutes)
}

// handleVPNDisconnected handles VPN disconnection event
func (m *Manager) handleVPNDisconnected() {
	m.logger.Info("VPN disconnected - removing bypass routes")

	if err := m.removeAllRoutes(); err != nil {
		m.logger.Error("Failed to remove routes: %v", err)
	}
}

// removeAllRoutes removes all active routes
func (m *Manager) removeAllRoutes() error {
	activeRoutes := m.network.GetActiveRoutes()
	if len(activeRoutes) == 0 {
		m.logger.Debug("No active routes to remove")
		return nil
	}

	m.logger.Info("Removing %d active routes", len(activeRoutes))
	
	if err := m.network.RemoveAllRoutes(); err != nil {
		return fmt.Errorf("failed to remove routes: %w", err)
	}

	// Update state
	m.state.SetRoutesActive(false)
	for name := range m.config.Get().Services {
		m.state.SetServiceActive(name, false)
	}

	m.logger.Info("All routes removed successfully")
	return nil
}

// verifyRoutes verifies that active routes are working
func (m *Manager) verifyRoutes() {
	results := m.network.VerifyRoutes()
	failedCount := 0
	
	for network, ok := range results {
		if !ok {
			failedCount++
			m.logger.Warn("Route verification failed for %s", network)
		}
	}

	if failedCount > 0 {
		m.logger.Warn("%d routes failed verification - attempting to restore", failedCount)
		
		// Try to restore routes
		_, err := m.network.DetectGateway()
		if err != nil {
			m.logger.Error("Failed to detect gateway for route restoration: %v", err)
			return
		}

		// Re-add all routes
		m.handleVPNConnected()
	}
}

// setupSignalHandling sets up signal handlers
func (m *Manager) setupSignalHandling() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		m.logger.Info("Received signal: %v", sig)
		m.logger.Info("VPN Route Manager shutting down")
		m.cancel()  // Cancel the context to stop monitoring
	}()
}

// Status returns the current service status
func (m *Manager) Status() (*Status, error) {
	m.mu.Lock()
	running := m.isRunning
	m.mu.Unlock()

	// Get network status
	netStatus := m.network.GetStatus()

	// Get state
	state := m.state.GetState()

	// Get enabled services
	enabledServices := make(map[string]bool)
	for name, svc := range m.config.Get().Services {
		if svc.Enabled {
			enabledServices[name] = m.state.IsServiceActive(name)
		}
	}

	return &Status{
		Running:         running,
		VPNConnected:    netStatus["vpn_connected"].(bool),
		RoutesActive:    state.RoutesActive,
		ActiveRoutes:    m.network.GetActiveRoutes(),
		EnabledServices: enabledServices,
		Gateway:         fmt.Sprintf("%v", netStatus["local_gateway"]),
		LastCheck:       state.LastCheck,
		Uptime:          time.Since(state.StartTime),
	}, nil
}

// EnableService enables a service
func (m *Manager) EnableService(name string) error {
	if err := m.config.EnableService(name); err != nil {
		return err
	}

	// Save config
	if err := m.config.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// If VPN is connected, add routes immediately
	if m.network.IsVPNConnected() {
		service := m.config.Get().Services[name]
		gateway, err := m.network.DetectGateway()
		if err != nil {
			return fmt.Errorf("failed to detect gateway: %w", err)
		}
		
		if err := m.network.AddServiceRoutes(name, service.Networks, gateway); err != nil {
			return fmt.Errorf("failed to add routes: %w", err)
		}
		
		m.state.SetServiceActive(name, true)
		m.logger.Info("Service %s enabled and routes added", name)
	} else {
		m.logger.Info("Service %s enabled (routes will be added when VPN connects)", name)
	}

	return nil
}

// DisableService disables a service
func (m *Manager) DisableService(name string) error {
	if err := m.config.DisableService(name); err != nil {
		return err
	}

	// Save config
	if err := m.config.Save(); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	// Remove routes if active
	if m.state.IsServiceActive(name) {
		if err := m.network.RemoveServiceRoutes(name); err != nil {
			return fmt.Errorf("failed to remove routes: %w", err)
		}
		
		m.state.SetServiceActive(name, false)
		m.logger.Info("Service %s disabled and routes removed", name)
	} else {
		m.logger.Info("Service %s disabled", name)
	}

	return nil
}