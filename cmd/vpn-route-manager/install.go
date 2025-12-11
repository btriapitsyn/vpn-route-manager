package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"
	"vpn-route-manager/internal/config"
	"vpn-route-manager/internal/system"
)

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install VPN Route Manager as a system service",
	Long:  `Installs VPN Route Manager as a macOS LaunchAgent that starts automatically at login.`,
	RunE:  runInstall,
}

func runInstall(cmd *cobra.Command, args []string) error {
	fmt.Println("🚀 Installing VPN Route Manager...")

	// Get current user
	username := os.Getenv("USER")
	if username == "" {
		return fmt.Errorf("could not determine current user")
	}

	// For system operations, check if we have necessary permissions
	if os.Geteuid() != 0 {
		// Check if we can write to /usr/local/bin
		testFile := "/usr/local/bin/.vpn-route-manager-test"
		if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
			fmt.Println("\n⚠️  This command requires administrator privileges.")
			fmt.Println("Please run with sudo:")
			fmt.Printf("\n  sudo %s install\n\n", os.Args[0])
			return fmt.Errorf("insufficient privileges")
		}
		os.Remove(testFile)
	}

	// Get binary path
	binaryPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	// Ensure binary is in a permanent location
	homeDir, _ := os.UserHomeDir()
	installPath := filepath.Join("/usr/local/bin", "vpn-route-manager")
	
	// Check if we need to copy the binary
	if binaryPath != installPath {
		fmt.Printf("📁 Installing binary to %s...\n", installPath)
		
		// Ensure /usr/local/bin exists
		if err := os.MkdirAll("/usr/local/bin", 0755); err != nil {
			return fmt.Errorf("failed to create /usr/local/bin: %w", err)
		}

		// Copy binary
		copyCmd := exec.Command("cp", binaryPath, installPath)
		if output, err := copyCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to copy binary: %s", string(output))
		}

		// Make executable
		if err := os.Chmod(installPath, 0755); err != nil {
			return fmt.Errorf("failed to make binary executable: %w", err)
		}
		
		binaryPath = installPath
	}

	// Create configuration directories
	fmt.Println("📂 Creating configuration directories...")
	configDir := filepath.Join(homeDir, ".vpn-route-manager")
	dirs := []string{
		filepath.Join(configDir, "config"),
		filepath.Join(configDir, "config", "services"),
		filepath.Join(configDir, "logs"),
		filepath.Join(configDir, "state"),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// Create default configuration
	fmt.Println("⚙️  Creating default configuration...")
	cfgManager := config.NewManager(filepath.Join(configDir, "config", "config.json"))
	
	// Set default services
	cfg := cfgManager.Get()
	cfg.Services = config.GetDefaultServiceConfigs()
	
	// Ensure directories are set
	if err := config.EnsureDirectories(cfg); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Save configuration
	if err := cfgManager.Save(); err != nil {
		return fmt.Errorf("failed to save configuration: %w", err)
	}

	// Save individual service files
	servicesDir := filepath.Join(configDir, "config", "services")
	for name, svc := range cfg.Services {
		svcPath := filepath.Join(servicesDir, name+".json")
		if err := saveServiceFile(svcPath, svc); err != nil {
			fmt.Printf("⚠️  Warning: failed to save service %s: %v\n", name, err)
		}
	}

	// Setup sudo permissions
	fmt.Println("🔐 Setting up sudo permissions...")
	sudoMgr := system.NewSudoManager(username)
	if err := sudoMgr.Setup(); err != nil {
		return fmt.Errorf("failed to setup sudo: %w", err)
	}

	// Test sudo access
	if err := sudoMgr.TestAccess(); err != nil {
		return fmt.Errorf("sudo test failed: %w", err)
	}
	fmt.Println("✅ Sudo permissions configured")

	// Install LaunchAgent
	fmt.Println("🎯 Installing LaunchAgent...")
	launchAgent := system.NewLaunchAgent(username)
	if err := launchAgent.Install(binaryPath); err != nil {
		return fmt.Errorf("failed to install LaunchAgent: %w", err)
	}

	// Verify installation
	if launchAgent.IsLoaded() {
		fmt.Println("✅ LaunchAgent installed and loaded")
		
		// Check if running
		if running, pid := launchAgent.IsRunning(); running {
			fmt.Printf("✅ Service is running (PID: %d)\n", pid)
		} else {
			fmt.Println("⚠️  Service loaded but not yet running")
		}
	} else {
		return fmt.Errorf("LaunchAgent installation verification failed")
	}

	// Print summary
	fmt.Println("\n✅ Installation completed successfully!")
	fmt.Println("\n📋 Installation Summary:")
	fmt.Printf("  • Binary: %s\n", binaryPath)
	fmt.Printf("  • Config: %s\n", filepath.Join(configDir, "config", "config.json"))
	fmt.Printf("  • Services: %s\n", servicesDir)
	fmt.Printf("  • Logs: %s\n", filepath.Join(configDir, "logs"))
	fmt.Println("\n📋 Default Services:")
	fmt.Println("  ✅ Telegram: ENABLED")
	fmt.Println("  ✅ YouTube: ENABLED")
	fmt.Println("  ❌ WhatsApp: disabled")
	fmt.Println("  ❌ Spotify: disabled")
	fmt.Println("  ❌ Apple Music: disabled")
	fmt.Println("  ❌ Facebook: disabled")
	fmt.Println("  ❌ Instagram: disabled")
	fmt.Println("\n💡 Management Commands:")
	fmt.Println("  • Status:  vpn-route-manager status")
	fmt.Println("  • Services: vpn-route-manager service list")
	fmt.Println("  • Logs:    vpn-route-manager logs")
	fmt.Println("\n🎉 VPN Route Manager is now monitoring your VPN connection!")

	return nil
}

func saveServiceFile(path string, service *config.Service) error {
	// Create wrapper format for compatibility
	wrapper := map[string]*config.Service{
		service.Name: service,
	}
	
	data, err := json.MarshalIndent(wrapper, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(path, data, 0644)
}