package client

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"

	"github.com/rnemeth90/yahba/internal/config"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// Create the HTTP Client
func NewClient(cfg config.Config) (*http.Client, error) {
	var proxyURL *url.URL
	var err error

	// Parse Proxy if provided
	if cfg.Proxy != "" {
		proxyURL, err = url.Parse(cfg.Proxy)
		if err != nil {
			return nil, err
		}

		if cfg.ProxyUser != "" && cfg.ProxyPassword != "" {
			proxyURL.User = url.UserPassword(cfg.ProxyUser, cfg.ProxyPassword)
		}
	}

	// HTTP Transport
	t := &http.Transport{
		DisableKeepAlives:  cfg.KeepAlive,
		DisableCompression: cfg.Compression,
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: cfg.Insecure},
	}

	// Assign Proxy to Transport if provided
	if proxyURL != nil {
		t.Proxy = http.ProxyURL(proxyURL)
	}

	// HTTP/2 Support
	if cfg.HTTP2 {
		http2.ConfigureTransport(t)
	}

	// Create HTTP Client with Timeout
	client := &http.Client{
		Transport: t,
		Timeout:   time.Second * time.Duration(cfg.Timeout),
	}

	// HTTP/3 Support
	if cfg.HTTP3 {
		// Configure H2C (HTTP/2 Cleartext) for HTTP/3 fallback
		client.Transport = h2c.NewTransport(client.Transport)
	}

	return client, nil
}
