package main

import (
	"fmt"
	"os"

	"github.com/rnemeth90/yahba/internal/config"
	"github.com/rnemeth90/yahba/internal/logger"
	"github.com/rnemeth90/yahba/internal/report"
	"github.com/rnemeth90/yahba/internal/stressor"
	"github.com/rnemeth90/yahba/internal/util"
	"github.com/spf13/pflag"
)

var c config.Config

func init() {
	pflag.StringVarP(&c.Host, "host", "h", "", "The target URL to stress test. This should include the protocol (e.g., http:// or https://)")
	pflag.IntVarP(&c.Requests, "requests", "r", 4, "The total number of requests to send during the test")
	pflag.StringVarP(&c.Method, "method", "m", "GET", "The HTTP method to use for each request (e.g., GET, POST, PUT)")
	pflag.StringVarP(&c.Headers, "headers", "H", "", "Custom headers to add to each request, formatted as 'Key1:Value1,Key2:Value2'")
	pflag.StringVarP(&c.Body, "body", "b", "", "The request body to include with POST and PUT methods")
	pflag.IntVarP(&c.Timeout, "timeout", "t", 10, "The timeout in seconds for each request, after which it will be considered failed")
	pflag.IntVar(&c.RPS, "rps", 1, "The number of requests per second (RPS) to send during the test")
	pflag.BoolVarP(&c.Insecure, "insecure", "i", false, "If set, disables SSL/TLS certificate verification for HTTPS requests")
	pflag.StringVar(&c.Resolver, "resolver", "", "A custom DNS resolver to use, specified as 'IP:Port'")
	pflag.StringVarP(&c.Proxy, "proxy", "P", "", "The proxy server to route requests through, specified as 'IP:Port'")
	pflag.BoolVarP(&c.KeepAlive, "keep-alive", "k", false, "Enable HTTP keep-alive, allowing TCP connections to remain open for multiple requests")
	pflag.BoolVar(&c.HTTP2, "http2", true, "Enable HTTP/2 support for requests (default: true)")
	pflag.StringVarP(&c.LogLevel, "log-level", "l", "info", "The logging level to use (options: debug, info, warn, error)")
	pflag.BoolVar(&c.Compression, "compression", false, "Enable HTTP compression for requests (e.g., gzip)")
	pflag.StringVar(&c.ProxyUser, "proxy-user", "", "Username for proxy authentication")
	pflag.StringVar(&c.ProxyPassword, "proxy-password", "", "Password for proxy authentication")
	pflag.IntVarP(&c.Sleep, "sleep", "s", 1, "Sleep time in seconds between requests in a single worker (throttles requests)")
	pflag.BoolVar(&c.SkipDNS, "skip-dns", false, "If set, skips DNS resolution and uses a direct IP address")
	pflag.StringVar(&c.OutputFormat, "output-format", "raw", "Output format: json, yaml, or raw")
	pflag.StringVar(&c.OutputFile, "out", "stdout", "File path to write results to; defaults to stdout")
}

func main() {
	l := logger.NewLogger(c.LogLevel, "stdout")
	l.Debug("parsing flags")
	pflag.Parse()

	if err := run(c, l); err != nil {
		l.Error("error: %v\n", err)
		os.Exit(1)
	}
}

func run(c config.Config, l *logger.Logger) error {
	if err := c.Validate(); err != nil {
		return err
	}

	// Parse headers once
	if c.Headers != "" {
		parsedHeaders, err := util.ParseHeaders(c.Headers)
		if err != nil {
			return fmt.Errorf("error parsing headers: %w", err)
		}
		c.ParsedHeaders = parsedHeaders
	}

	// Create jobs based on the number of requests
	jobs := make([]stressor.Job, c.Requests)
	for i := 0; i < c.Requests; i++ {
		jobs[i] = stressor.Job{Host: c.Host, Method: c.Method, Body: c.Body}
	}

	reportChan := make(chan report.Report, c.Requests)
	go func() {
		stressor.WorkerPool(c, jobs, reportChan)
	}()

	// Generate report based on output format
	var reportOutput string
	var err error
	switch c.OutputFormat {
	case "json":
		reportOutput, err = report.ParseJSON(reportChan)
	case "yaml":
		reportOutput, err = report.ParseYAML(reportChan)
	default:
		reportOutput, err = report.ParseRaw(reportChan)
	}
	if err != nil {
		l.Error("error generating report: %v", err)
		return err
	}

	fmt.Println(reportOutput)
	return nil
}
