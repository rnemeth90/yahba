package report

import "fmt"

func PrintRaw(report chan Report) {
	result := <-report
	fmt.Println("parsing report...")
	for _, r := range result.Results {
		fmt.Println(r.ResultCode)
	}
}
