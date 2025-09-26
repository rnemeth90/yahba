package server

import (
	"net/http"

	"github.com/rnemeth90/yahba/internal/logger"
)

func testHandler(w http.ResponseWriter, r *http.Request) {
	logger := logger.New("debug", "stdout", false)
	logger.Info("Hello sent")
	w.WriteHeader(http.StatusOK)
}
