package config

import (
	"net/url"
	"strings"

	"github.com/rnemeth90/yahba/internal/util"
)

type Config struct {
	Host          string
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
	JSONOutput    bool
	YAMLOutput    bool
	RawOutput     bool
	Compression   bool
	Proxy         string
	ProxyUser     string
	ProxyPassword string
	ParsedHeaders []util.Header
	Sleep         int
	SkipDNS       bool
}

var validHTTPMethods = []string{"GET", "HEAD", "PUT", "POST"}

func (config *Config) Validate() error {
	if config.Host == "" {
		return ErrMissingHost
	}

	if !strings.HasPrefix(config.Host, "http") {
		return ErrInvalidProtocolScheme
	}

	if config.RawOutput && config.YAMLOutput && config.JSONOutput {
		return ErrInvalidOutputFormat
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

	return nil
}
