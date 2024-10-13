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

	if cfg.Proxy != "" {
		proxyURL, err = url.Parse(cfg.Proxy)
		if err != nil {
			return nil, err
		}

		if cfg.ProxyUser != "" && cfg.ProxyPassword != "" {
			proxyURL.User = url.UserPassword(cfg.ProxyUser, cfg.ProxyPassword)
		}
	}
	t := &http.Transport{
		DisableKeepAlives:  cfg.KeepAlive,
		DisableCompression: cfg.Compression,
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: cfg.Insecure},
	}

	if proxyURL != nil {
		t.Proxy = http.ProxyURL(proxyURL)
	}

	if cfg.HTTP2 {
		http2.ConfigureTransport(t)
	}

	client := &http.Client{
		Transport: t,
		Timeout:   time.Second * time.Duration(cfg.Timeout),
	}

	
	if cfg.HTTP3 {
		client.Transport = h2c.NewTransport(client.Transport)
	}

	return client, nil
}
