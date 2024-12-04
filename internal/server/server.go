package server

import (
	"net/http"

	"github.com/rnemeth90/yahba/internal/logger"
)

func New() {
	l := logger.New("debug", "stdout", false, "")

	http.HandleFunc("/test", testHandler)

	l.Debug("Starting server")
	http.ListenAndServe(":8085", nil)
}
