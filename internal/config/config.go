package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config represents the main configuration structure
type Config struct {
	Gateway       string              `json:"gateway"`
	CheckInterval int                 `json:"check_interval"`
	LogDir        string              `json:"log_dir"`
	StateDir      string              `json:"state_dir"`
	Services      map[string]*Service `json:"services"`
	AutoStart     bool                `json:"auto_start"`
	Debug         bool                `json:"debug"`
}

// Service represents a service that can bypass VPN
type Service struct {
	Name        string   `json:"name"`
	Enabled     bool     `json:"enabled"`
	Networks    []string `json:"networks"`
	Domains     []string `json:"domains,omitempty"`
	Priority    int      `json:"priority"`
	Description string   `json:"description"`
}

// Manager handles configuration loading and saving
type Manager struct {
	configPath string
	config     *Config
}

// NewManager creates a new configuration manager
func NewManager(configPath string) *Manager {
	return &Manager{
		configPath: configPath,
		config:     GetDefaultConfig(),
	}
}

// Load reads configuration from file
func (m *Manager) Load() error {
	data, err := os.ReadFile(m.configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Use default config if file doesn't exist
			return nil
		}
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := json.Unmarshal(data, &m.config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	return m.Validate()
}

// Save writes configuration to file
func (m *Manager) Save() error {
	// Ensure directory exists
	dir := filepath.Dir(m.configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(m.config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(m.configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Get returns the current configuration
func (m *Manager) Get() *Config {
	return m.config
}

// Set updates the configuration
func (m *Manager) Set(config *Config) error {
	if err := ValidateConfig(config); err != nil {
		return err
	}
	m.config = config
	return nil
}

// Validate checks if the current configuration is valid
func (m *Manager) Validate() error {
	return ValidateConfig(m.config)
}

// LoadServices loads service configurations from a directory
func (m *Manager) LoadServices(servicesDir string) error {
	entries, err := os.ReadDir(servicesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // No services directory is OK
		}
		return fmt.Errorf("failed to read services directory: %w", err)
	}

	if m.config.Services == nil {
		m.config.Services = make(map[string]*Service)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(servicesDir, entry.Name())
		service, err := LoadServiceFile(path)
		if err != nil {
			// Log error but continue loading other services
			fmt.Fprintf(os.Stderr, "Warning: failed to load service %s: %v\n", entry.Name(), err)
			continue
		}

		// Use filename without extension as key
		key := entry.Name()[:len(entry.Name())-5]
		m.config.Services[key] = service
	}

	return nil
}

// LoadServiceFile loads a single service configuration file
func LoadServiceFile(path string) (*Service, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read service file: %w", err)
	}

	// Support both direct service format and wrapped format
	var wrapper map[string]*Service
	if err := json.Unmarshal(data, &wrapper); err != nil {
		// Try direct service format
		var service Service
		if err := json.Unmarshal(data, &service); err != nil {
			return nil, fmt.Errorf("failed to parse service file: %w", err)
		}
		return &service, nil
	}

	// Extract first service from wrapper
	for _, service := range wrapper {
		return service, nil
	}

	return nil, fmt.Errorf("no service found in file")
}

// GetEnabledServices returns only enabled services
func (m *Manager) GetEnabledServices() map[string]*Service {
	enabled := make(map[string]*Service)
	for name, service := range m.config.Services {
		if service.Enabled {
			enabled[name] = service
		}
	}
	return enabled
}

// EnableService enables a service by name
func (m *Manager) EnableService(name string) error {
	service, exists := m.config.Services[name]
	if !exists {
		return fmt.Errorf("service '%s' not found", name)
	}
	service.Enabled = true
	
	// Also update the service file
	if err := m.saveServiceFile(name, service); err != nil {
		return fmt.Errorf("failed to update service file: %w", err)
	}
	
	return nil
}

// DisableService disables a service by name
func (m *Manager) DisableService(name string) error {
	service, exists := m.config.Services[name]
	if !exists {
		return fmt.Errorf("service '%s' not found", name)
	}
	service.Enabled = false
	
	// Also update the service file
	if err := m.saveServiceFile(name, service); err != nil {
		return fmt.Errorf("failed to update service file: %w", err)
	}
	
	return nil
}

// saveServiceFile saves a service configuration to its individual file
func (m *Manager) saveServiceFile(name string, service *Service) error {
	homeDir, _ := os.UserHomeDir()
	servicesDir := filepath.Join(homeDir, ".vpn-route-manager", "config", "services")
	filePath := filepath.Join(servicesDir, name+".json")
	
	// Create the wrapped format that matches the original files
	wrapper := map[string]*Service{
		name: service,
	}
	
	data, err := json.MarshalIndent(wrapper, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal service: %w", err)
	}
	
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}
	
	return nil
}