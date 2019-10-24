package api

import (
	"net/http"

	"github.com/go-chi/render"
)

func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	s.Logger.Debugf("healtcheck request received, status up")
	render.JSON(w, r, map[string]string{"status": "up"})
}
