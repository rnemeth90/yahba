package web

import (
	"errors"
	"fmt"
	"strings"

	"github.com/rnemeth90/yahba/internal/config"
	"github.com/rnemeth90/yahba/internal/logger"
	"github.com/rnemeth90/yahba/internal/util"
)

// maxRequests bounds the buffered result channels allocated per run.
const maxRequests = 1_000_000

// RunRequest is the subset of run options exposed in the web form. Proxy
// credentials are intentionally excluded because the config is stored in history.
type RunRequest struct {
	URL             string `json:"url"`
	Method          string `json:"method"`
	Requests        int    `json:"requests"`
	RPS             int    `json:"rps"`
	Workers         int    `json:"workers"`
	Timeout         int    `json:"timeout"`
	Headers         string `json:"headers"`
	Body            string `json:"body"`
	Insecure        bool   `json:"insecure"`
	KeepAlive       bool   `json:"keep_alive"`
	HTTP2           bool   `json:"http2"`
	Compression     bool   `json:"compression"`
	RandomUserAgent bool   `json:"random_user_agent"`
}

// ValidationError names the offending form field so the UI can highlight it.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

var fieldByError = map[error]string{
	config.ErrMissingHost:             "url",
	config.ErrInvalidProtocolScheme:   "url",
	config.ErrInvalidHost:             "url",
	config.ErrInvalidIPAddressForHost: "url",
	config.ErrInvalidMethod:           "method",
	config.ErrMissingBody:             "body",
	config.ErrInvalidRequests:         "requests",
	config.ErrInvalidTimeout:          "timeout",
	config.ErrInvalidRPS:              "rps",
	config.ErrInvalidHeaders:          "headers",
	config.ErrInvalidHTTPConfig:       "http2",
}

func fieldFor(err error) string {
	for target, field := range fieldByError {
		if errors.Is(err, target) {
			return field
		}
	}
	return ""
}

func (r RunRequest) ToConfig() (config.Config, error) {
	if r.Requests > maxRequests {
		return config.Config{}, &ValidationError{"requests", fmt.Sprintf("requests must be at most %d", maxRequests)}
	}
	if r.Workers <= 0 {
		return config.Config{}, &ValidationError{"workers", "workers must be greater than 0"}
	}

	method := strings.ToUpper(strings.TrimSpace(r.Method))
	if method == "" {
		method = "GET"
	}

	cfg := config.Config{
		URL:             strings.TrimSpace(r.URL),
		Method:          method,
		Requests:        r.Requests,
		RPS:             r.RPS,
		Workers:         r.Workers,
		Timeout:         r.Timeout,
		Headers:         r.Headers,
		Body:            r.Body,
		Insecure:        r.Insecure,
		KeepAlive:       r.KeepAlive,
		HTTP2:           r.HTTP2,
		Compression:     r.Compression,
		RandomUserAgent: r.RandomUserAgent,
		LogLevel:        "error",
		OutputFormat:    "raw",
		OutputFile:      "stdout",
		Silent:          true,
		Logger:          logger.New("error", "stdout", true),
	}

	if err := cfg.Validate(); err != nil {
		return config.Config{}, &ValidationError{Field: fieldFor(err), Message: err.Error()}
	}
	if cfg.Headers != "" {
		parsed, err := util.ParseHeaders(cfg.Headers)
		if err != nil {
			return config.Config{}, &ValidationError{"headers", err.Error()}
		}
		cfg.ParsedHeaders = parsed
	}
	return cfg, nil
}
