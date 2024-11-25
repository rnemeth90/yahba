package config

import (
	"net"
	"net/url"
	"strings"

	"github.com/rnemeth90/yahba/internal/logger"
	"github.com/rnemeth90/yahba/internal/util"
)

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
}

var validHTTPMethods = []string{"GET", "HEAD", "PUT", "POST"}

func (config *Config) Validate() error {
	if config.URL == "" {
		return ErrMissingHost
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
