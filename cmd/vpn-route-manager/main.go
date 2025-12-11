package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"vpn-route-manager/internal/config"
	"vpn-route-manager/internal/logger"
)

var (
	version = "1.0.0"
	cfgFile string
	debug   bool
)

var rootCmd = &cobra.Command{
	Use:   "vpn-route-manager",
	Short: "Automatically manage VPN bypass routes",
	Long: `VPN Route Manager automatically manages network routes to allow specific 
applications and services to bypass VPN connections while maintaining 
VPN protection for all other traffic.`,
	Version: version,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.vpn-route-manager/config/config.json)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug logging")

	// Add subcommands
	rootCmd.AddCommand(
		installCmd,
		uninstallCmd,
		startCmd,
		stopCmd,
		restartCmd,
		statusCmd,
		serviceCmd,
		routeCmd,
		configCmd,
		debugCmd,
		logsCmd,
	)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// getConfigPath returns the configuration file path
func getConfigPath() string {
	if cfgFile != "" {
		return cfgFile
	}
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".vpn-route-manager", "config", "config.json")
}

// getServicesPath returns the services directory path
func getServicesPath() string {
	homeDir, _ := os.UserHomeDir()
	return filepath.Join(homeDir, ".vpn-route-manager", "config", "services")
}

// createLogger creates a logger instance
func createLogger() (*logger.Logger, error) {
	homeDir, _ := os.UserHomeDir()
	logPath := filepath.Join(homeDir, ".vpn-route-manager", "logs", "vpn-route-manager.log")
	
	return logger.New(logger.Config{
		LogPath:    logPath,
		MaxSizeMB:  10,
		MaxBackups: 5,
		Debug:      debug,
	})
}

// loadConfig loads the configuration
func loadConfig() (*config.Manager, error) {
	cfgManager := config.NewManager(getConfigPath())
	
	// Load main config
	if err := cfgManager.Load(); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Load service configs
	if err := cfgManager.LoadServices(getServicesPath()); err != nil {
		return nil, fmt.Errorf("failed to load services: %w", err)
	}

	// If no services loaded, use defaults
	if len(cfgManager.Get().Services) == 0 {
		for name, svc := range config.GetDefaultServiceConfigs() {
			cfgManager.Get().Services[name] = svc
		}
	}

	return cfgManager, nil
}