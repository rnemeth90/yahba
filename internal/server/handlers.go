package server

import (
	"net/http"
)

func (s *Server) testHandler(w http.ResponseWriter, r *http.Request) {
	s.Logger.Debug("Hello sent")
	w.WriteHeader(http.StatusOK)
}
