package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// DefFileRequest describes a single named request within a test definition
// (def) file. Example:
//
// requests:
//   - name: get-homepage
//     url: https://example.com/
//     method: GET
//     headers:
//     Authorization: "******"
//     rps: 20
//     requests: 500
//   - name: create-user
//     url: https://example.com/api/users
//     method: POST
//     body: '{"name":"test"}'
//     headers:
//     Content-Type: application/json
//     rps: 5
//     requests: 100
//     depends_on: get-homepage   # optional sequencing
type DefFileRequest struct {
	Name      string            `yaml:"name"`
	URL       string            `yaml:"url"`
	Method    string            `yaml:"method"`
	Body      string            `yaml:"body"`
	Headers   map[string]string `yaml:"headers"`
	RPS       int               `yaml:"rps"`
	Requests  int               `yaml:"requests"`
	DependsOn string            `yaml:"depends_on"`
}

// DefFile represents the top-level structure of a test definition file.
type DefFile struct {
	Requests []DefFileRequest `yaml:"requests"`
}

// ValidateDefFile reads and validates that a definition file is well-formed:
// valid YAML, at least one request, unique non-empty names, valid HTTP
// methods, positive RPS/Requests counts (defaulted when omitted), bodies
// present for POST/PUT, and depends_on references that exist and don't form
// a cycle.
func ValidateDefFile(fileName string) error {
	def, err := loadAndValidateDefFile(fileName)
	if err != nil {
		return err
	}
	_, err = orderRequests(def.Requests)
	return err
}

// ParseDefFile reads, validates, and parses a definition file, returning the
// requests ordered so that dependencies (depends_on) always run before the
// requests that depend on them.
func ParseDefFile(fileName string) (*DefFile, error) {
	def, err := loadAndValidateDefFile(fileName)
	if err != nil {
		return nil, err
	}

	ordered, err := orderRequests(def.Requests)
	if err != nil {
		return nil, err
	}
	def.Requests = ordered

	return def, nil
}

func loadAndValidateDefFile(fileName string) (*DefFile, error) {
	if fileName == "" {
		return nil, ErrMissingDefFile
	}

	data, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to read definition file %q: %w", fileName, err)
	}

	var def DefFile
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, fmt.Errorf("failed to parse definition file %q: %w", fileName, err)
	}

	if len(def.Requests) == 0 {
		return nil, fmt.Errorf("definition file %q must contain at least one request", fileName)
	}

	seen := make(map[string]bool, len(def.Requests))
	for i, r := range def.Requests {
		if r.Name == "" {
			return nil, fmt.Errorf("request at index %d is missing a name", i)
		}
		if seen[r.Name] {
			return nil, fmt.Errorf("duplicate request name %q", r.Name)
		}
		seen[r.Name] = true

		if r.URL == "" {
			return nil, fmt.Errorf("request %q is missing a url", r.Name)
		}
		if !strings.HasPrefix(r.URL, "http") {
			return nil, fmt.Errorf("request %q has an invalid url scheme: %s", r.Name, r.URL)
		}

		method := strings.ToUpper(r.Method)
		if method == "" {
			method = "GET"
		}
		if !validHTTPMethods[method] {
			return nil, fmt.Errorf("request %q has an invalid method: %s", r.Name, r.Method)
		}
		def.Requests[i].Method = method

		if (method == "POST" || method == "PUT") && r.Body == "" {
			return nil, fmt.Errorf("request %q uses method %s but has no body", r.Name, method)
		}

		if r.RPS <= 0 {
			def.Requests[i].RPS = 1
		}
		if r.Requests <= 0 {
			def.Requests[i].Requests = 1
		}
	}

	for _, r := range def.Requests {
		if r.DependsOn == "" {
			continue
		}
		if r.DependsOn == r.Name {
			return nil, fmt.Errorf("request %q cannot depend on itself", r.Name)
		}
		if !seen[r.DependsOn] {
			return nil, fmt.Errorf("request %q depends on unknown request %q", r.Name, r.DependsOn)
		}
	}

	return &def, nil
}

// orderRequests returns requests ordered so that each request appears after
// the request it depends on (if any), returning an error if a circular
// depends_on chain is detected.
func orderRequests(requests []DefFileRequest) ([]DefFileRequest, error) {
	byName := make(map[string]DefFileRequest, len(requests))
	for _, r := range requests {
		byName[r.Name] = r
	}

	const (
		unvisited = iota
		visiting
		visited
	)
	state := make(map[string]int, len(requests))
	ordered := make([]DefFileRequest, 0, len(requests))

	var visit func(name string) error
	visit = func(name string) error {
		switch state[name] {
		case visited:
			return nil
		case visiting:
			return fmt.Errorf("circular depends_on detected involving request %q", name)
		}

		state[name] = visiting
		if r, ok := byName[name]; ok && r.DependsOn != "" {
			if err := visit(r.DependsOn); err != nil {
				return err
			}
		}
		state[name] = visited
		ordered = append(ordered, byName[name])
		return nil
	}

	for _, r := range requests {
		if err := visit(r.Name); err != nil {
			return nil, err
		}
	}

	return ordered, nil
}
