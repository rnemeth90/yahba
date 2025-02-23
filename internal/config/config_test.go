package config

import "testing"

func TestSetupProxy(t *testing.T) {
	// Create a new Config struct
	cfg := Config{
		Proxy: "http://proxy:8080",
	}

	// Call the SetupProxy method
	_, err := cfg.SetupProxy()

	// Check if the error is nil
	if err != nil {
		t.Errorf("Expected nil, got %v", err)
	}
}

func TestSetupProxyInvalidURL(t *testing.T) {
	// Create a new Config struct
	cfg := Config{
		Proxy: "invalid",
	}

	// Call the SetupProxy method
	_, err := cfg.SetupProxy()

	// Check if the error is not nil
	if err == nil {
		t.Error("Expected error, got nil")
	}
}
