package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/rnemeth90/yahba/internal/config"
	"github.com/rnemeth90/yahba/internal/logger"
	"github.com/rnemeth90/yahba/internal/stressor"
	"github.com/rnemeth90/yahba/internal/util"
	"github.com/spf13/pflag"
)

var c config.Config

func init() {
	pflag.StringVarP(&c.Host, "host", "h", "", "specify the URL to stress test")
	pflag.IntVarP(&c.Requests, "requests", "r", 4, "the total number of requests that should be sent")
	pflag.StringVarP(&c.Method, "method", "m", "GET", "which HTTP method to use (GET, POST, etc.)")
	pflag.StringVarP(&c.Headers, "headers", "H", "", "allows adding custom headers to the requests")
	pflag.StringVarP(&c.Body, "body", "b", "", "for POST and PUT requests, users can define the request body")
	pflag.IntVarP(&c.Timeout, "timeout", "t", 10, "the timeout for each request in seconds")
	pflag.IntVar(&c.RPS, "rps", 1, "requests per second")
	pflag.BoolVarP(&c.Insecure, "insecure", "i", false, "disable SSL/TLS certificate verification")
	pflag.StringVar(&c.Resolver, "resolver", "", "custom DNS resolver")
	pflag.StringVarP(&c.Proxy, "proxy", "P", "", "proxy server address")
	pflag.BoolVarP(&c.KeepAlive, "keep-alive", "k", false, "use keep-alives")
	pflag.StringVarP(&c.Cookies, "cookies", "C", "", "set cookies")
	pflag.BoolVar(&c.HTTP2, "http2", true, "use HTTP/2")
	pflag.StringVarP(&c.LogLevel, "log-level", "l", "info", "logging level. debug, info, warn, error")
	pflag.StringVarP(&c.OutputFormat, "output", "o", "json", "output format (json/yaml/raw)")
	pflag.BoolVar(&c.Compression, "compression", false, "use compression")
	pflag.StringVar(&c.ProxyUser, "proxy-user", "", "proxy user name")
	pflag.StringVar(&c.ProxyPassword, "proxy-password", "", "proxy password")
	pflag.IntVarP(&c.Sleep, "sleep", "s", 1, "sleep seconds")
	pflag.BoolVar(&c.SkipDNS, "skip-dns", false, "skip dns resolution")
}

func main() {
	l := logger.NewLogger("debug", "stdout")
	l.Debug("parsing flags")
	pflag.Parse()

	if err := run(c); err != nil {
		fmt.Fprintln(os.Stdout, err)
		os.Exit(1)
	}
}

func run(c config.Config) error {
	if err := c.Validate(); err != nil {
		return err
	}

	// Parse headers here, so we only parse them once
	heads := []util.Header{}
	if c.Headers != "" {
		if strings.Contains(c.Headers, ",") {
			headers, err := util.ParseHeaders(c.Headers)
			if err != nil {
				return err
			}

			for _, h := range headers {
				heads = append(heads, h)
			}
		} else {
			header, err := util.ParseHeader(c.Headers)
			if err != nil {
				return err
			}

			heads = append(heads, header)
		}
	}
	c.ParsedHeaders = heads

	jobs := make([]stressor.Job, c.Requests)
	for i := 0; i < c.Requests; i++ {
		jobs[i] = stressor.Job{
			Host:   c.Host,
			Method: c.Method,
			Body:   c.Body,
		}
	}

	stressor.WorkerPool(c, jobs)

	return nil
}
