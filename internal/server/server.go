package server

import (
	"net/http"

	"github.com/rnemeth90/yahba/internal/logger"
)

type Server struct {
	Config *Config
	Logger *logger.Logger
}

func (s *Server) Run() error {
	http.HandleFunc("/test", testHandler)

	if s.Config.Port != "" {
		s.Logger.Debug("Starting server on port %s", s.Config.Port)
		return http.ListenAndServe(s.Config.Port, nil)
	}

	return ErrInvalidPort
}
