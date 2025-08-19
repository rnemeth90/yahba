package config

import (
	"net/url"
	"testing"

	"github.com/rnemeth90/yahba/internal/logger"
)

func TestSetupProxy(t *testing.T) {
	want, _ := url.Parse("http://www.google.com")
	config := &Config{
		Proxy:  "http://www.google.com",
		Logger: logger.New("error", "stdout", false),
	}

	proxy, err := config.SetupProxy()
	if err != nil {
		t.Fatalf("SetupProxy() error = %v, wantErr %v", err, nil)
	}

	if proxy.String() != want.String() {
		t.Errorf("SetupProxy() got = %v, want %v", proxy.String(), want.String())
	}
}

func TestSkipNameResolution(t *testing.T) {

}

func TestSetupCustomResolver(t *testing.T) {

}
