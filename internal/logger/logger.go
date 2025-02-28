package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
)

const (
	DEBUG = iota
	INFO
	WARN
	ERROR
)

type iLogger interface {
	Debug(message string, v ...any)
	Info(message string, v ...any)
	Warn(message string, v ...any)
	Error(message string, v ...any)
	SetLogLevel(logLevel string) error
	SetOutputDestination(destination string) error
}

type Logger struct {
	Level  int
	Silent bool
	*log.Logger
}

// New creates a new logger with the specified log level and output destination
func New(level string, destination string, silent bool) *Logger {
	l := &Logger{
		Silent: silent,
	}

	if err := l.SetOutputDestination(destination); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set output destination: %v\n", err)
		os.Exit(1)
	}

	if err := l.SetLogLevel(level); err != nil {
		fmt.Printf("Logger initialized with level: %d\n", l.Level)
		fmt.Fprintf(os.Stderr, "failed to set log level: %v\n", err)
		os.Exit(1)
	}

	return l
}

// shouldLog returns true if the log level is less than or equal to the specified log level
func (l *Logger) shouldLog(logLevel int) bool {
	return l.Level <= logLevel
}

// logOutput logs the message to the output destination
func (l *Logger) logOutput(level string, message string, v ...any) {
	if l.Silent {
		return
	}

	// Format message and arguments
	formattedMessage := fmt.Sprintf(message, v...)
	l.Logger.Output(2, fmt.Sprintf("%s %s", level, formattedMessage))
}

// Log debug, info, warn, and error messages
func (l *Logger) Debug(message string, v ...any) {
	if l.Level <= DEBUG {
		l.logOutput("[DEBUG]", message, v...)
	}
}

// Log info, warn, and error messages
func (l *Logger) Info(message string, v ...any) {
	if l.Level <= INFO {
		l.logOutput("[INFO]", message, v...)
	}
}

// Log warn and error messages
func (l *Logger) Warn(message string, v ...any) {
	if l.Level <= WARN {
		l.logOutput("[WARN]", message, v...)
	}
}

// Log error messages
func (l *Logger) Error(message string, v ...any) {
	if l.Level <= ERROR {
		l.logOutput("[ERROR]", message, v...)
	}
}

// SetLogLevel sets the log level of the logger
// Valid log levels are: debug, info, warn, error
func (l *Logger) SetLogLevel(logLevel string) error {
	logger := strings.ToUpper(logLevel)

	switch logger {
	case "DEBUG":
		l.Level = DEBUG
	case "INFO":
		l.Level = INFO
	case "WARN":
		l.Level = WARN
	case "ERROR":
		l.Level = ERROR
	default:
		return fmt.Errorf("invalid log level. Valid levels: debug, info, warn, error")
	}

	return nil
}

// SetOutputDestination sets the output destination of the logger
// Valid destinations are: stdout, stderr, or a file path
func (l *Logger) SetOutputDestination(destination string) error {
	if destination == "" {
		return fmt.Errorf("invalid log destination. output destination cannot be empty")
	}

	dest := strings.ToLower(destination)

	switch dest {
	case "stdout":
		l.Logger = log.New(os.Stdout, "", log.LstdFlags)
	case "stderr":
		l.Logger = log.New(os.Stderr, "", log.LstdFlags)
	default:
		f, err := os.OpenFile(destination, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		l.Logger = log.New(f, "", log.LstdFlags)
	}

	return nil
}
