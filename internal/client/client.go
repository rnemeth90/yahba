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

	if cfg.SkipDNS {
		t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			_, port, err := net.SplitHostPort(addr)
			if err != nil {
				if strings.HasPrefix(cfg.Host, "https://") {
					port = "443"
				} else {
					port = "80"
				}
			}

			return net.Dial(network, net.JoinHostPort(cfg.Host, port))
		}
	}

	if cfg.Resolver != "" {
		t.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
			dialer := &net.Dialer{
				Resolver: &net.Resolver{
					PreferGo: true,
					Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
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
		t.Proxy = http.ProxyURL(proxyURL)
	}

	client := &http.Client{
		Transport: t,
		Timeout:   time.Second * time.Duration(cfg.Timeout),
	}

	return client, nil
}
