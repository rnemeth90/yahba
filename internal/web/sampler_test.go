package web

import (
	"testing"
	"time"

	"github.com/rnemeth90/yahba/internal/report"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func res(start time.Time, endOffset, elapsed time.Duration, code int) report.Result {
	return report.Result{
		StartTime:   start.Add(endOffset - elapsed),
		EndTime:     start.Add(endOffset),
		ElapsedTime: elapsed,
		ResultCode:  code,
	}
}

func TestSamplerBucketsBySecond(t *testing.T) {
	start := time.Now()
	s := NewSampler(start)

	assert.Empty(t, s.Add(res(start, 100*time.Millisecond, 10*time.Millisecond, 200)))
	assert.Empty(t, s.Add(res(start, 900*time.Millisecond, 30*time.Millisecond, 500)))

	got := s.Add(res(start, 1100*time.Millisecond, 20*time.Millisecond, 200))
	require.Len(t, got, 1)
	assert.Equal(t, 0, got[0].T)
	assert.Equal(t, 2, got[0].Count)
	assert.Equal(t, 1, got[0].Errors)
	assert.InDelta(t, 20.0, got[0].AvgMs, 0.001)
	assert.InDelta(t, 10.0, got[0].P50Ms, 0.001)
	assert.InDelta(t, 30.0, got[0].P99Ms, 0.001)
	assert.Equal(t, map[int]int{200: 1, 500: 1}, got[0].StatusCodes)

	final := s.Flush()
	require.Len(t, final, 1)
	assert.Equal(t, 1, final[0].T)
	assert.Equal(t, 1, final[0].Count)
	assert.Nil(t, s.Flush())
}

func TestSamplerLateResultStaysInCurrentBucket(t *testing.T) {
	start := time.Now()
	s := NewSampler(start)

	s.Add(res(start, 2500*time.Millisecond, 5*time.Millisecond, 200))
	assert.Empty(t, s.Add(res(start, 500*time.Millisecond, 5*time.Millisecond, 200)))

	final := s.Flush()
	require.Len(t, final, 1)
	assert.Equal(t, 2, final[0].T)
	assert.Equal(t, 2, final[0].Count)
}

func TestSamplerCountsConnectionErrorsAsFailures(t *testing.T) {
	start := time.Now()
	s := NewSampler(start)
	s.Add(res(start, 10*time.Millisecond, time.Millisecond, 0))
	final := s.Flush()
	require.Len(t, final, 1)
	assert.Equal(t, 1, final[0].Errors)
}
