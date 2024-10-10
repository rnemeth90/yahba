package config

import "slices"

type Config struct {
	URL           string
	Concurrency   int
	Requests      int
	Method        string
	Headers       string
	Payload       string
	Timeout       int
	RPS           int
	Insecure      bool
	Resolvers     string
	KeepAlive     bool
	Cookies       string
	HTTP2         bool
	HTTP3         bool
	Verbose       bool
	OutputFormat  string
	Compression   bool
	Proxy         string
	ProxyUser     string
	ProxyPassword string
}

var validOutputFormats = []string{"json", "yaml", "raw"}
var validHTTPMethods = []string{"GET", "HEAD", "PUT", "POST"}

func (config *Config) Validate() error {
	if config.URL == "" {
		return ErrMissingURL
	}

	if config.Method != "GET" && config.Method != "POST" && config.Method != "PUT" && config.Method != "DELETE" {
		return ErrInvalidMethod
	}

	if (config.Method == "POST" || config.Method == "PUT") && config.Payload == "" {
		return ErrMissingPayload
	}

	if config.Concurrency <= 0 {
		return ErrInvalidConcurrency
	}

	if config.Requests <= 0 {
		return ErrInvalidRequests
	}

	if config.Timeout <= 0 {
		return ErrInvalidTimeout
	}

	// check proxy config

	if !slices.Contains(validOutputFormats, config.OutputFormat) {
		return ErrInvalidOutputFormat
	}

	return nil
}
