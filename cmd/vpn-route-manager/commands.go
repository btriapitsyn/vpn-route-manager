package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"vpn-route-manager/internal/network"
	"vpn-route-manager/internal/service"
	"vpn-route-manager/internal/system"
)

// Start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the VPN Route Manager service",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if daemon flag is set
		daemon, _ := cmd.Flags().GetBool("daemon")
		if daemon {
			return runDaemon()
		}

		// Otherwise, start via LaunchAgent
		username := os.Getenv("USER")
		launchAgent := system.NewLaunchAgent(username)
		
		if !launchAgent.IsLoaded() {
			return fmt.Errorf("service not installed. Run 'vpn-route-manager install' first")
		}

		fmt.Println("Starting VPN Route Manager service...")
		// The service is already loaded, just needs to start
		fmt.Println("✅ Service started")
		return nil
	},
}

// Stop command
var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the VPN Route Manager service",
	RunE: func(cmd *cobra.Command, args []string) error {
		username := os.Getenv("USER")
		launchAgent := system.NewLaunchAgent(username)
		
		if !launchAgent.IsLoaded() {
			return fmt.Errorf("service not running")
		}

		fmt.Println("Stopping VPN Route Manager service...")
		if err := launchAgent.Unload(); err != nil {
			return fmt.Errorf("failed to stop service: %w", err)
		}
		
		// Reload to keep it registered but not running
		if err := launchAgent.Load(); err != nil {
			return fmt.Errorf("failed to reload service: %w", err)
		}

		fmt.Println("✅ Service stopped")
		return nil
	},
}

// Restart command
var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the VPN Route Manager service",
	RunE: func(cmd *cobra.Command, args []string) error {
		username := os.Getenv("USER")
		launchAgent := system.NewLaunchAgent(username)
		
		fmt.Println("Restarting VPN Route Manager service...")
		
		if launchAgent.IsLoaded() {
			if err := launchAgent.Unload(); err != nil {
				return fmt.Errorf("failed to stop service: %w", err)
			}
		}
		
		if err := launchAgent.Load(); err != nil {
			return fmt.Errorf("failed to start service: %w", err)
		}

		fmt.Println("✅ Service restarted")
		return nil
	},
}

// Status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show service status",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check LaunchAgent status
		username := os.Getenv("USER")
		launchAgent := system.NewLaunchAgent(username)
		
		fmt.Println("🔍 VPN Route Manager Status")
		fmt.Println("============================")
		
		// Service status
		if launchAgent.IsLoaded() {
			running, pid := launchAgent.IsRunning()
			if running {
				fmt.Printf("Service: ✅ RUNNING (PID: %d)\n", pid)
			} else {
				fmt.Println("Service: ⚠️  LOADED but NOT RUNNING")
			}
		} else {
			fmt.Println("Service: ❌ NOT INSTALLED")
			return nil
		}

		// Read the saved state
		homeDir, _ := os.UserHomeDir()
		stateFile := filepath.Join(homeDir, ".vpn-route-manager", "state", "state.json")
		
		var savedState map[string]interface{}
		if data, err := os.ReadFile(stateFile); err == nil {
			json.Unmarshal(data, &savedState)
		}

		// Get actual route count from routing table
		activeRouteCount := 0
		countCmd := exec.Command("sh", "-c", `netstat -rn | grep -E "149\.154|91\.108|185\.76\.151|95\.161\.64|172\.217|142\.250|216\.58|74\.125|64\.233|66\.249|72\.14|209\.85" | grep -v "^default" | wc -l`)
		if output, err := countCmd.Output(); err == nil {
			fmt.Sscanf(strings.TrimSpace(string(output)), "%d", &activeRouteCount)
		}

		// Get gateway
		gateway := "unknown"
		gwCmd := exec.Command("route", "get", "default")
		if output, err := gwCmd.Output(); err == nil {
			lines := strings.Split(string(output), "\n")
			for _, line := range lines {
				if strings.Contains(line, "gateway:") {
					parts := strings.Fields(line)
					if len(parts) >= 2 {
						gateway = parts[1]
					}
				}
			}
		}

		// Get VPN status from state
		vpnConnected := false
		if val, ok := savedState["vpn_connected"].(bool); ok {
			vpnConnected = val
		}

		// Get last check time
		lastCheck := "unknown"
		if val, ok := savedState["last_check"].(string); ok {
			if t, err := time.Parse(time.RFC3339, val); err == nil {
				lastCheck = t.Format("15:04:05")
			}
		}

		// Network status
		fmt.Println("\n📡 Network Status")
		fmt.Println("------------------")
		if vpnConnected {
			fmt.Println("VPN: ✅ CONNECTED")
		} else {
			fmt.Println("VPN: ❌ DISCONNECTED")
		}
		fmt.Printf("Gateway: %s\n", gateway)
		fmt.Printf("Last Check: %s\n", lastCheck)

		// Routes status
		fmt.Println("\n🛣️  Routes Status")
		fmt.Println("------------------")
		if activeRouteCount > 0 {
			fmt.Printf("Active Routes: %d\n", activeRouteCount)
		} else {
			fmt.Println("Active Routes: None")
		}

		// Services status
		fmt.Println("\n📦 Services Status")
		fmt.Println("------------------")
		
		// Load current configuration to check which services are enabled
		cfg, err := loadConfig()
		if err == nil {
			// Get all enabled services from config
			enabledServices := cfg.GetEnabledServices()
			
			// Get active services from state
			activeServicesMap := make(map[string]bool)
			if activeServices, ok := savedState["active_services"].(map[string]interface{}); ok {
				for name, active := range activeServices {
					if isActive, ok := active.(bool); ok {
						activeServicesMap[name] = isActive
					}
				}
			}
			
			// Show status for each enabled service
			// Sort service names for consistent output
			var serviceNames []string
			for name := range enabledServices {
				serviceNames = append(serviceNames, name)
			}
			sort.Strings(serviceNames)
			
			for _, name := range serviceNames {
				if activeServicesMap[name] && vpnConnected {
					fmt.Printf("%s: ✅ ACTIVE\n", name)
				} else if !vpnConnected {
					fmt.Printf("%s: ⭕ ENABLED\n", name)
				} else {
					// VPN is connected but service has no routes yet
					fmt.Printf("%s: 🔄 LOADING\n", name)
				}
			}
			
			if len(enabledServices) == 0 {
				fmt.Println("No services enabled")
			}
		} else {
			// Fallback if can't load config
			if activeServices, ok := savedState["active_services"].(map[string]interface{}); ok {
				for name, active := range activeServices {
					if isActive, ok := active.(bool); ok && isActive {
						fmt.Printf("%s: ✅ ACTIVE\n", name)
					}
				}
			}
		}

		// Show logs tail
		fmt.Println("\n📋 Recent Activity")
		fmt.Println("------------------")
		logFile := filepath.Join(homeDir, ".vpn-route-manager", "logs", "stdout.log")
		if data, err := os.ReadFile(logFile); err == nil {
			lines := strings.Split(string(data), "\n")
			start := len(lines) - 6
			if start < 0 {
				start = 0
			}
			for i := start; i < len(lines) && i < start+5; i++ {
				if lines[i] != "" {
					fmt.Println(lines[i])
				}
			}
		}

		return nil
	},
}

// Uninstall command
var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Uninstall VPN Route Manager",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("🗑️  Uninstalling VPN Route Manager...")
		
		username := os.Getenv("USER")
		
		// Stop and remove LaunchAgent
		fmt.Println("📋 Removing LaunchAgent...")
		launchAgent := system.NewLaunchAgent(username)
		if err := launchAgent.Uninstall(); err != nil {
			fmt.Printf("⚠️  Warning: %v\n", err)
		}

		// Remove sudo configuration
		fmt.Println("🔐 Removing sudo configuration...")
		sudoMgr := system.NewSudoManager(username)
		if err := sudoMgr.Remove(); err != nil {
			fmt.Printf("⚠️  Warning: %v\n", err)
		}

		// Kill any remaining processes
		fmt.Println("🛑 Stopping any remaining processes...")
		procMgr := system.NewProcessManager("vpn-route-manager")
		if err := procMgr.KillAllProcesses(false); err != nil {
			fmt.Printf("⚠️  Warning: %v\n", err)
		}

		// Ask about removing configuration
		fmt.Print("\nRemove configuration and logs? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		
		if strings.ToLower(response) == "y" {
			homeDir, _ := os.UserHomeDir()
			configDir := filepath.Join(homeDir, ".vpn-route-manager")
			
			fmt.Printf("📁 Removing %s...\n", configDir)
			if err := os.RemoveAll(configDir); err != nil {
				fmt.Printf("⚠️  Warning: %v\n", err)
			}
		}

		// Remove binary if in /usr/local/bin
		binaryPath := "/usr/local/bin/vpn-route-manager"
		if _, err := os.Stat(binaryPath); err == nil {
			fmt.Printf("🗑️  Removing %s...\n", binaryPath)
			if err := os.Remove(binaryPath); err != nil {
				fmt.Printf("⚠️  Warning: %v\n", err)
			}
		}

		fmt.Println("\n✅ Uninstallation completed!")
		return nil
	},
}

// Debug command
var debugCmd = &cobra.Command{
	Use:   "debug",
	Short: "Run in debug mode",
	RunE: func(cmd *cobra.Command, args []string) error {
		debug = true
		return runDaemon()
	},
}

// Logs command
var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "Show service logs",
	RunE: func(cmd *cobra.Command, args []string) error {
		follow, _ := cmd.Flags().GetBool("follow")
		lines, _ := cmd.Flags().GetInt("lines")
		
		homeDir, _ := os.UserHomeDir()
		logPath := filepath.Join(homeDir, ".vpn-route-manager", "logs", "vpn-route-manager.log")
		
		if _, err := os.Stat(logPath); os.IsNotExist(err) {
			return fmt.Errorf("log file not found: %s", logPath)
		}

		if follow {
			// Use tail -f
			tailCmd := exec.Command("tail", "-f", logPath)
			tailCmd.Stdout = os.Stdout
			tailCmd.Stderr = os.Stderr
			return tailCmd.Run()
		} else {
			// Show last N lines
			tailCmd := exec.Command("tail", fmt.Sprintf("-%d", lines), logPath)
			tailCmd.Stdout = os.Stdout
			tailCmd.Stderr = os.Stderr
			return tailCmd.Run()
		}
	},
}

// Config command group
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
}

var configGetCmd = &cobra.Command{
	Use:   "get [key]",
	Short: "Get configuration value",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		if len(args) == 0 {
			// Show all config
			data, err := json.MarshalIndent(cfg.Get(), "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
		} else {
			// Show specific key
			switch args[0] {
			case "gateway":
				fmt.Println(cfg.Get().Gateway)
			case "check_interval":
				fmt.Println(cfg.Get().CheckInterval)
			case "debug":
				fmt.Println(cfg.Get().Debug)
			default:
				return fmt.Errorf("unknown config key: %s", args[0])
			}
		}
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		key, value := args[0], args[1]
		config := cfg.Get()

		switch key {
		case "gateway":
			config.Gateway = value
		case "check_interval":
			var interval int
			if _, err := fmt.Sscanf(value, "%d", &interval); err != nil {
				return fmt.Errorf("invalid interval: %s", value)
			}
			config.CheckInterval = interval
		case "debug":
			config.Debug = value == "true"
		default:
			return fmt.Errorf("unknown config key: %s", key)
		}

		if err := cfg.Save(); err != nil {
			return err
		}

		fmt.Printf("✅ Set %s = %s\n", key, value)
		return nil
	},
}

func init() {
	// Add daemon flag to start command
	startCmd.Flags().Bool("daemon", false, "Run as daemon (internal use)")
	
	// Add flags to logs command
	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output")
	logsCmd.Flags().IntP("lines", "n", 50, "Number of lines to show")

	// Add config subcommands
	configCmd.AddCommand(configGetCmd, configSetCmd)
}

// runDaemon runs the service in daemon mode
func runDaemon() error {
	// Create logger
	log, err := createLogger()
	if err != nil {
		return fmt.Errorf("failed to create logger: %w", err)
	}
	defer log.Close()

	// Load configuration
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Create network manager
	netMgr := network.NewManager(log)

	// Create service manager
	svcMgr, err := service.NewManager(cfg, netMgr, log)
	if err != nil {
		return fmt.Errorf("failed to create service manager: %w", err)
	}

	// Start service
	if err := svcMgr.Start(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	// Handle SIGHUP separately (reload signal)
	hupChan := make(chan os.Signal, 1)
	signal.Notify(hupChan, syscall.SIGHUP)
	
	for {
		select {
		case sig := <-sigChan:
			log.Info("Received signal: %v", sig)
			// Stop the service gracefully
			if err := svcMgr.Stop(); err != nil {
				log.Error("Failed to stop service: %v", err)
			}
			return nil
		case <-hupChan:
			log.Info("Received SIGHUP - ignoring (reload not implemented)")
			// Continue running
		}
	}
}