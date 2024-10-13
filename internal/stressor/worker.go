package stressor

import (
	"net/http"
	"sync"
	"time"

	"github.com/rnemeth90/yahba/internal/client"
	"github.com/rnemeth90/yahba/internal/config"
)

type Worker struct {
	ID      int
	Jobs    <-chan Job
	Results chan<- Result
	Config  config.Config
	Client  *http.Client
}

type Job struct {
	URL     string
	Method  string
	Payload string
}

type Result struct {
	ResultCode int
	WorkerID   int
	Error      error
}

func newWorker(id int, jobs chan Job, results chan Result, client *http.Client) *Worker {
	return &Worker{
		ID:      id,
		Jobs:    jobs,
		Results: results,
		Client:  client,
	}
}

func (w *Worker) work(wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range w.Jobs {
		req, err := http.NewRequest(job.Method, job.URL, nil)
		if err != nil {
			w.Results <- Result{WorkerID: w.ID, Error: err}
			continue
		}

		req.Header.Add

		resp, err := w.Client.Do(req)
		if err != nil {
			w.Results <- Result{WorkerID: w.ID, Error: err}
			continue
		}

		w.Results <- Result{ResultCode: resp.StatusCode, Error: err, WorkerID: w.ID}
		resp.Body.Close()
	}
}

func WorkerPool(config config.Config, jobs []Job) {
	client, err := client.NewClient(config)
	if err != nil {

	}

	jobChan := make(chan Job, len(jobs))
	resultChan := make(chan Result, len(jobs))
	wg := &sync.WaitGroup{}

	// create a worker for each request. Requests = total requests
	for i := 0; i < config.Requests; i++ {

		// but only create config.Rps workers per second
		for j := 0; j < config.RPS; j++ {
			worker := newWorker(j, jobChan, resultChan, client)
			wg.Add(1)
			go worker.work(wg)
			time.Sleep(1 * time.Second)
		}
	}

	for _, job := range jobs {
		jobChan <- job
	}
	close(jobChan)

	wg.Wait()
	close(resultChan)

	for result := range resultChan {
		if result.Error != nil {
			// do something if error is not nil
			// create results and return for report?

		} else {
			// do something if the request was successful

		}
	}
}
