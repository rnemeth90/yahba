package logger

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	l := New("info", "stdout", false)

	if l.Level != INFO {
		t.Errorf("expected level %d, got %d", INFO, l.Level)
	}
	if l.Silent {
		t.Errorf("expected silent to be false, got true")
	}
	if l.Logger == nil {
		t.Errorf("expected logger to be initialized, got nil")
	}
}

func TestShouldLog(t *testing.T) {
	l := New("info", "stdout", false)

	tests := []struct {
		level    int
		expected bool
	}{
		{DEBUG, true},
		{INFO, true},
		{WARN, true},
		{ERROR, true},
	}

	for _, tt := range tests {
		l.Level = tt.level
		if l.shouldLog(tt.level) != tt.expected {
			t.Errorf("expected %v, got %v for level %d", tt.expected, l.shouldLog(tt.level), tt.level)
		}
	}
}

func TestLogOutput(t *testing.T) {
	var buf bytes.Buffer
	l := New("info", "stdout", false)
	l.Logger.SetOutput(&buf)

	tests := []struct {
		level   string
		message string
	}{
		{"debug", "Debug message"},
		{"info", "Info message"},
		{"warn", "Warn message"},
		{"error", "Error message"},
	}

	for _, tt := range tests {
		l.logOutput(tt.level, tt.message)
		if !strings.Contains(buf.String(), tt.message) {
			t.Errorf("expected message to be logged at %s level", tt.level)
		}
	}
}

func TestInfo(t *testing.T) {
	var buf bytes.Buffer
	l := New("info", "stdout", false)
	l.Logger.SetOutput(&buf)

	l.Info("Test message")
	if !strings.Contains(buf.String(), "[INFO] Test message") {
		t.Errorf("expected message to be logged at info level")
	}
}

func TestDebug(t *testing.T) {
	var buf bytes.Buffer
	l := New("debug", "stdout", false)
	l.Logger.SetOutput(&buf)

	l.Debug("Test message")
	if !strings.Contains(buf.String(), "[DEBUG] Test message") {
		t.Errorf("expected message to be logged at debug level")
	}
}

func TestWarn(t *testing.T) {
	var buf bytes.Buffer
	l := New("warn", "stdout", false)
	l.Logger.SetOutput(&buf)

	l.Warn("Test message")
	if !strings.Contains(buf.String(), "[WARN] Test message") {
		t.Errorf("expected message to be logged at warn level")
	}
}

func TestError(t *testing.T) {
	var buf bytes.Buffer
	l := New("error", "stdout", false)
	l.Logger.SetOutput(&buf)

	l.Error("Test message")
	if !strings.Contains(buf.String(), "[ERROR] Test message") {
		t.Errorf("expected message to be logged at error level")
	}
}

func TestSetLogLevel(t *testing.T) {
	l := New("info", "stdout", false)

	tests := []struct {
		level       string
		expectedInt int
	}{
		{"debug", DEBUG},
		{"info", INFO},
		{"warn", WARN},
		{"error", ERROR},
	}

	for _, tt := range tests {
		err := l.SetLogLevel(tt.level)
		if err != nil {
			t.Errorf("unexpected error for level %s: %v", tt.level, err)
		}
		if l.Level != tt.expectedInt {
			t.Errorf("expected level %d, got %d for input %s", tt.expectedInt, l.Level, tt.level)
		}
	}

	err := l.SetLogLevel("l33t h4x0r")
	if err == nil || !strings.Contains(err.Error(), "invalid log level") {
		t.Errorf("expected error for invalid log level, got %v", err)
	}
}

func TestLogLevelBehavior(t *testing.T) {
	tests := []struct {
		setLevel  string
		logLevel  string
		message   string
		shouldLog bool
	}{
		{"debug", "Debug", "Debug message", true},
		{"info", "Debug", "Debug message", false},
		{"info", "Info", "Info message", true},
		{"warn", "Info", "Info message", false},
		{"error", "Error", "Error message", true},
	}

	for _, tt := range tests {
		var buf bytes.Buffer
		l := New(tt.setLevel, "stdout", false)
		l.Logger.SetOutput(&buf) // Redirect output to buffer for testing

		switch tt.logLevel {
		case "Debug":
			l.Debug(tt.message)
		case "Info":
			l.Info(tt.message)
		case "Warn":
			l.Warn(tt.message)
		case "Error":
			l.Error(tt.message)
		}

		if tt.shouldLog && !strings.Contains(buf.String(), tt.message) {
			t.Errorf("expected message %q to be logged at level %s", tt.message, tt.setLevel)
		} else if !tt.shouldLog && strings.Contains(buf.String(), tt.message) {
			t.Errorf("expected message %q to not be logged at level %s", tt.message, tt.setLevel)
		}
	}
}

func TestSetOutputDestination(t *testing.T) {
	tests := []struct {
		destination string
		expected    *os.File
	}{
		{"stdout", os.Stdout},
		{"stderr", os.Stderr},
	}

	for _, tt := range tests {
		var buf bytes.Buffer
		l := New("info", tt.destination, false)
		l.Logger.SetOutput(&buf) // Redirect to buffer to verify output

		l.Info("Test message")

		if !strings.Contains(buf.String(), "Test message") {
			t.Errorf("expected message to be logged to %s, got %v", tt.destination, buf.String())
		}
	}

	l := New("info", "yahba.log", false)
	defer func() {
		if l.Logger != nil && l.Logger.Writer() != os.Stdout && l.Logger.Writer() != os.Stderr {
			if f, ok := l.Logger.Writer().(*os.File); ok {
				os.Remove(f.Name())
			}
		}
	}()
	l.Info("Test file output message")

	fileOutput := l.Logger.Writer().(*os.File)
	data, err := os.ReadFile(fileOutput.Name())
	if err != nil {
		t.Fatalf("failed to read log file: %v", err)
	}
	if !strings.Contains(string(data), "Test file output message") {
		t.Errorf("expected message to be logged to file, got %v", string(data))
	}

	err = l.SetOutputDestination("")
	if err == nil || !strings.Contains(err.Error(), "invalid log destination") {
		t.Errorf("expected error for invalid log destination, got %v", err)
	}
}
