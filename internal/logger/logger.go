package logger

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

const (
	DEBUG = iota
	INFO
	WARN
	ERROR
)

type Logger struct {
	Level int
	*log.Logger
}

func NewLogger(level string, output string) *Logger {
	l := &Logger{}

	if err := l.SetOutputDestination(output); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set output destination: %v\n", err)
		os.Exit(1)
	}

	if err := l.SetLogLevel(level); err != nil {
		fmt.Fprintf(os.Stderr, "failed to set log level: %v\n", err)
		os.Exit(1)
	}

	return l
}

func (l *Logger) logOutput(level string, message string, v ...any) {
	// Format message and arguments
	formattedMessage := fmt.Sprintf(message, v...)
	l.Logger.Output(2, fmt.Sprintf("%s %s", level, formattedMessage))
}

func (l *Logger) Debug(message string, v ...any) {
	if l.Level <= DEBUG {
		l.logOutput("[DEBUG]", message, v...)
	}
}

func (l *Logger) Info(message string, v ...any) {
	if l.Level <= INFO {
		l.logOutput("[INFO]", message, v...)
	}
}

func (l *Logger) Warn(message string, v ...any) {
	if l.Level <= WARN {
		l.logOutput("[WARN]", message, v...)
	}
}

func (l *Logger) Error(message string, v ...any) {
	if l.Level <= ERROR {
		l.logOutput("[ERROR]", message, v...)
	}
}

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

func (l *Logger) SetOutputDestination(destination string) error {
	dest := strings.ToLower(destination)

	switch dest {
	case "stdout":
		l.Logger = log.New(os.Stdout, "", log.LstdFlags)
	case "stderr":
		l.Logger = log.New(os.Stderr, "", log.LstdFlags)
	case "file":
		dateTimeString := time.Now().Format("2006-01-02-15-04-05")
		fileName := fmt.Sprintf("%s%c%s-YAHBA.log", os.TempDir(), os.PathSeparator, dateTimeString)
		f, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return err
		}
		l.Logger = log.New(f, "", log.LstdFlags)
	default:
		return fmt.Errorf("invalid log destination. Valid destinations: stdout, stderr, file")
	}

	return nil
}
