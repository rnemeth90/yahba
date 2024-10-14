package util

import (
	"fmt"
	"strings"
)

// ParseHeader parses a single header string into a key-value pair
func ParseHeader(raw string) (string, string, error) {
	// Split the header string by ":"
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid header format: %s", raw)
	}

	key := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	return key, value, nil
}

// ParseHeaders parses a raw string of multiple headers (comma-separated)
// into a map of header key-value pairs
func ParseHeaders(raw string) (map[string]string, error) {
	headers := make(map[string]string)

	// Split the headers by comma
	headerList := strings.Split(raw, ",")

	// Parse each individual header
	for _, h := range headerList {
		key, value, err := ParseHeader(h)
		if err != nil {
			return nil, err
		}
		headers[key] = value
	}

	return headers, nil
}
