package report

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLatencyHistogram(t *testing.T) {
	results := []Result{
		{ElapsedTime: 500 * time.Microsecond},
		{ElapsedTime: time.Millisecond}, // boundary is inclusive
		{ElapsedTime: 3 * time.Millisecond},
		{ElapsedTime: 40 * time.Millisecond},
		{ElapsedTime: 10 * time.Second},
	}
	h := LatencyHistogram(results)

	assert.Equal(t, "+Inf", h[len(h)-1].LE)
	assert.Equal(t, 2, h[0].Count)
	assert.Equal(t, 1, h[2].Count) // <=5ms
	assert.Equal(t, 1, h[5].Count) // <=50ms
	assert.Equal(t, 1, h[len(h)-1].Count)

	total := 0
	for _, b := range h {
		total += b.Count
	}
	assert.Equal(t, len(results), total)
}

func TestMarkKnee(t *testing.T) {
	step := func(rps int, p99 time.Duration) Step { return Step{TargetRPS: rps, P99Duration: p99, Passed: true} }

	r := RampReport{Steps: []Step{
		step(100, 2*time.Millisecond), step(200, 3*time.Millisecond),
		step(300, 5*time.Millisecond), step(400, 40*time.Millisecond),
	}}
	assert.Equal(t, 300, r.MarkKnee())
	assert.Equal(t, 300, r.KneeRPS)
	assert.True(t, r.Steps[2].Knee)
	assert.False(t, r.Steps[3].Knee)

	flat := RampReport{Steps: []Step{step(100, 2*time.Millisecond), step(200, 3*time.Millisecond)}}
	assert.Zero(t, flat.MarkKnee())

	// sub-millisecond baselines are floored to 1ms so jitter is not a knee
	jitter := RampReport{Steps: []Step{step(100, 100*time.Microsecond), step(200, 900*time.Microsecond)}}
	assert.Zero(t, jitter.MarkKnee())

	assert.Zero(t, (&RampReport{}).MarkKnee())
}

func TestFormatStepLine(t *testing.T) {
	line := FormatStepLine(Step{TargetRPS: 500, Passed: false, Reason: "p99 too high"})
	assert.Contains(t, line, "500")
	assert.Contains(t, line, "FAIL (p99 too high)")
}
