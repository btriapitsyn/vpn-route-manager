package system

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// SudoManager handles sudo configuration
type SudoManager struct {
	username    string
	sudoersFile string
}

// NewSudoManager creates a new sudo manager
func NewSudoManager(username string) *SudoManager {
	return &SudoManager{
		username:    username,
		sudoersFile: fmt.Sprintf("/etc/sudoers.d/vpn-route-bypass-%s", username),
	}
}

// Setup configures passwordless sudo for route commands
func (sm *SudoManager) Setup() error {
	// Check if already configured
	if sm.IsConfigured() {
		return nil
	}

	// Create sudoers content
	content := fmt.Sprintf("%s ALL=(root) NOPASSWD: /sbin/route\n", sm.username)

	// Write to temporary file
	tmpFile := filepath.Join("/tmp", fmt.Sprintf("sudoers-%s-%d", sm.username, os.Getpid()))
	if err := os.WriteFile(tmpFile, []byte(content), 0440); err != nil {
		return fmt.Errorf("failed to create temp sudoers file: %w", err)
	}
	defer os.Remove(tmpFile)

	// Validate sudoers file
	cmd := exec.Command("visudo", "-c", "-f", tmpFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("invalid sudoers syntax: %s", string(output))
	}

	// Move to sudoers.d
	cmd = exec.Command("sudo", "cp", tmpFile, sm.sudoersFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to install sudoers file: %s", string(output))
	}

	// Set correct permissions
	cmd = exec.Command("sudo", "chmod", "440", sm.sudoersFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to set sudoers permissions: %s", string(output))
	}

	return nil
}

// Remove removes the sudo configuration
func (sm *SudoManager) Remove() error {
	if !sm.IsConfigured() {
		return nil
	}

	cmd := exec.Command("sudo", "rm", "-f", sm.sudoersFile)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to remove sudoers file: %s", string(output))
	}

	return nil
}

// IsConfigured checks if sudo is already configured
func (sm *SudoManager) IsConfigured() bool {
	// Test if we can run route command without password
	cmd := exec.Command("sudo", "-n", "route", "get", "default")
	err := cmd.Run()
	return err == nil
}

// TestAccess verifies sudo access works
func (sm *SudoManager) TestAccess() error {
	if !sm.IsConfigured() {
		return fmt.Errorf("sudo not configured for passwordless route access")
	}

	// Try to get default route
	cmd := exec.Command("sudo", "-n", "route", "get", "default")
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("sudo test failed: %s", string(output))
	}

	return nil
}

// GetSudoersFile returns the path to the sudoers file
func (sm *SudoManager) GetSudoersFile() string {
	return sm.sudoersFile
}

// RequiresSudo checks if the current process needs sudo
func RequiresSudo() bool {
	return os.Geteuid() != 0
}

// EnsureSudo ensures the command is running with sudo
func EnsureSudo() error {
	if !RequiresSudo() {
		return nil
	}

	// Re-execute with sudo
	args := append([]string{"sudo"}, os.Args...)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute with sudo: %w", err)
	}

	// Exit as the sudo version is now running
	os.Exit(0)
	return nil
}