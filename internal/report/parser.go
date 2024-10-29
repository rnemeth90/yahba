package report

import (
	"encoding/json"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// this is dumb. Don't do this.
func ParseRaw(report chan Report) (string, error) {
	result, ok := <-report
	if !ok {
		return "", errors.New("channel unexpectedly closed...")
	}

	fmt.Println("parsing report...")
	for _, r := range result.Results {
		fmt.Println(r.ResultCode)
	}

	return "", nil
}

func ParseJSON(report chan Report) (string, error) {
	result, ok := <-report
	if !ok {
		return "", errors.New("channel unexpectedly closed...")
	}

	fmt.Println("parsing json report...")
	jsonStr, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonStr), nil
}

func ParseYAML(report chan Report) (string, error) {
	result, ok := <-report
	if !ok {
		return "", errors.New("channel unexpectedly closed...")
	}

	fmt.Println("parsing yaml report...")
	yamlStr, err := yaml.Marshal(result)
	if err != nil {
		return "", err
	}

	return string(yamlStr), nil
}
