package worker

import (
	"net/http"
	"net/url"
	"time"

	"github.com/rnemeth90/yahba/internal/report"
)

func (w *Worker) handleClientError(job Job, result report.Result, resp *http.Response, err error, start time.Time, end time.Time) {
	if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
		w.Config.Logger.Warn("worker %d: Request to %s timed out", w.ID, job.Host)
		result.Timeout = true
		result.ResultCode = http.StatusRequestTimeout
		// urlErr.Unwrap()
	} else {
		w.Config.Logger.Error("worker %d: Request to %s failed: %v", w.ID, job.Host, err)
	}

	result.Error = err
	result.EndTime = end
	result.ElapsedTime = result.EndTime.Sub(start)

	if resp != nil {
		result.ResultCode = resp.StatusCode
		result.Method = resp.Request.Method
		result.TargetURL = resp.Request.URL.RawPath
	}

	w.Results <- result
}

func (w *Worker) handleRequestError(job Job, err error) {
	w.Results <- report.Result{
		WorkerID:  w.ID,
		Method:    job.Method,
		TargetURL: job.Host,
		Error:     err,
	}
}
