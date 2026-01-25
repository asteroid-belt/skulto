// Package log provides logging functionality to both console and file.
package log

import (
	"fmt"
	"io"
	stdlog "log"
	"os"
	"path/filepath"
	"time"
)

// Logger writes output to both console and a log file.
type Logger struct {
	file   *os.File
	writer io.Writer
}

// New creates a new logger that writes to both console and a log file.
// The log file is created in the specified directory with a timestamp.
func New(logDir string) (*Logger, error) {
	// Ensure log directory exists
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}

	// Create log file with timestamp
	logPath := filepath.Join(logDir, "skulto.log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	// Create a multi-writer that writes to both stdout and file
	multiWriter := io.MultiWriter(os.Stdout, file)

	return &Logger{
		file:   file,
		writer: multiWriter,
	}, nil
}

// Printf writes a formatted message to console and log file.
func (l *Logger) Printf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	_, _ = fmt.Fprint(l.writer, msg)
}

// Println writes a message to console and log file with a newline.
func (l *Logger) Println(args ...interface{}) {
	msg := fmt.Sprintln(args...)
	_, _ = fmt.Fprint(l.writer, msg)
}

// Errorf writes a formatted error message to stderr and log file.
func (l *Logger) Errorf(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	formatted := fmt.Sprintf("[%s] %s\n", timestamp, msg)
	_, _ = fmt.Fprint(os.Stderr, formatted)
	_, _ = fmt.Fprint(l.file, formatted)
}

// Close closes the log file.
func (l *Logger) Close() error {
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

// Global logger instance
var globalLogger *Logger

// Init initializes the global logger.
// Also redirects Go's standard log package to write to the log file,
// so any log.Printf calls (e.g., timing instrumentation) go to the file.
func Init(logDir string) error {
	logger, err := New(logDir)
	if err != nil {
		return err
	}
	globalLogger = logger

	// Redirect Go's standard log package to write to our log file
	// This ensures timing/debug logs don't corrupt the TUI
	stdlog.SetOutput(logger.file)
	stdlog.SetFlags(stdlog.Ldate | stdlog.Ltime)

	return nil
}

// Printf uses the global logger to print formatted output.
func Printf(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Printf(format, args...)
	} else {
		fmt.Printf(format, args...)
	}
}

// Println uses the global logger to print output with newline.
func Println(args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Println(args...)
	} else {
		fmt.Println(args...)
	}
}

// Errorf uses the global logger to print formatted error output.
func Errorf(format string, args ...interface{}) {
	if globalLogger != nil {
		globalLogger.Errorf(format, args...)
	} else {
		fmt.Fprintf(os.Stderr, format, args...)
	}
}

// Close closes the global logger.
func Close() error {
	if globalLogger != nil {
		return globalLogger.Close()
	}
	return nil
}
