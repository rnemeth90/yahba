package server

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"
)

func (s *Server) testHandler(w http.ResponseWriter, r *http.Request) {
	s.logRequest(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "YAHBA test server: OK")
}

func (s *Server) aliveHandler(w http.ResponseWriter, r *http.Request) {
	s.logRequest(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "YAHBA test server: Alive")
}

func (s *Server) readyHandler(w http.ResponseWriter, r *http.Request) {
	s.logRequest(r)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "YAHBA test server: Ready")
}

func (s *Server) slowHandler(w http.ResponseWriter, r *http.Request) {
	s.logRequest(r)
	time.Sleep(2 * time.Second)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "YAHBA test server: Slow response")
}

func (s *Server) errorHandler(w http.ResponseWriter, r *http.Request) {
	s.logRequest(r)
	w.WriteHeader(http.StatusInternalServerError)
	fmt.Fprintln(w, "YAHBA test server: Internal error")
}

func (s *Server) randomDelayHandler(w http.ResponseWriter, r *http.Request) {
	s.logRequest(r)
	delay := time.Duration(rand.Intn(3000)) * time.Millisecond
	time.Sleep(delay)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "YAHBA test server: Random delay of %v\n", delay)
}

func (s *Server) randomErrorHandler(w http.ResponseWriter, r *http.Request) {
	s.logRequest(r)
	statuses := []int{200, 200, 200, 500, 503, 400}
	status := statuses[rand.Intn(len(statuses))]
	w.WriteHeader(status)
	fmt.Fprintf(w, "YAHBA test server: Simulated status code %d\n", status)
}

func (s *Server) logRequest(r *http.Request) {
	s.Logger.Info("➡️  %s %s", r.Method, r.URL.Path)
}
