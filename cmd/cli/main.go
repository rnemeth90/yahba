package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rnemeth90/yahba/internal/config"
	"github.com/rnemeth90/yahba/internal/logger"
	"github.com/rnemeth90/yahba/internal/report"
	"github.com/rnemeth90/yahba/internal/server"
	"github.com/rnemeth90/yahba/internal/util"
	"github.com/rnemeth90/yahba/internal/worker"
	"github.com/spf13/pflag"
)

var c config.Config
var help bool

func init() {
	pflag.StringVarP(&c.URL, "url", "u", "", "The target URL to stress test. This should include the protocol (e.g., http:// or https://)")
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
	pflag.BoolVar(&c.HTTP2, "http2", false, "Enable HTTP/2 support for requests (default: false)")
	pflag.StringVarP(&c.LogLevel, "log-level", "l", "error", "The logging level to use (options: debug, info, warn, error)")
	pflag.BoolVar(&c.Compression, "compression", false, "Enable HTTP compression for requests (e.g., gzip)")
	pflag.StringVar(&c.ProxyUser, "proxy-user", "", "Username for proxy authentication")
	pflag.StringVar(&c.ProxyPassword, "proxy-password", "", "Password for proxy authentication")
	pflag.IntVarP(&c.Sleep, "sleep", "s", 1, "Sleep time in seconds between requests in a single worker (throttles requests)")
	pflag.BoolVar(&c.SkipDNS, "skip-dns", false, "If set, skips DNS resolution and uses a direct IP address")
	pflag.StringVar(&c.OutputFormat, "output-format", "raw", "Output format: json, yaml, or raw")
	pflag.StringVar(&c.OutputFile, "out", "stdout", "File path to write results to; defaults to stdout. stdout, stderr, file")
	pflag.StringVar(&c.FileName, "filename", "", "Specify a file name when --out is set to file file")
	pflag.BoolVar(&c.Server, "server", false, "Start a test server")
	pflag.BoolVarP(&help, "help", "h", false, "help")
	pflag.Usage = usage
}

func usage() {
	fmt.Fprintf(os.Stderr, `Usage of %s:
A high-performance HTTP load testing tool.

Options:
`, os.Args[0])

	// Print all the defined flags and their default values
	pflag.PrintDefaults()

	fmt.Fprintf(os.Stderr, `
Examples:
  Basic Usage:
    %s --url=http://example.com --requests=100 --rps=10

  Test with Custom Headers:
    %s --url=https://example.com --headers="Authorization:Bearer abc123,Content-Type:application/json"

  Test with POST Method and Payload:
    %s --url=https://api.example.com --method=POST --body='{"key":"value"}' --headers="Content-Type:application/json"

  Disable SSL/TLS Verification:
    %s --url=https://example.com --insecure

  Use a Proxy:
    %s --url=http://example.com --proxy="http://proxy.example.com:8080" --proxy-user="user" --proxy-password="pass"

  Test with Keep-Alive and HTTP/2 Disabled:
    %s --url=https://example.com --keep-alive --http2=false

  Specify Custom DNS Resolver:
    %s --url=http://example.com --resolver="1.1.1.1:53"

  Generate JSON Output:
    %s --url=http://example.com --output-format=json > result.json

`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

func main() {
	pflag.Parse()
	c.Logger = logger.New(c.LogLevel, c.OutputFile, c.Silent, c.FileName)

	if help {
		usage()
		os.Exit(0)
	}

	ctx, cancel := context.WithCancel(context.Background())
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-shutdown
		c.Logger.Info("Shutting down...")
		cancel()
	}()

	if c.OutputFormat == "json" || c.OutputFormat == "yaml" {
		c.Logger.Silent = true
	}

	c.Logger.Debug("Starting YAHBA with parsed flags")

	if err := run(ctx, c); err != nil {
		c.Logger.Error("Application encountered a critical error: %v", err)
		os.Exit(1)
	}

	// this obviously isn't actually doing anything yet
	cleanup(c.Logger)
}

func run(ctx context.Context, c config.Config) error {
	if c.Server {
		server.New()
		return nil
	}

	c.Logger.Debug("Validating configuration")
	if err := c.Validate(); err != nil {
		c.Logger.Error("Configuration validation failed: %v", err)
		return err
	}
	c.Logger.Info("Configuration validated successfully")

	if c.Headers != "" {
		c.Logger.Debug("Parsing headers: %s", c.Headers)
		parsedHeaders, err := util.ParseHeaders(c.Headers)
		if err != nil {
			c.Logger.Error("Error parsing headers: %v", err)
			return fmt.Errorf("error parsing headers: %w", err)
		}
		c.ParsedHeaders = parsedHeaders
		c.Logger.Info("Headers parsed successfully")
	}

	c.Logger.Info("Creating %d jobs for requests to %s", c.Requests, c.URL)
	jobs := make([]worker.Job, c.Requests)
	for i := 0; i < c.Requests; i++ {
		jobs[i] = worker.Job{Host: c.URL, Method: c.Method, Body: c.Body}
	}

	reportChan := make(chan report.Report, c.Requests)
	c.Logger.Info("Starting worker pool with %d requests per second (RPS)", c.RPS)
	startTime := time.Now().Format("Mon, 02 Jan 2006 15:04:05 MST")
	go worker.Work(ctx, c, jobs, reportChan)

	select {
	case <-ctx.Done():
		c.Logger.Info("Shutdown signal received. Cleaning up.")
		return nil
	case r := <-reportChan:
		r.StartTime = startTime
		r.EndTime = time.Now().Format("Mon, 02 Jan 2006 15:04:05 MST")
		parsedStartTime, err := time.Parse("Mon, 02 Jan 2006 15:04:05 MST", startTime)
		if err != nil {
			return err
		}
		r.Duration = time.Since(parsedStartTime)

		return generateReport(c, r)
	}
}

func generateReport(c config.Config, r report.Report) error {
	c.Logger.Info("Generating report in %s format", c.OutputFormat)

	var reportOutput string
	var err error

	switch c.OutputFormat {
	case "json":
		reportOutput, err = report.ParseJSON(r)
	case "yaml":
		reportOutput, err = report.ParseYAML(r)
	default:
		reportOutput, err = report.ParseRaw(r)
	}

	if err != nil {
		c.Logger.Error("Error generating report: %v", err)
		return err
	}

	c.Logger.Info("Report generated successfully")
	fmt.Fprintln(c.Logger.Writer(), reportOutput)
	return nil
}

func cleanup(logger *logger.Logger, channels ...chan any) {
	for _, ch := range channels {
		close(ch)
	}
	logger.Info("Cleanup complete")
}
