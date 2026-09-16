package tui

import (
	"context"
	"strings"

	"github.com/rnemeth90/yahba/internal/config"
	"github.com/rnemeth90/yahba/internal/report"
	"github.com/rnemeth90/yahba/internal/util"
	"github.com/rnemeth90/yahba/internal/worker"
)

// runDefFileTest runs each named request in a definition file, in dependency
// order, forwarding individual results to progressChan as they arrive and
// finally sending one combined, aggregated report to reportChan. Mirrors
// cmd.runFromDefFile, adapted to stream live progress for the TUI.
func runDefFileTest(ctx context.Context, baseCfg config.Config, def *config.DefFile, factory worker.WorkerFactory, progressChan chan<- report.Result, reportChan chan<- report.Report) {
	defer close(reportChan)
	defer close(progressChan)

	var allResults []report.Result
	hosts := make([]string, 0, len(def.Requests))
	methods := make(map[string]bool)

	// don't judge me ...
requestLoop:
	for _, reqDef := range def.Requests {
		select {
		case <-ctx.Done():
			break requestLoop
		default:
		}

		reqCfg := baseCfg
		reqCfg.URL = reqDef.URL
		reqCfg.Method = reqDef.Method
		reqCfg.Body = reqDef.Body
		reqCfg.RPS = reqDef.RPS
		reqCfg.Requests = reqDef.Requests
		reqCfg.Headers = ""
		reqCfg.ParsedHeaders = util.HeaderMapToParsedHeaders(reqDef.Headers)

		jobs := make([]worker.Job, reqCfg.Requests)
		for i := 0; i < reqCfg.Requests; i++ {
			jobs[i] = worker.Job{ID: i, Host: reqCfg.URL, Method: reqCfg.Method, Body: reqCfg.Body}
		}

		reqProgressChan := make(chan report.Result, reqCfg.Requests)
		reqReportChan := make(chan report.Report, 1)

		go worker.Work(ctx, reqCfg, jobs, reqReportChan, factory, reqProgressChan)

		for r := range reqProgressChan {
			progressChan <- r
		}

		if reqReport, ok := <-reqReportChan; ok {
			allResults = append(allResults, reqReport.Results...)
			hosts = append(hosts, reqCfg.URL)
			methods[reqCfg.Method] = true
		}
	}

	combined := report.Aggregate(allResults)
	combined.Host = strings.Join(hosts, ", ")
	switch {
	case len(methods) == 1:
		for method := range methods {
			combined.Method = method
		}
	case len(methods) > 1:
		combined.Method = "MULTI"
	}

	reportChan <- combined
}
