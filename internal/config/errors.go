package config

import "errors"

var (
	// General Errors
	ErrMissingHost           = errors.New("URL is required, please specify it using --url or -u")
	ErrInvalidMethod         = errors.New("Invalid HTTP method. Supported methods are GET, POST, PUT, DELETE, etc.")
	ErrMissingBody           = errors.New("Payload is required when using POST or PUT methods")
	ErrInvalidConcurrency    = errors.New("Concurrency must be greater than 0")
	ErrInvalidRequests       = errors.New("Requests must be greater than 0")
	ErrInvalidTimeout        = errors.New("Timeout must be greater than 0")
	ErrInvalidRPS            = errors.New("Requests per second (RPS) must be greater than 0")
	ErrInvalidOutputFormat   = errors.New("Invalid output format. Supported formats are json, yaml, raw")
	ErrInvalidProxy          = errors.New("Invalid proxy server address")
	ErrInvalidResolvers      = errors.New("Invalid DNS resolvers format. Expected a comma-separated list")
	ErrInvalidHeaders        = errors.New("Invalid headers format. Expected a semi-colon separated list of 'Key: Value' pairs")
	ErrInvalidCookies        = errors.New("Invalid cookies format. Expected a semi-colon separated list of 'Key=Value' pairs")
	ErrHTTP2Disabled         = errors.New("HTTP/2 is disabled")
	ErrHTTP3Disabled         = errors.New("HTTP/3 is disabled")
	ErrInvalidHTTPConfig     = errors.New("Invalid HTTP config. Only one value can be supplied")
	ErrInvalidHost           = errors.New("invalid host")
	ErrInvalidProtocolScheme = errors.New("invalid protocol scheme")
)
