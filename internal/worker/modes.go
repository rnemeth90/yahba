package worker

import (
	"context"
	"fmt"
	"time"

	"github.com/rnemeth90/yahba/internal/client"
	"github.com/rnemeth90/yahba/internal/config"
	"github.com/rnemeth90/yahba/internal/report"
)

// rampPassFraction is the share of the target rate a step must achieve to pass.
const rampPassFraction = 0.9

// RunMax is the closed-loop mode: every worker sends requests back-to-back
// for cfg.Duration, so throughput is limited only by the server. Requests in
// flight when the duration ends are allowed to finish.
func RunMax(ctx context.Context, cfg config.Config, reportChan chan<- report.Report, factory WorkerFactory, progressChan chan<- report.Result) {
	c, err := client.NewClient(cfg)
	if err != nil {
		cfg.Logger.Error("Error creating HTTP client: %v", err)
		return
	}

	workers := numWorkers(cfg)
	produce := func(ctx context.Context, jobs chan<- Job) {
		deadline := time.NewTimer(cfg.Duration)
		defer deadline.Stop()
		for i := 0; ; i++ {
			select {
			case <-ctx.Done():
				return
			case <-deadline.C:
				return
			case jobs <- Job{ID: i, Host: cfg.URL, Method: cfg.Method, Body: cfg.Body}:
			}
		}
	}

	// unbuffered dispatch: no queued work runs after the deadline
	rep := runPoolBuffered(ctx, cfg, c, factory, produce, workers, 0, progressChan)
	rep.Mode = config.ModeMax

	reportChan <- rep
	close(reportChan)
}

// RunRamp is the open-loop mode: it offers cfg.RampStart req/s for
// cfg.StepDuration, then raises the rate by cfg.RampStep, and so on until a
// step breaches a threshold (error rate, p99, or failing to reach the target
// rate), cfg.RampMax is passed, or ctx ends.
func RunRamp(ctx context.Context, cfg config.Config, factory WorkerFactory) (report.RampReport, error) {
	rr := report.RampReport{
		Host:         cfg.URL,
		Method:       cfg.Method,
		MaxErrorRate: cfg.MaxErrorRate,
	}
	if cfg.MaxP99 > 0 {
		rr.MaxP99 = cfg.MaxP99.String()
	}

	c, err := client.NewClient(cfg)
	if err != nil {
		return rr, err
	}

	workers := numWorkers(cfg)
	for rps := cfg.RampStart; ; rps += cfg.RampStep {
		switch {
		case ctx.Err() != nil:
			rr.StopReason = "interrupted"
			return rr, nil
		case cfg.RampMax > 0 && rps > cfg.RampMax:
			rr.StopReason = fmt.Sprintf("reached --rps-max (%d) without breaking", cfg.RampMax)
			return rr, nil
		case rps > config.MaxRPS:
			rr.StopReason = fmt.Sprintf("reached the supported maximum of %d RPS without breaking", config.MaxRPS)
			return rr, nil
		}

		cfg.Logger.Info("Ramp step: %d RPS for %s", rps, cfg.StepDuration)
		job := func(i int) Job { return Job{ID: i, Host: cfg.URL, Method: cfg.Method, Body: cfg.Body} }
		produce := pacedProducer(rps, 0, cfg.StepDuration, 0, job)
		rep := runPool(ctx, cfg, c, factory, produce, workers, nil)

		if ctx.Err() != nil {
			rr.StopReason = "interrupted"
			return rr, nil
		}

		step := evaluateStep(cfg, rps, rep)
		rr.Steps = append(rr.Steps, step)
		if !step.Passed {
			rr.BreakingRPS = rps
			rr.StopReason = "step failed: " + step.Reason
			return rr, nil
		}
		rr.SustainedRPS = rps
	}
}

func evaluateStep(cfg config.Config, rps int, rep report.Report) report.Step {
	step := report.Step{
		TargetRPS:   rps,
		AchievedRPS: float64(rep.TotalRequests) / cfg.StepDuration.Seconds(),
		Requests:    rep.TotalRequests,
		Failures:    rep.Failures,
		P50:         rep.Latency.P50,
		P95:         rep.Latency.P95,
		P99:         rep.Latency.P99,
		Passed:      true,
	}
	if rep.TotalRequests > 0 {
		step.ErrorRate = float64(rep.Failures) / float64(rep.TotalRequests) * 100
	}

	switch {
	case rep.TotalRequests == 0:
		step.Passed, step.Reason = false, "no requests completed"
	case step.ErrorRate > cfg.MaxErrorRate:
		step.Passed = false
		step.Reason = fmt.Sprintf("error rate %.2f%% > %.2f%%", step.ErrorRate, cfg.MaxErrorRate)
	case cfg.MaxP99 > 0 && report.LatencyPercentile(rep.Results, 99) > cfg.MaxP99:
		step.Passed = false
		step.Reason = fmt.Sprintf("p99 %s > %s", step.P99, cfg.MaxP99)
	case step.AchievedRPS < float64(rps)*rampPassFraction:
		step.Passed = false
		step.Reason = fmt.Sprintf("achieved %.1f RPS < %d%% of target", step.AchievedRPS, int(rampPassFraction*100))
	}
	return step
}
