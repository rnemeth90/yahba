package client

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"

	"github.com/rnemeth90/yahba/internal/config"
	"golang.org/x/net/http2"
)

func NewClient(cfg config.Config) (*http.Client, error) {
	cfg.Logger.Debug("Initializing HTTP client")

	var proxyURL *url.URL
	var err error
	if cfg.Proxy != "" {
		proxyURL, err = cfg.SetupProxy()
		if err != nil {
			return nil, err
		}
	}

	var transport http.RoundTripper
	if cfg.HTTP2 {
		transport = &http2.Transport{
			DisableCompression: cfg.Compression,
			TLSClientConfig:    &tls.Config{InsecureSkipVerify: cfg.Insecure},
			IdleConnTimeout:    time.Duration(cfg.Timeout) * time.Second,
		}
	} else {
		tr := &http.Transport{
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 500,
			IdleConnTimeout:     time.Duration(cfg.Timeout) * time.Second,
			DisableKeepAlives:   cfg.KeepAlive,
			DisableCompression:  cfg.Compression,
			ForceAttemptHTTP2:   false,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: cfg.Insecure},
			Proxy:               http.ProxyURL(proxyURL),
		}

		// skipping DNS resolution only works with HTTP 1.1, not HTTP 2.0
		if cfg.SkipDNS {
			cfg.SkipNameResolution(tr, nil)
		}

		if cfg.Resolver != "" {
			cfg.SetupCustomResolver(tr)
		}

		transport = tr
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(cfg.Timeout) * time.Second,
	}

	cfg.Logger.Debug("HTTP client successfully initialized with timeout: %d seconds", cfg.Timeout)
	return client, nil
}
