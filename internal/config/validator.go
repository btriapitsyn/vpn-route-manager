package config

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
)

// ValidateConfig validates the configuration
func ValidateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("configuration is nil")
	}

	// Validate gateway
	if cfg.Gateway != "auto" && cfg.Gateway != "" {
		if net.ParseIP(cfg.Gateway) == nil {
			return fmt.Errorf("invalid gateway IP: %s", cfg.Gateway)
		}
	}

	// Validate check interval
	if cfg.CheckInterval < 1 || cfg.CheckInterval > 300 {
		return fmt.Errorf("check_interval must be between 1 and 300 seconds")
	}

	// Validate directories
	if cfg.LogDir == "" {
		return fmt.Errorf("log_dir cannot be empty")
	}
	if cfg.StateDir == "" {
		return fmt.Errorf("state_dir cannot be empty")
	}

	// Validate services
	for name, service := range cfg.Services {
		if err := ValidateService(name, service); err != nil {
			return fmt.Errorf("service '%s': %w", name, err)
		}
	}

	return nil
}

// ValidateService validates a service configuration
func ValidateService(name string, service *Service) error {
	if service == nil {
		return fmt.Errorf("service is nil")
	}

	if service.Name == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	if len(service.Networks) == 0 {
		return fmt.Errorf("service must have at least one network")
	}

	// Validate network CIDR notation
	for _, network := range service.Networks {
		_, _, err := net.ParseCIDR(network)
		if err != nil {
			return fmt.Errorf("invalid network CIDR '%s': %w", network, err)
		}
	}

	// Validate priority
	if service.Priority < 0 || service.Priority > 1000 {
		return fmt.Errorf("priority must be between 0 and 1000")
	}

	return nil
}

// EnsureDirectories creates necessary directories
func EnsureDirectories(cfg *Config) error {
	dirs := []string{
		cfg.LogDir,
		cfg.StateDir,
		filepath.Dir(cfg.LogDir),    // Parent directory
		filepath.Dir(cfg.StateDir),  // Parent directory
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}