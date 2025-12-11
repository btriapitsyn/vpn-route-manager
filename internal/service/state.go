package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// State represents the service state
type State struct {
	VPNConnected    bool                   `json:"vpn_connected"`
	RoutesActive    bool                   `json:"routes_active"`
	ActiveServices  map[string]bool        `json:"active_services"`
	LastCheck       time.Time              `json:"last_check"`
	StartTime       time.Time              `json:"start_time"`
	LastGateway     string                 `json:"last_gateway"`
	Version         string                 `json:"version"`
}

// StateManager manages service state persistence
type StateManager struct {
	mu        sync.RWMutex
	state     *State
	stateFile string
	pidFile   string
}

// NewStateManager creates a new state manager
func NewStateManager(stateDir string) (*StateManager, error) {
	// Ensure state directory exists
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	sm := &StateManager{
		stateFile: filepath.Join(stateDir, "state.json"),
		pidFile:   filepath.Join(stateDir, "daemon.pid"),
		state: &State{
			ActiveServices: make(map[string]bool),
			StartTime:      time.Now(),
			Version:        "1.0.0",
		},
	}

	// Write PID file
	if err := sm.writePID(); err != nil {
		return nil, fmt.Errorf("failed to write PID file: %w", err)
	}

	return sm, nil
}

// Load loads state from file
func (sm *StateManager) Load() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	data, err := os.ReadFile(sm.stateFile)
	if err != nil {
		if os.IsNotExist(err) {
			// No state file yet, use defaults
			return nil
		}
		return fmt.Errorf("failed to read state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to parse state file: %w", err)
	}

	// Preserve start time if loading existing state
	if sm.state.StartTime.IsZero() {
		sm.state.StartTime = time.Now()
	}

	// Merge loaded state
	sm.state.VPNConnected = state.VPNConnected
	sm.state.RoutesActive = state.RoutesActive
	sm.state.LastCheck = state.LastCheck
	sm.state.LastGateway = state.LastGateway
	
	if state.ActiveServices != nil {
		sm.state.ActiveServices = state.ActiveServices
	}

	return nil
}

// Save saves state to file
func (sm *StateManager) Save() error {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	sm.state.LastCheck = time.Now()

	data, err := json.MarshalIndent(sm.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	// Write to temporary file first
	tmpFile := sm.stateFile + ".tmp"
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpFile, sm.stateFile); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("failed to update state file: %w", err)
	}

	return nil
}

// GetState returns a copy of the current state
func (sm *StateManager) GetState() State {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Deep copy
	state := *sm.state
	state.ActiveServices = make(map[string]bool)
	for k, v := range sm.state.ActiveServices {
		state.ActiveServices[k] = v
	}

	return state
}

// SetVPNConnected updates VPN connection state
func (sm *StateManager) SetVPNConnected(connected bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.state.VPNConnected = connected
}

// SetRoutesActive updates routes active state
func (sm *StateManager) SetRoutesActive(active bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.state.RoutesActive = active
}

// SetServiceActive updates service active state
func (sm *StateManager) SetServiceActive(service string, active bool) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.state.ActiveServices[service] = active
}

// SetLastGateway updates the last known gateway
func (sm *StateManager) SetLastGateway(gateway string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.state.LastGateway = gateway
}

// IsServiceActive checks if a service is active
func (sm *StateManager) IsServiceActive(service string) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state.ActiveServices[service]
}

// HasActiveRoutes checks if any routes are active
func (sm *StateManager) HasActiveRoutes() bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state.RoutesActive
}

// writePID writes the process PID to file
func (sm *StateManager) writePID() error {
	pid := os.Getpid()
	return os.WriteFile(sm.pidFile, []byte(fmt.Sprintf("%d", pid)), 0644)
}

// RemovePID removes the PID file
func (sm *StateManager) RemovePID() error {
	return os.Remove(sm.pidFile)
}

// GetPID reads the PID from file
func (sm *StateManager) GetPID() (int, error) {
	data, err := os.ReadFile(sm.pidFile)
	if err != nil {
		return 0, err
	}

	var pid int
	if _, err := fmt.Sscanf(string(data), "%d", &pid); err != nil {
		return 0, fmt.Errorf("invalid PID format: %w", err)
	}

	return pid, nil
}

// IsProcessRunning checks if the process with stored PID is running
func (sm *StateManager) IsProcessRunning() bool {
	pid, err := sm.GetPID()
	if err != nil {
		return false
	}

	// Check if process exists
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// On Unix, this doesn't actually check if process is running
	// We need to send signal 0 to check
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// UpdateLastCheck updates the last check timestamp
func (sm *StateManager) UpdateLastCheck() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.state.LastCheck = time.Now()
}

// GetLastCheck returns the last check timestamp
func (sm *StateManager) GetLastCheck() time.Time {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return sm.state.LastCheck
}

// Cleanup removes state files
func (sm *StateManager) Cleanup() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	var errors []error

	if err := os.Remove(sm.stateFile); err != nil && !os.IsNotExist(err) {
		errors = append(errors, fmt.Errorf("failed to remove state file: %w", err))
	}

	if err := os.Remove(sm.pidFile); err != nil && !os.IsNotExist(err) {
		errors = append(errors, fmt.Errorf("failed to remove PID file: %w", err))
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}

	return nil
}