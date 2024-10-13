package util

import "strings"

// parseHeader
func ParseHeader(raw string) []string {
	return strings.Split(raw, ":")
}

// parseHeaders
func ParseHeaders(raw string) []string {
	h := strings.Split(raw, ",")
	headers := []string{}

	for _, v := range h {
		headers = append(headers, strings.Split(v, ":")...)
	}

	return headers
}
