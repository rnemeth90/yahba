package server

import (
	"net/http"

	"github.com/rnemeth90/yahba/internal/logger"
)

func Run(port string) error {
	l := logger.New("debug", "stdout", false)

	http.HandleFunc("/test", testHandler)

	if port != "" {
		l.Debug("Starting server on port ", port)
		return http.ListenAndServe(port, nil)
	}

	return ErrInvalidPort
}
