package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/rnemeth90/yahba/internal/config"
	"github.com/rnemeth90/yahba/internal/util"
	"github.com/spf13/pflag"
)

var c config.Config

func init() {
	pflag.StringVarP(&c.URL, "url", "u", "", "specify the URL to stress test")
	pflag.IntVarP(&c.Requests, "requests", "r", 4, "the total number of requests that should be sent")
	pflag.StringVarP(&c.Method, "method", "m", "GET", "which HTTP method to use (GET, POST, etc.)")
	pflag.StringVarP(&c.Headers, "headers", "H", "", "allows adding custom headers to the requests")
	pflag.StringVarP(&c.Payload, "payload", "p", "", "for POST and PUT requests, users can define the request body")
	pflag.IntVarP(&c.Timeout, "timeout", "t", 10, "the timeout for each request in seconds")
	pflag.IntVar(&c.RPS, "rps", 1, "requests per second")
	pflag.BoolVarP(&c.Insecure, "insecure", "i", false, "disable SSL/TLS certificate verification")
	pflag.StringVar(&c.Resolvers, "resolvers", "", "custom DNS resolvers (comma-separated list)")
	pflag.StringVarP(&c.Proxy, "proxy", "P", "", "proxy server address")
	pflag.BoolVarP(&c.KeepAlive, "keep-alive", "k", false, "use keep-alives")
	pflag.StringVarP(&c.Cookies, "cookies", "C", "", "set cookies")
	pflag.BoolVar(&c.HTTP2, "http2", true, "use HTTP/2")
	pflag.BoolVarP(&c.Verbose, "verbose", "v", false, "enable verbose mode")
	pflag.StringVarP(&c.OutputFormat, "output", "o", "json", "output format (json/yaml/raw)")
	pflag.BoolVar(&c.Compression, "compression", false, "use compression")
	pflag.StringVar(&c.ProxyUser, "proxy-user", "", "proxy user name")
	pflag.StringVar(&c.ProxyPassword, "proxy-password", "", "proxy password")
}

func main() {
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
	headers := []string{}
	if c.Headers != "" {
		if strings.Contains(c.Headers, ",") {
			headers = util.ParseHeaders(c.Headers)
		} else {
			headers = util.ParseHeader(c.Headers)
		}
	}

	return nil
}
