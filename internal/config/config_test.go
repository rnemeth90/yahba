package config

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/rnemeth90/yahba/internal/logger"
	"github.com/rnemeth90/yahba/internal/netutil"
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

func TestSkipNameResolution_MocksDialer(t *testing.T) {
	tr := &http.Transport{}
	cfg := &Config{
		URL:    "127.0.0.1",
		Logger: logger.New("debug", "stdout", false),
	}

	mock := netutil.
	mockDialer := &netutil.MockDialer{}
	cfg.SkipNameResolution(tr, mockDialer)

	client := &http.Client{
		Transport: tr,
		Timeout:   2 * time.Second,
	}

	_, err := client.Get("http://127.0.0.1:9999")
	if err != nil {
		// It's okay if this fails to connect — we care about whether Dial was called
	}

	if !mockDialer.Called {
		t.Fatal("expected DialContext to be called")
	}
	if mockDialer.AddrUsed != "127.0.0.1:9999" {
		t.Errorf("unexpected dial address: got %s", mockDialer.AddrUsed)
	}
}

func TestSkipNameResolution(t *testing.T) {

}

func TestSetupCustomResolver(t *testing.T) {

}
