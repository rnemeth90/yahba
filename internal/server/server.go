package server

import (
	"net/http"

	"github.com/rnemeth90/yahba/internal/logger"
)

func New() error {
	l := logger.New("debug", "stdout", false)

	http.HandleFunc("/test", testHandler)

	l.Debug("Starting server")
	err := http.ListenAndServe(":8085", nil)
	return err
}
