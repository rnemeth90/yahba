package web

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validRequest() RunRequest {
	return RunRequest{URL: "http://127.0.0.1:1", Method: "get", Requests: 10, RPS: 5, Workers: 2, Timeout: 5}
}

func TestToConfigValid(t *testing.T) {
	cfg, err := validRequest().ToConfig()
	require.NoError(t, err)
	assert.Equal(t, "GET", cfg.Method)
	assert.Equal(t, 10, cfg.Requests)
	assert.NotNil(t, cfg.Logger)
}

func TestToConfigParsesHeaders(t *testing.T) {
	req := validRequest()
	req.Headers = "X-Test:abc"
	cfg, err := req.ToConfig()
	require.NoError(t, err)
	require.Len(t, cfg.ParsedHeaders, 1)
	assert.Equal(t, "X-Test", cfg.ParsedHeaders[0].Key)
}

func TestToConfigFieldErrors(t *testing.T) {
	cases := map[string]func(*RunRequest){
		"url":      func(r *RunRequest) { r.URL = "" },
		"method":   func(r *RunRequest) { r.Method = "BREW" },
		"requests": func(r *RunRequest) { r.Requests = 0 },
		"rps":      func(r *RunRequest) { r.RPS = 0 },
		"timeout":  func(r *RunRequest) { r.Timeout = 0 },
		"workers":  func(r *RunRequest) { r.Workers = 0 },
		"body":     func(r *RunRequest) { r.Method = "POST" },
	}
	for field, mutate := range cases {
		t.Run(field, func(t *testing.T) {
			req := validRequest()
			mutate(&req)
			_, err := req.ToConfig()
			var ve *ValidationError
			require.True(t, errors.As(err, &ve), "got %v", err)
			assert.Equal(t, field, ve.Field)
		})
	}
}

func TestToConfigCapsRequests(t *testing.T) {
	req := validRequest()
	req.Requests = maxRequests + 1
	_, err := req.ToConfig()
	var ve *ValidationError
	require.True(t, errors.As(err, &ve))
	assert.Equal(t, "requests", ve.Field)
}

func TestToConfigCapsRPS(t *testing.T) {
	req := validRequest()
	req.RPS = maxRPS + 1
	_, err := req.ToConfig()
	var ve *ValidationError
	require.True(t, errors.As(err, &ve))
	assert.Equal(t, "rps", ve.Field)
}

func TestToConfigCapsTimeout(t *testing.T) {
	req := validRequest()
	req.Timeout = maxTimeoutSeconds + 1
	_, err := req.ToConfig()
	var ve *ValidationError
	require.True(t, errors.As(err, &ve))
	assert.Equal(t, "timeout", ve.Field)
}
