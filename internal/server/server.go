package server

import (
	"net/http"

	"github.com/rnemeth90/yahba/internal/logger"
)

var (
	count int
)

type Server struct {
	Config *Config
	Logger *logger.Logger
}

func (s *Server) Run() error {
	s.Logger = logger.New("debug", "stdout", false)

	http.HandleFunc("/", s.testHandler)
	http.HandleFunc("/alive", s.aliveHandler)
	http.HandleFunc("/ready", s.readyHandler)
	http.HandleFunc("/slow", s.slowHandler)
	http.HandleFunc("/error", s.errorHandler)
	http.HandleFunc("/random-delay", s.randomDelayHandler)
	http.HandleFunc("/random-error", s.randomErrorHandler)

	s.Logger.Info("test server listening on port %s", s.Config.Port)

	if s.Config.Port != "" {
		s.Logger.Info("Starting server on port %s", s.Config.Port)
		return http.ListenAndServe(s.Config.Port, nil)
	}

	return ErrInvalidPort
}
