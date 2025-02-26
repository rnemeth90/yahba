package config

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rnemeth90/yahba/internal/logger"
	"github.com/rnemeth90/yahba/internal/util"
)

// Config holds the configuration for the load test
type Config struct {
	URL           string
	Method        string
	Headers       string
	Body          string
	Timeout       int
	RPS           int
	Requests      int
	Insecure      bool
	Resolver      string
	KeepAlive     bool
	HTTP2         bool
	HTTP3         bool
	LogLevel      string
	Logger        *logger.Logger
	RawOutput     bool
	Compression   bool
	Proxy         string
	ProxyUser     string
	ProxyPassword string
	ParsedHeaders []util.Header
	Sleep         int
	SkipDNS       bool
	OutputFile    string
	OutputFormat  string
	FileName      string
	Silent        bool
	Server        bool
}

var validHTTPMethods = []string{"GET", "HEAD", "PUT", "POST"}

// This monstrosity validates your config :)
func (config *Config) Validate() error {
	if config.URL == "" {
		return ErrMissingHost
	}

	if config.OutputFormat == "file" && config.FileName == "" {
		return ErrInvalidLogFilePath
	}

	ipAddy := net.ParseIP(config.URL)
	if config.SkipDNS && ipAddy == nil {
		return ErrInvalidIPAddressForHost
	}

	if !strings.HasPrefix(config.URL, "http") {
		return ErrInvalidProtocolScheme
	}

	u, err := url.Parse(config.URL)
	if err != nil {
		return ErrInvalidHost
	}

	if u.Scheme == "https" && config.Insecure {
		return ErrInvalidProtocolScheme
	}

	if config.Method != "GET" && config.Method != "POST" && config.Method != "PUT" && config.Method != "DELETE" {
		return ErrInvalidMethod
	}

	if (config.Method == "POST" || config.Method == "PUT") && config.Body == "" {
		return ErrMissingBody
	}

	if config.Requests <= 0 {
		return ErrInvalidRequests
	}

	if config.Timeout <= 0 {
		return ErrInvalidTimeout
	}

	if config.Proxy != "" {
		if _, err := url.Parse(config.Proxy); err != nil {
			return ErrInvalidProxy
		}
		if (config.ProxyUser == "" && config.ProxyPassword != "") || (config.ProxyUser != "" && config.ProxyPassword == "") {
			return ErrInvalidProxyAuth
		}
	}

	if config.SkipDNS && config.Resolver != "" {
		return ErrConflictingDNSOptions
	}

	if (config.Method == "POST" || config.Method == "PUT") && config.Body == "" {
		return ErrMissingBody
	}

	if config.Headers != "" {
		if _, err := util.ParseHeaders(config.Headers); err != nil {
			return ErrInvalidHeaders
		}
	}

	if config.Resolver != "" {
		if _, _, err := net.SplitHostPort(config.Resolver); err != nil {
			return ErrInvalidResolvers
		}
	}

	if config.Timeout <= 0 {
		return ErrInvalidTimeout
	}

	if config.RPS <= 0 {
		return ErrInvalidRPS
	}

	if config.Requests <= 0 {
		return ErrInvalidRequests
	}

	if config.HTTP2 && config.HTTP3 {
		return ErrInvalidHTTPConfig
	}

	return nil
}

// SetupProxy configures the proxy settings for the client
func (c *Config) SetupProxy() (*url.URL, error) {
	c.Logger.Debug("Configuring proxy: %s", c.Proxy)
	proxyURL, err := url.Parse(c.Proxy)
	if err != nil {
		c.Logger.Error("Invalid proxy URL: %v", err)
		return nil, err
	}

	if c.ProxyUser != "" && c.ProxyPassword != "" {
		c.Logger.Debug("Configuring proxy authentication")
		proxyURL.User = url.UserPassword(c.ProxyUser, c.ProxyPassword)
	}

	return proxyURL, nil
}

// DNSBypass is a function type that can be replaced in tests
type DNSBypassFunc func(ctx context.Context, network, addr string) (net.Conn, error)

// SkipNameResolution bypasses DNS resolution by replacing the default DialContext function
func (c *Config) SkipNameResolution(tr *http.Transport, dialFunc DNSBypassFunc) {
	if dialFunc == nil {
		dialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
			_, port, err := net.SplitHostPort(addr)
			if err != nil {
				if strings.HasPrefix(c.URL, "https://") {
					port = "443"
				} else {
					port = "80"
				}
			}
			return net.Dial(network, net.JoinHostPort(c.URL, port))
		}
	}

	tr.DialContext = dialFunc
}

// SetupCustomResolver configures a custom DNS resolver for the client
func (c *Config) SetupCustomResolver(tr *http.Transport) {
	c.Logger.Debug("Configuring custom DNS resolver: %s", c.Resolver)
	tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		dialer := &net.Dialer{
			Resolver: &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					c.Logger.Debug("Using custom resolver %s to resolve: %s", c.Resolver, c.URL)
					d := net.Dialer{
						Timeout: time.Duration(c.Timeout) * time.Second,
					}

					return d.DialContext(ctx, "udp", c.Resolver)
				},
			},
		}
		return dialer.DialContext(ctx, network, addr)
	}
}
