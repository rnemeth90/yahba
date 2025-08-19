package client

import (
	"crypto/tls"
	"net"
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
	dialer := &net.Dialer{}
	if cfg.HTTP2 {
		tr, err := setupHTTP2Transport(cfg)
		if err != nil {
			return nil, err
		}
		transport = tr
	} else {
		tr := setupHTTPTransport(cfg, proxyURL)

		// skipping DNS resolution only works with HTTP 1.1, not HTTP 2.0
		if cfg.SkipDNS {
			cfg.SkipNameResolution(tr, dialer)
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

func setupHTTP2Transport(cfg config.Config) (*http2.Transport, error) {
	transport := &http2.Transport{}
	if cfg.ReuseConnections {
		transport = &http2.Transport{
			DisableCompression: cfg.Compression,
			TLSClientConfig:    &tls.Config{InsecureSkipVerify: cfg.Insecure},
			IdleConnTimeout:    time.Duration(cfg.Timeout) * time.Second,
		}
	} else {
		transport = &http2.Transport{
			DisableCompression: cfg.Compression,
			TLSClientConfig:    &tls.Config{InsecureSkipVerify: cfg.Insecure},
			IdleConnTimeout:    time.Duration(cfg.Timeout) * time.Second,
			DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
				tcpConn, err := net.Dial(network, addr)
				if err != nil {
					return nil, err
				}
				return tls.Client(tcpConn, cfg), nil
			},
		}
	}

	cfg.Logger.Debug("Setting up HTTP/2 transport")
	return transport, nil
}

func setupHTTPTransport(cfg config.Config, proxyURL *url.URL) *http.Transport {
	return &http.Transport{
		MaxIdleConns:        1000,
		MaxIdleConnsPerHost: 500,
		IdleConnTimeout:     time.Duration(cfg.Timeout) * time.Second,
		DisableKeepAlives:   cfg.KeepAlive,
		DisableCompression:  cfg.Compression,
		ForceAttemptHTTP2:   false,
		TLSClientConfig:     &tls.Config{InsecureSkipVerify: cfg.Insecure},
		Proxy:               http.ProxyURL(proxyURL),
	}
}
