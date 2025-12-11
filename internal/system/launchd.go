package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"text/template"
)

// LaunchAgent handles macOS LaunchAgent management
type LaunchAgent struct {
	serviceName string
	plistPath   string
	username    string
}

// LaunchAgentConfig holds configuration for the plist template
type LaunchAgentConfig struct {
	Label            string
	BinaryPath       string
	WorkingDirectory string
	LogDirectory     string
	Username         string
	HomeDirectory    string
}

// NewLaunchAgent creates a new LaunchAgent manager
func NewLaunchAgent(username string) *LaunchAgent {
	serviceName := fmt.Sprintf("com.%s.vpn.route.manager", username)
	homeDir, _ := os.UserHomeDir()
	plistPath := filepath.Join(homeDir, "Library", "LaunchAgents", serviceName+".plist")

	return &LaunchAgent{
		serviceName: serviceName,
		plistPath:   plistPath,
		username:    username,
	}
}

// Install creates and loads the LaunchAgent
func (la *LaunchAgent) Install(binaryPath string) error {
	// Ensure LaunchAgents directory exists
	launchAgentsDir := filepath.Dir(la.plistPath)
	if err := os.MkdirAll(launchAgentsDir, 0755); err != nil {
		return fmt.Errorf("failed to create LaunchAgents directory: %w", err)
	}

	// Create plist file
	if err := la.createPlist(binaryPath); err != nil {
		return fmt.Errorf("failed to create plist: %w", err)
	}

	// Load the agent
	if err := la.Load(); err != nil {
		return fmt.Errorf("failed to load agent: %w", err)
	}

	return nil
}

// Uninstall unloads and removes the LaunchAgent
func (la *LaunchAgent) Uninstall() error {
	// Unload if loaded
	if la.IsLoaded() {
		if err := la.Unload(); err != nil {
			return fmt.Errorf("failed to unload agent: %w", err)
		}
	}

	// Remove plist file
	if err := os.Remove(la.plistPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove plist: %w", err)
	}

	return nil
}

// Load loads the LaunchAgent
func (la *LaunchAgent) Load() error {
	cmd := exec.Command("launchctl", "load", la.plistPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl load failed: %s", string(output))
	}
	return nil
}

// Unload unloads the LaunchAgent
func (la *LaunchAgent) Unload() error {
	cmd := exec.Command("launchctl", "unload", la.plistPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("launchctl unload failed: %s", string(output))
	}
	return nil
}

// IsLoaded checks if the LaunchAgent is loaded
func (la *LaunchAgent) IsLoaded() bool {
	cmd := exec.Command("launchctl", "list", la.serviceName)
	err := cmd.Run()
	return err == nil
}

// IsRunning checks if the service is actually running
func (la *LaunchAgent) IsRunning() (bool, int) {
	// Use launchctl list without service name and grep for it
	// This gives us the simple format: "PID Status Label"
	cmd := exec.Command("sh", "-c", fmt.Sprintf("launchctl list | grep %s", la.serviceName))
	output, err := cmd.Output()
	if err != nil {
		return false, 0
	}

	// Parse launchctl list output
	// Format: "PID	Status	Label" or "-	Status	Label" if not running
	outputStr := strings.TrimSpace(string(output))
	parts := strings.Fields(outputStr)
	if len(parts) < 3 {
		return false, 0
	}

	// First field is PID or "-"
	if parts[0] == "-" {
		return false, 0
	}

	pid, err := strconv.Atoi(parts[0])
	if err != nil {
		return false, 0
	}
	
	// Verify the process is actually running
	if pid > 0 {
		// Check if process exists
		process, err := os.FindProcess(pid)
		if err != nil {
			return false, 0
		}
		
		// Send signal 0 to check if process is alive
		err = process.Signal(syscall.Signal(0))
		if err != nil {
			return false, 0
		}
		
		return true, pid
	}
	
	return false, 0
}

// createPlist creates the LaunchAgent plist file
func (la *LaunchAgent) createPlist(binaryPath string) error {
	homeDir, _ := os.UserHomeDir()
	
	config := LaunchAgentConfig{
		Label:            la.serviceName,
		BinaryPath:       binaryPath,
		WorkingDirectory: filepath.Dir(binaryPath),
		LogDirectory:     filepath.Join(homeDir, ".vpn-route-manager", "logs"),
		Username:         la.username,
		HomeDirectory:    homeDir,
	}

	tmpl, err := template.New("plist").Parse(plistTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	file, err := os.Create(la.plistPath)
	if err != nil {
		return fmt.Errorf("failed to create plist file: %w", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, config); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	return nil
}

const plistTemplate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>{{.Label}}</string>
    
    <key>ProgramArguments</key>
    <array>
        <string>{{.BinaryPath}}</string>
        <string>start</string>
        <string>--daemon</string>
    </array>
    
    <key>WorkingDirectory</key>
    <string>{{.WorkingDirectory}}</string>
    
    <key>RunAtLoad</key>
    <true/>
    
    <key>KeepAlive</key>
    <true/>
    
    <key>ProcessType</key>
    <string>Background</string>
    
    <key>StandardOutPath</key>
    <string>{{.LogDirectory}}/stdout.log</string>
    
    <key>StandardErrorPath</key>
    <string>{{.LogDirectory}}/stderr.log</string>
    
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin</string>
        <key>HOME</key>
        <string>{{.HomeDirectory}}</string>
        <key>USER</key>
        <string>{{.Username}}</string>
    </dict>
    
    <key>ThrottleInterval</key>
    <integer>10</integer>
    
    <key>StartInterval</key>
    <integer>15</integer>
    
    <key>ExitTimeOut</key>
    <integer>30</integer>
    
    <key>Nice</key>
    <integer>1</integer>
    
    <key>LowPriorityIO</key>
    <true/>
</dict>
</plist>
`