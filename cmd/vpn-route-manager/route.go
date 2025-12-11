package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	"vpn-route-manager/internal/network"
)

// Route command group
var routeCmd = &cobra.Command{
	Use:   "route",
	Short: "Route management commands",
	Long:  "Manage network routes for VPN bypass",
}

var routeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List active routes",
	RunE: func(cmd *cobra.Command, args []string) error {
		log, err := createLogger()
		if err != nil {
			return err
		}
		defer log.Close()

		netMgr := network.NewManager(log)
		routes := netMgr.GetActiveRoutes()

		if len(routes) == 0 {
			fmt.Println("No active routes")
			return nil
		}

		// Print table
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NETWORK\tGATEWAY\tSERVICE\tAGE")
		fmt.Fprintln(w, "-------\t-------\t-------\t---")

		for _, route := range routes {
			age := time.Since(route.AddedAt).Round(time.Second)
			fmt.Fprintf(w, "%s\t%s\t%s\t%v\n", 
				route.Network, route.Gateway, route.Service, age)
		}
		w.Flush()

		fmt.Printf("\nTotal: %d routes\n", len(routes))
		return nil
	},
}

var routeAddCmd = &cobra.Command{
	Use:   "add <network>",
	Short: "Manually add a route",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		networkCIDR := args[0]
		gateway, _ := cmd.Flags().GetString("gateway")

		log, err := createLogger()
		if err != nil {
			return err
		}
		defer log.Close()

		netMgr := network.NewManager(log)

		// Detect gateway if not specified
		if gateway == "" {
			gateway, err = netMgr.DetectGateway()
			if err != nil {
				return fmt.Errorf("failed to detect gateway: %w", err)
			}
			fmt.Printf("Using detected gateway: %s\n", gateway)
		}

		// Add route
		if err := netMgr.AddRoute(networkCIDR, gateway, "manual"); err != nil {
			return fmt.Errorf("failed to add route: %w", err)
		}

		fmt.Printf("✅ Route added: %s -> %s\n", networkCIDR, gateway)
		return nil
	},
}

var routeRemoveCmd = &cobra.Command{
	Use:   "remove <network>",
	Short: "Manually remove a route",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		networkCIDR := args[0]

		log, err := createLogger()
		if err != nil {
			return err
		}
		defer log.Close()

		netMgr := network.NewManager(log)

		// Remove route
		if err := netMgr.RemoveRoute(networkCIDR); err != nil {
			return fmt.Errorf("failed to remove route: %w", err)
		}

		fmt.Printf("✅ Route removed: %s\n", networkCIDR)
		return nil
	},
}

var routeClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Remove all routes",
	RunE: func(cmd *cobra.Command, args []string) error {
		log, err := createLogger()
		if err != nil {
			return err
		}
		defer log.Close()

		netMgr := network.NewManager(log)
		routes := netMgr.GetActiveRoutes()

		if len(routes) == 0 {
			fmt.Println("No routes to remove")
			return nil
		}

		fmt.Printf("Remove %d routes? [y/N]: ", len(routes))
		var response string
		fmt.Scanln(&response)
		
		if response != "y" && response != "Y" {
			fmt.Println("Cancelled")
			return nil
		}

		// Remove all routes
		if err := netMgr.RemoveAllRoutes(); err != nil {
			return fmt.Errorf("failed to remove routes: %w", err)
		}

		fmt.Printf("✅ Removed %d routes\n", len(routes))
		return nil
	},
}

var routeTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Test route functionality",
	RunE: func(cmd *cobra.Command, args []string) error {
		log, err := createLogger()
		if err != nil {
			return err
		}
		defer log.Close()

		netMgr := network.NewManager(log)

		// Test gateway detection
		fmt.Println("🔍 Testing gateway detection...")
		gateway, err := netMgr.DetectGateway()
		if err != nil {
			fmt.Printf("❌ Gateway detection failed: %v\n", err)
		} else {
			fmt.Printf("✅ Detected gateway: %s\n", gateway)
		}

		// Test VPN detection
		fmt.Println("\n🔍 Testing VPN detection...")
		if netMgr.IsVPNConnected() {
			fmt.Println("✅ VPN is connected")
		} else {
			fmt.Println("❌ VPN is not connected")
		}

		// Test route verification
		routes := netMgr.GetActiveRoutes()
		if len(routes) > 0 {
			fmt.Printf("\n🔍 Verifying %d active routes...\n", len(routes))
			results := netMgr.VerifyRoutes()
			
			working := 0
			for network, ok := range results {
				if ok {
					fmt.Printf("✅ %s: Working\n", network)
					working++
				} else {
					fmt.Printf("❌ %s: Not working\n", network)
				}
			}
			
			fmt.Printf("\nVerification: %d/%d routes working\n", working, len(results))
		}

		return nil
	},
}

func init() {
	// Add subcommands
	routeCmd.AddCommand(
		routeListCmd,
		routeAddCmd,
		routeRemoveCmd,
		routeClearCmd,
		routeTestCmd,
	)

	// Add flags
	routeAddCmd.Flags().String("gateway", "", "Gateway IP (auto-detect if not specified)")
}