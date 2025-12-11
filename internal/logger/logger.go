package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Level represents log levels
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
)

// Logger handles structured logging with rotation
type Logger struct {
	mu           sync.Mutex
	level        Level
	file         *os.File
	logger       *log.Logger
	logPath      string
	maxSize      int64
	maxBackups   int
	rotator      *Rotator
	debugEnabled bool
}

// Config holds logger configuration
type Config struct {
	LogPath      string
	MaxSizeMB    int
	MaxBackups   int
	Debug        bool
}

// New creates a new logger instance
func New(config Config) (*Logger, error) {
	// Ensure log directory exists
	logDir := filepath.Dir(config.LogPath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open or create log file
	file, err := os.OpenFile(config.LogPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	level := InfoLevel
	if config.Debug {
		level = DebugLevel
	}

	l := &Logger{
		level:        level,
		file:         file,
		logger:       log.New(io.MultiWriter(file, os.Stdout), "", 0),
		logPath:      config.LogPath,
		maxSize:      int64(config.MaxSizeMB) * 1024 * 1024,
		maxBackups:   config.MaxBackups,
		debugEnabled: config.Debug,
	}

	// Initialize rotator
	l.rotator = NewRotator(l)

	// Write startup message
	l.Info("VPN Route Manager started")
	l.Info("Log file: %s", config.LogPath)
	l.Info("Debug mode: %v", config.Debug)

	return l, nil
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetDebug enables or disables debug logging
func (l *Logger) SetDebug(enabled bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.debugEnabled = enabled
	if enabled {
		l.level = DebugLevel
	} else {
		l.level = InfoLevel
	}
}

// log writes a log entry with the specified level
func (l *Logger) log(level Level, format string, args ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if level < l.level {
		return
	}

	// Check if rotation is needed
	if l.rotator.ShouldRotate() {
		if err := l.rotator.Rotate(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to rotate log: %v\n", err)
		}
	}

	// Format timestamp and level
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	levelStr := l.levelString(level)
	
	// Format message
	message := fmt.Sprintf(format, args...)
	
	// Write log entry
	logEntry := fmt.Sprintf("%s [%s] %s", timestamp, levelStr, message)
	l.logger.Println(logEntry)
}

// Debug logs a debug message
func (l *Logger) Debug(format string, args ...interface{}) {
	l.log(DebugLevel, format, args...)
}

// Info logs an info message
func (l *Logger) Info(format string, args ...interface{}) {
	l.log(InfoLevel, format, args...)
}

// Warn logs a warning message
func (l *Logger) Warn(format string, args ...interface{}) {
	l.log(WarnLevel, format, args...)
}

// Error logs an error message
func (l *Logger) Error(format string, args ...interface{}) {
	l.log(ErrorLevel, format, args...)
}

// Fatal logs a fatal error and exits
func (l *Logger) Fatal(format string, args ...interface{}) {
	l.log(ErrorLevel, "FATAL: "+format, args...)
	os.Exit(1)
}

// Close closes the logger
func (l *Logger) Close() error {
	// Log shutdown message before locking
	l.Info("VPN Route Manager shutting down")
	
	l.mu.Lock()
	defer l.mu.Unlock()
	
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// levelString returns the string representation of a log level
func (l *Logger) levelString(level Level) string {
	switch level {
	case DebugLevel:
		return "DEBUG"
	case InfoLevel:
		return "INFO"
	case WarnLevel:
		return "WARN"
	case ErrorLevel:
		return "ERROR"
	default:
		return "UNKNOWN"
	}
}

// GetLogPath returns the current log file path
func (l *Logger) GetLogPath() string {
	return l.logPath
}

// GetLogSize returns the current log file size
func (l *Logger) GetLogSize() (int64, error) {
	// Don't lock here as this is called from within log() which already holds the lock
	info, err := l.file.Stat()
	if err != nil {
		return 0, err
	}
	return info.Size(), nil
}

// reopenFile reopens the log file (used by rotator)
func (l *Logger) reopenFile() error {
	if l.file != nil {
		l.file.Close()
	}

	file, err := os.OpenFile(l.logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	l.file = file
	l.logger = log.New(io.MultiWriter(file, os.Stdout), "", 0)
	return nil
}