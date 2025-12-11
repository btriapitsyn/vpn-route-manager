package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Rotator handles log file rotation
type Rotator struct {
	logger     *Logger
	maxSize    int64
	maxBackups int
}

// NewRotator creates a new log rotator
func NewRotator(logger *Logger) *Rotator {
	return &Rotator{
		logger:     logger,
		maxSize:    logger.maxSize,
		maxBackups: logger.maxBackups,
	}
}

// ShouldRotate checks if rotation is needed
func (r *Rotator) ShouldRotate() bool {
	size, err := r.logger.GetLogSize()
	if err != nil {
		return false
	}
	return size >= r.maxSize
}

// Rotate performs log rotation
func (r *Rotator) Rotate() error {
	// Close current file
	if r.logger.file != nil {
		r.logger.file.Close()
	}

	// Get base path and extension
	basePath := r.logger.logPath
	ext := filepath.Ext(basePath)
	base := strings.TrimSuffix(basePath, ext)

	// Rename existing backups
	for i := r.maxBackups - 1; i > 0; i-- {
		oldPath := fmt.Sprintf("%s.%d%s", base, i, ext)
		newPath := fmt.Sprintf("%s.%d%s", base, i+1, ext)
		
		if _, err := os.Stat(oldPath); err == nil {
			os.Rename(oldPath, newPath)
		}
	}

	// Rename current log to .1
	backupPath := fmt.Sprintf("%s.1%s", base, ext)
	if err := os.Rename(basePath, backupPath); err != nil {
		return fmt.Errorf("failed to rename log file: %w", err)
	}

	// Create new log file
	if err := r.logger.reopenFile(); err != nil {
		return fmt.Errorf("failed to create new log file: %w", err)
	}

	// Clean up old logs
	r.cleanOldLogs()

	r.logger.Info("Log rotated successfully")
	return nil
}

// cleanOldLogs removes logs beyond maxBackups
func (r *Rotator) cleanOldLogs() {
	basePath := r.logger.logPath
	ext := filepath.Ext(basePath)
	base := strings.TrimSuffix(basePath, ext)
	dir := filepath.Dir(basePath)

	// Find all backup files
	pattern := fmt.Sprintf("%s.*%s", filepath.Base(base), ext)
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err != nil {
		return
	}

	// Sort by modification time
	type fileInfo struct {
		path    string
		modTime time.Time
	}

	var files []fileInfo
	for _, match := range matches {
		if match == basePath {
			continue // Skip current log
		}
		
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		
		files = append(files, fileInfo{
			path:    match,
			modTime: info.ModTime(),
		})
	}

	// Sort by modification time (newest first)
	sort.Slice(files, func(i, j int) bool {
		return files[i].modTime.After(files[j].modTime)
	})

	// Remove files beyond maxBackups
	for i := r.maxBackups; i < len(files); i++ {
		os.Remove(files[i].path)
	}

	// Also remove files older than 30 days
	cutoff := time.Now().Add(-30 * 24 * time.Hour)
	for _, file := range files {
		if file.modTime.Before(cutoff) {
			os.Remove(file.path)
		}
	}
}

// GetLogFiles returns all log files (current and backups)
func (r *Rotator) GetLogFiles() ([]string, error) {
	basePath := r.logger.logPath
	ext := filepath.Ext(basePath)
	base := strings.TrimSuffix(basePath, ext)
	dir := filepath.Dir(basePath)

	var files []string
	
	// Add current log file
	if _, err := os.Stat(basePath); err == nil {
		files = append(files, basePath)
	}

	// Add backup files
	for i := 1; i <= r.maxBackups; i++ {
		backupPath := fmt.Sprintf("%s.%d%s", base, i, ext)
		if _, err := os.Stat(backupPath); err == nil {
			files = append(files, backupPath)
		}
	}

	// Also check for any other matching files
	pattern := fmt.Sprintf("%s.*%s", filepath.Base(base), ext)
	matches, err := filepath.Glob(filepath.Join(dir, pattern))
	if err == nil {
		for _, match := range matches {
			// Avoid duplicates
			found := false
			for _, existing := range files {
				if existing == match {
					found = true
					break
				}
			}
			if !found {
				files = append(files, match)
			}
		}
	}

	return files, nil
}

// GetTotalLogSize returns the total size of all log files
func (r *Rotator) GetTotalLogSize() (int64, error) {
	files, err := r.GetLogFiles()
	if err != nil {
		return 0, err
	}

	var totalSize int64
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}
		totalSize += info.Size()
	}

	return totalSize, nil
}