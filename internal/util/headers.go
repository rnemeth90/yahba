package util

import (
	"fmt"
	"strings"
)

type Header struct {
	Key   string
	Value string
}

// ParseHeader parses a single header string into a key-value pair
func ParseHeader(raw string) (Header, error) {
	// Split the header string by ":"
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return Header{}, fmt.Errorf("invalid header format: %s", raw)
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	return Header{Key: key, Value: value}, nil
}

// ParseHeaders parses a raw string of multiple headers (comma-separated)
// into a map of header key-value pairs
func ParseHeaders(raw string) ([]Header, error) {
	headers := []Header{}

	// Split the headers by comma
	headerList := strings.Split(raw, ",")

	// Parse each individual header
	for _, h := range headerList {
		header, err := ParseHeader(h)
		if err != nil {
			return nil, err
		}
		headers = append(headers, header)
	}

	return headers, nil
}

// HeaderMapToParsedHeaders converts a def file's header map (as parsed from
// YAML) into the []Header format expected by the worker/client.
func HeaderMapToParsedHeaders(headers map[string]string) []Header {
	if len(headers) == 0 {
		return nil
	}
	parsed := make([]Header, 0, len(headers))
	for k, v := range headers {
		parsed = append(parsed, Header{Key: k, Value: v})
	}
	return parsed
}
