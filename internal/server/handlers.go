package server

import (
	"net/http"

	"github.com/rnemeth90/yahba/internal/logger"
)

var count int

func testHandler(w http.ResponseWriter, r *http.Request) {
	logger := logger.New("debug", "stdout", false)
	logger.Info("Hello sent")
	w.WriteHeader(http.StatusOK)
}

func aliveHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func readyHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}
