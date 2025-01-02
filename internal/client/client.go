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
	"golang.org/x/net/http2"
)

func NewClient(cfg config.Config) (*http.Client, error) {
	cfg.Logger.Debug("Initializing HTTP client")

	var proxyURL *url.URL
	var err error

	// proxy
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

	var transport http.RoundTripper
	if cfg.HTTP2 {
		transport = &http2.Transport{
			DisableCompression: cfg.Compression,
			TLSClientConfig:    &tls.Config{InsecureSkipVerify: cfg.Insecure},
		}
	} else {
		tr := &http.Transport{
			DisableKeepAlives:  cfg.KeepAlive,
			DisableCompression: cfg.Compression,
			ForceAttemptHTTP2:  false,
			TLSClientConfig:    &tls.Config{InsecureSkipVerify: cfg.Insecure},
			Proxy:              http.ProxyURL(proxyURL),
		}

		// skipping DNS resolution only works with HTTP 1.1, not HTTP 2.0
		if cfg.SkipDNS {
			cfg.Logger.Debug("Configuring DNS skipping for host: %s", cfg.URL)
			tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				_, port, err := net.SplitHostPort(addr)
				if err != nil {
					if strings.HasPrefix(cfg.URL, "https://") {
						port = "443"
					} else {
						port = "80"
					}
				}

				cfg.Logger.Debug("Bypassing DNS resolution for host: %s:%s", cfg.URL, port)
				return net.Dial(network, net.JoinHostPort(cfg.URL, port))
			}
		}

		if cfg.Resolver != "" {
			cfg.Logger.Debug("Configuring custom DNS resolver: %s", cfg.Resolver)
			tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				dialer := &net.Dialer{
					Resolver: &net.Resolver{
						PreferGo: true,
						Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
							cfg.Logger.Debug("Using custom resolver %s to resolve: %s", cfg.Resolver, cfg.URL)
							d := net.Dialer{
								Timeout: time.Duration(cfg.Timeout) * time.Second,
							}

							return d.DialContext(ctx, "udp", cfg.Resolver)
						},
					},
				}
				return dialer.DialContext(ctx, network, addr)
			}
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
