package api

import (
	"net/http"

	"github.com/go-chi/render"
)

func (s *Server) statusHandler(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, map[string]string{"status": "up"})
}
