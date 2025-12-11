package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"vpn-route-manager/internal/config"
	"vpn-route-manager/internal/system"
)

// Service command group
var serviceCmd = &cobra.Command{
	Use:   "service",
	Short: "Service management commands",
	Long:  "Manage services that can bypass VPN connections",
}

var serviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all services",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		services := cfg.Get().Services
		if len(services) == 0 {
			fmt.Println("No services configured")
			return nil
		}

		// Sort services by name
		var names []string
		for name := range services {
			names = append(names, name)
		}
		sort.Strings(names)

		// Print table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tSTATUS\tNETWORKS\tDESCRIPTION")
		fmt.Fprintln(w, "----\t------\t--------\t-----------")

		for _, name := range names {
			svc := services[name]
			status := "DISABLED"
			if svc.Enabled {
				status = "ENABLED"
			}
			fmt.Fprintf(w, "%s\t%s\t%d\t%s\n", 
				name, status, len(svc.Networks), svc.Description)
		}
		w.Flush()

		return nil
	},
}

var serviceShowCmd = &cobra.Command{
	Use:   "show <name>",
	Short: "Show service details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		name := args[0]
		svc, exists := cfg.Get().Services[name]
		if !exists {
			return fmt.Errorf("service '%s' not found", name)
		}

		fmt.Printf("Service: %s\n", svc.Name)
		fmt.Printf("Description: %s\n", svc.Description)
		fmt.Printf("Enabled: %v\n", svc.Enabled)
		fmt.Printf("Priority: %d\n", svc.Priority)
		
		fmt.Printf("\nNetworks (%d):\n", len(svc.Networks))
		for _, network := range svc.Networks {
			fmt.Printf("  %s\n", network)
		}

		if len(svc.Domains) > 0 {
			fmt.Printf("\nDomains (%d):\n", len(svc.Domains))
			for _, domain := range svc.Domains {
				fmt.Printf("  %s\n", domain)
			}
		}

		return nil
	},
}

var serviceEnableCmd = &cobra.Command{
	Use:   "enable <name>",
	Short: "Enable a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		name := args[0]
		if err := cfg.EnableService(name); err != nil {
			return err
		}

		if err := cfg.Save(); err != nil {
			return err
		}

		fmt.Printf("✅ Service '%s' enabled\n", name)
		fmt.Println("💡 Routes will be added when VPN connects")
		
		// Check if daemon is running
		username := os.Getenv("USER")
		launchAgent := system.NewLaunchAgent(username)
		if running, _ := launchAgent.IsRunning(); running {
			fmt.Println("⚠️  Restart the service to apply changes: vpn-route-manager restart")
		}
		
		return nil
	},
}

var serviceDisableCmd = &cobra.Command{
	Use:   "disable <name>",
	Short: "Disable a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		name := args[0]
		if err := cfg.DisableService(name); err != nil {
			return err
		}

		if err := cfg.Save(); err != nil {
			return err
		}

		fmt.Printf("✅ Service '%s' disabled\n", name)
		fmt.Println("💡 Routes will be removed if currently active")
		
		// Check if daemon is running
		username := os.Getenv("USER")
		launchAgent := system.NewLaunchAgent(username)
		if running, _ := launchAgent.IsRunning(); running {
			fmt.Println("⚠️  Restart the service to apply changes: vpn-route-manager restart")
		}
		
		return nil
	},
}

var serviceAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		networks, _ := cmd.Flags().GetString("networks")
		description, _ := cmd.Flags().GetString("description")
		priority, _ := cmd.Flags().GetInt("priority")

		if networks == "" {
			return fmt.Errorf("--networks is required")
		}

		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		// Check if service already exists
		if _, exists := cfg.Get().Services[name]; exists {
			return fmt.Errorf("service '%s' already exists", name)
		}

		// Parse networks
		networkList := strings.Split(networks, ",")
		for i, net := range networkList {
			networkList[i] = strings.TrimSpace(net)
		}

		// Create service
		service := &config.Service{
			Name:        name,
			Description: description,
			Enabled:     false,
			Networks:    networkList,
			Priority:    priority,
		}

		// Validate service
		if err := config.ValidateService(name, service); err != nil {
			return err
		}

		// Add to config
		cfg.Get().Services[name] = service

		// Save
		if err := cfg.Save(); err != nil {
			return err
		}

		fmt.Printf("✅ Service '%s' added (disabled by default)\n", name)
		fmt.Printf("💡 Enable with: vpn-route-manager service enable %s\n", name)
		return nil
	},
}

var serviceRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a service",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		
		cfg, err := loadConfig()
		if err != nil {
			return err
		}

		if _, exists := cfg.Get().Services[name]; !exists {
			return fmt.Errorf("service '%s' not found", name)
		}

		// Confirm
		fmt.Printf("Remove service '%s'? [y/N]: ", name)
		var response string
		fmt.Scanln(&response)
		
		if strings.ToLower(response) != "y" {
			fmt.Println("Cancelled")
			return nil
		}

		delete(cfg.Get().Services, name)

		if err := cfg.Save(); err != nil {
			return err
		}

		fmt.Printf("✅ Service '%s' removed\n", name)
		return nil
	},
}

func init() {
	// Add subcommands
	serviceCmd.AddCommand(
		serviceListCmd,
		serviceShowCmd,
		serviceEnableCmd,
		serviceDisableCmd,
		serviceAddCmd,
		serviceRemoveCmd,
	)

	// Add flags to add command
	serviceAddCmd.Flags().String("networks", "", "Comma-separated list of networks (CIDR format)")
	serviceAddCmd.Flags().String("description", "", "Service description")
	serviceAddCmd.Flags().Int("priority", 50, "Service priority (0-1000)")
}