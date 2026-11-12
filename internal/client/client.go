package client

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/rnemeth90/yahba/internal/config"
	"golang.org/x/net/http2"
)

const http3KeepAlivePeriod = 15 * time.Second

func NewClient(cfg config.Config) (*http.Client, error) {
	cfg.Logger.Debug("Initializing HTTP client")

	var proxyURL *url.URL
	var err error
	if cfg.Proxy != "" {
		proxyURL, err = cfg.SetupProxy()
		if err != nil {
			return nil, err
		}
	}

	var transport http.RoundTripper
	if cfg.HTTP3 {
		var keepAlive time.Duration
		if cfg.KeepAlive {
			keepAlive = http3KeepAlivePeriod
		}
		transport = &http3.Transport{
			DisableCompression: !cfg.Compression,
			TLSClientConfig:    &tls.Config{InsecureSkipVerify: cfg.Insecure},
			QUICConfig: &quic.Config{
				MaxIdleTimeout:  time.Duration(cfg.Timeout) * time.Second,
				KeepAlivePeriod: keepAlive,
			},
		}
	} else if cfg.HTTP2 {
		if cfg.ReuseConnections {
			transport = &http2.Transport{
				DisableCompression: !cfg.Compression,
				TLSClientConfig:    &tls.Config{InsecureSkipVerify: cfg.Insecure},
				IdleConnTimeout:    time.Duration(cfg.Timeout) * time.Second,
			}
		} else {
			transport = &http2.Transport{
				DisableCompression: !cfg.Compression,
				TLSClientConfig:    &tls.Config{InsecureSkipVerify: cfg.Insecure},
				IdleConnTimeout:    time.Duration(cfg.Timeout) * time.Second,
				DialTLS: func(network, addr string, cfg *tls.Config) (net.Conn, error) {
					tcpConn, err := net.Dial(network, addr)
					if err != nil {
						return nil, err
					}
					return tls.Client(tcpConn, cfg), nil
				},
			}
		}
	} else {
		tr := &http.Transport{
			MaxIdleConns:        1000,
			MaxIdleConnsPerHost: 1000,
			IdleConnTimeout:     time.Duration(cfg.Timeout) * time.Second,
			DisableKeepAlives:   !cfg.KeepAlive,
			DisableCompression:  !cfg.Compression,
			ForceAttemptHTTP2:   false,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: cfg.Insecure},
			Proxy:               http.ProxyURL(proxyURL),
		}

		// skipping DNS resolution only works with HTTP 1.1, not HTTP 2.0
		if cfg.SkipDNS {
			cfg.SkipNameResolution(tr)
		}

		if cfg.Resolver != "" {
			cfg.SetupCustomResolver(tr)
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

// Close releases the client's connections. An HTTP/3 transport also owns a UDP
// socket that only Close frees, so callers should defer this rather than
// CloseIdleConnections.
func Close(c *http.Client) {
	if closer, ok := c.Transport.(io.Closer); ok {
		closer.Close()
		return
	}
	c.CloseIdleConnections()
}
