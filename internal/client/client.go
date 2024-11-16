package client

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rnemeth90/yahba/internal/config"
)

func NewClient(cfg config.Config) (*http.Client, error) {
	cfg.Logger.Debug("Initializing HTTP client")

	var proxyURL *url.URL
	var err error

	if cfg.Proxy != "" {
		cfg.Logger.Debug("Configuring proxy: %s", cfg.Proxy)
		proxyURL, err = url.Parse(cfg.Proxy)
		if err != nil {
			cfg.Logger.Error("Invalid proxy URL: %v", err)
			return nil, err
		}

		if cfg.ProxyUser != "" && cfg.ProxyPassword != "" {
			cfg.Logger.Debug("Configuring proxy authentication")
			proxyURL.User = url.UserPassword(cfg.ProxyUser, cfg.ProxyPassword)
		}
	}

	t := &http.Transport{
		DisableKeepAlives:  cfg.KeepAlive,
		DisableCompression: cfg.Compression,
		TLSClientConfig:    &tls.Config{InsecureSkipVerify: cfg.Insecure},
	}

	if cfg.Insecure {
		cfg.Logger.Warn("InsecureSkipVerify is enabled; TLS certificate verification is disabled")
	}

	if cfg.SkipDNS {
		cfg.Logger.Debug("Configuring DNS skipping for host: %s", cfg.Host)
		t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			_, port, err := net.SplitHostPort(addr)
			if err != nil {
				if strings.HasPrefix(cfg.Host, "https://") {
					port = "443"
				} else {
					port = "80"
				}
			}

			cfg.Logger.Debug("Bypassing DNS resolution for host: %s:%s", cfg.Host, port)
			return net.Dial(network, net.JoinHostPort(cfg.Host, port))
		}
	}

	if cfg.Resolver != "" {
		cfg.Logger.Debug("Configuring custom DNS resolver: %s", cfg.Resolver)
		t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{
				Resolver: &net.Resolver{
					PreferGo: true,
					Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
						cfg.Logger.Debug("Using custom resolver for address: %s", address)
						d := net.Dialer{
							Timeout: 500 * time.Millisecond,
						}

						return d.DialContext(ctx, "udp", cfg.Resolver)
					},
				},
			}

			return dialer.DialContext(ctx, network, addr)
		}
	}

	if proxyURL != nil {
		cfg.Logger.Debug("Setting proxy for HTTP transport")
		t.Proxy = http.ProxyURL(proxyURL)
	}

	client := &http.Client{
		Transport: t,
		Timeout:   time.Second * time.Duration(cfg.Timeout),
	}
	cfg.Logger.Debug("HTTP client successfully initialized with timeout: %d seconds", cfg.Timeout)

	return client, nil
}
