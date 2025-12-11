package system

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// ProcessManager handles process management
type ProcessManager struct {
	processName string
}

// NewProcessManager creates a new process manager
func NewProcessManager(processName string) *ProcessManager {
	return &ProcessManager{
		processName: processName,
	}
}

// FindProcess finds processes by name
func (pm *ProcessManager) FindProcess() ([]int, error) {
	cmd := exec.Command("pgrep", "-f", pm.processName)
	output, err := cmd.Output()
	if err != nil {
		// pgrep returns exit code 1 when no processes found
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return []int{}, nil
		}
		return nil, fmt.Errorf("failed to find process: %w", err)
	}

	var pids []int
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		pid, err := strconv.Atoi(line)
		if err != nil {
			continue
		}
		pids = append(pids, pid)
	}

	return pids, nil
}

// IsProcessRunning checks if a specific PID is running
func (pm *ProcessManager) IsProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Send signal 0 to check if process exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// KillProcess kills a process by PID
func (pm *ProcessManager) KillProcess(pid int, force bool) error {
	process, err := os.FindProcess(pid)
	if err != nil {
		return fmt.Errorf("process not found: %w", err)
	}

	if force {
		return process.Kill()
	}

	// Try graceful termination first
	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	// Wait for process to exit
	done := make(chan bool)
	go func() {
		for i := 0; i < 10; i++ {
			if !pm.IsProcessRunning(pid) {
				done <- true
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
		done <- false
	}()

	if <-done {
		return nil
	}

	// Force kill if still running
	return process.Kill()
}

// KillAllProcesses kills all processes matching the name
func (pm *ProcessManager) KillAllProcesses(force bool) error {
	pids, err := pm.FindProcess()
	if err != nil {
		return err
	}

	var errors []error
	for _, pid := range pids {
		if pid == os.Getpid() {
			continue // Don't kill ourselves
		}
		if err := pm.KillProcess(pid, force); err != nil {
			errors = append(errors, fmt.Errorf("failed to kill PID %d: %w", pid, err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to kill some processes: %v", errors)
	}

	return nil
}

// GetProcessInfo returns information about running processes
func (pm *ProcessManager) GetProcessInfo() ([]ProcessInfo, error) {
	pids, err := pm.FindProcess()
	if err != nil {
		return nil, err
	}

	var infos []ProcessInfo
	for _, pid := range pids {
		info, err := pm.getProcessInfoByPID(pid)
		if err != nil {
			continue
		}
		infos = append(infos, info)
	}

	return infos, nil
}

// ProcessInfo contains process information
type ProcessInfo struct {
	PID       int
	Command   string
	StartTime time.Time
	CPUUsage  float64
	Memory    int64
}

// getProcessInfoByPID gets detailed info for a specific PID
func (pm *ProcessManager) getProcessInfoByPID(pid int) (ProcessInfo, error) {
	cmd := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "pid,comm,lstart,pcpu,rss")
	output, err := cmd.Output()
	if err != nil {
		return ProcessInfo{}, err
	}

	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return ProcessInfo{}, fmt.Errorf("unexpected ps output")
	}

	// Parse the output (skip header)
	fields := strings.Fields(lines[1])
	if len(fields) < 5 {
		return ProcessInfo{}, fmt.Errorf("insufficient fields in ps output")
	}

	info := ProcessInfo{
		PID:     pid,
		Command: fields[1],
	}

	// Parse CPU usage
	if cpu, err := strconv.ParseFloat(fields[len(fields)-2], 64); err == nil {
		info.CPUUsage = cpu
	}

	// Parse memory (RSS in KB)
	if mem, err := strconv.ParseInt(fields[len(fields)-1], 10, 64); err == nil {
		info.Memory = mem * 1024 // Convert to bytes
	}

	return info, nil
}

// WaitForProcess waits for a process to start
func (pm *ProcessManager) WaitForProcess(timeout time.Duration) (int, error) {
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		pids, err := pm.FindProcess()
		if err != nil {
			return 0, err
		}

		if len(pids) > 0 {
			return pids[0], nil
		}

		time.Sleep(100 * time.Millisecond)
	}

	return 0, fmt.Errorf("timeout waiting for process to start")
}

// CreatePIDFile creates a PID file for the current process
func CreatePIDFile(path string) error {
	pid := os.Getpid()
	return os.WriteFile(path, []byte(strconv.Itoa(pid)), 0644)
}

// ReadPIDFile reads a PID from a file
func ReadPIDFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID in file: %w", err)
	}

	return pid, nil
}

// RemovePIDFile removes a PID file
func RemovePIDFile(path string) error {
	return os.Remove(path)
}