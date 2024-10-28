package report

import (
	"encoding/json"
	"fmt"
)

// this is dumb. Don't do this.
func ParseRaw(report chan Report) string {
	result := <-report
	fmt.Println("parsing report...")
	for _, r := range result.Results {
		fmt.Println(r.ResultCode)
	}
	return ""
}

func ParseJSON(report chan Report) (string, error) {
	result := <-report
	fmt.Println("parsing json report...")
	jsonStr, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}

	return string(jsonStr), nil
}
