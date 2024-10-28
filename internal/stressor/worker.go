package stressor

import (
	"bytes"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/rnemeth90/yahba/internal/client"
	"github.com/rnemeth90/yahba/internal/config"
	"github.com/rnemeth90/yahba/internal/report"
)

type Worker struct {
	ID      int
	Jobs    <-chan Job
	Results chan<- report.Result
	Client  *http.Client
	Config  config.Config
}

type Job struct {
	Host   string
	Method string
	Body   string
}

func newWorker(id int, jobs <-chan Job, results chan<- report.Result, client *http.Client, cfg config.Config) *Worker {
	return &Worker{
		ID:      id,
		Jobs:    jobs,
		Results: results,
		Client:  client,
		Config:  cfg,
	}
}

func (w *Worker) work(wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range w.Jobs {
		log.Printf("worker %d processing job %s", w.ID, job.Host)

		req, err := http.NewRequest(job.Method, job.Host, bytes.NewReader([]byte(job.Body)))
		if err != nil {
			w.Results <- report.Result{WorkerID: w.ID, Error: err}
			continue
		}

		for _, h := range w.Config.ParsedHeaders {
			req.Header.Add(h.Key, h.Value)
		}
		log.Println("headers:", req.Header)

		if !w.Config.HTTP2 {
			req.Proto = "HTTP/1.1"
		}

		start := time.Now()
		resp, err := w.Client.Do(req)
		if err != nil {
			w.Results <- report.Result{WorkerID: w.ID, Error: err}
			continue
		}
		end := time.Now()
		total := time.Since(start)

		w.Results <- report.Result{
			StartTime:   start,
			EndTime:     end,
			ElapsedTime: total,
			WorkerID:    w.ID,
			ResultCode:  resp.StatusCode,
			Error:       nil,
			TargetURL:   job.Host,
			Method:      job.Method,
		}
		resp.Body.Close()
	}
}

func WorkerPool(cfg config.Config, jobs []Job, reportChan chan<- report.Report) {
	client, err := client.NewClient(cfg)
	if err != nil {
		log.Fatalf("Error creating HTTP client: %v", err)
	}

	jobChan := make(chan Job, len(jobs))
	resultChan := make(chan report.Result, len(jobs))
	wg := &sync.WaitGroup{}

	for i := 0; i < cfg.RPS; i++ {
		worker := newWorker(i, jobChan, resultChan, client, cfg)
		wg.Add(1)
		go worker.work(wg)
	}

	// Throttle the rate at which jobs are added to the job channel based on cfg.RPS
	ticker := time.NewTicker(time.Second / time.Duration(cfg.RPS))
	defer ticker.Stop()

	// put jobs on the job channel
	go func() {
		for _, job := range jobs {
			<-ticker.C // Limit the job addition based on the RPS
			log.Printf("Sending job to jobChan: %s", job.Host)
			jobChan <- job
		}
		close(jobChan)
	}()

	// Wait for all workers to finish
	wg.Wait()
	close(resultChan)

	report := report.Report{}
	// var totalSent, totalReceived int
	// var totalSuccess, totalFailure int
	// var statusCodeCounts = make(map[int]int)

	log.Println("parsing results...")
	for result := range resultChan {
		report.Results = append(report.Results, result)
	}

	reportChan <- report
	close(reportChan)
}
