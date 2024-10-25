package config

import (
	"net/url"
	"slices"
	"strings"

	"github.com/rnemeth90/yahba/internal/util"
)

type Config struct {
	Host          string
	Concurrency   int
	Requests      int
	Method        string
	Headers       string
	Body          string
	Timeout       int
	RPS           int
	Insecure      bool
	Resolver      string
	KeepAlive     bool
	HTTP2         bool
	HTTP3         bool
	LogLevel      string
	OutputFormat  string
	Compression   bool
	Proxy         string
	ProxyUser     string
	ProxyPassword string
	ParsedHeaders []util.Header
	Sleep         int
	SkipDNS       bool
}

var validOutputFormats = []string{"json", "yaml", "raw"}
var validHTTPMethods = []string{"GET", "HEAD", "PUT", "POST"}

func (config *Config) Validate() error {
	if config.Host == "" {
		return ErrMissingHost
	}

	if !strings.HasPrefix(config.Host, "http") {
		return ErrInvalidProtocolScheme
	}

	u, err := url.Parse(config.Host)
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

	// check proxy config

	if config.HTTP2 && config.HTTP3 {
		return ErrInvalidHTTPConfig
	}

	if !slices.Contains(validOutputFormats, config.OutputFormat) {
		return ErrInvalidOutputFormat
	}

	return nil
}
