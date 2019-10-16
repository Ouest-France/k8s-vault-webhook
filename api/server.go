package api

import (
	"fmt"
	"net/http"

	"github.com/Ouest-France/k8s-vault-webhook/vault"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
)

type Server struct {
	Listen       string
	Cert         string
	Key          string
	Vault        vault.Client
	VaultPattern string
}

func (s *Server) Serve() error {
	router := chi.NewRouter()
	router.Use(middleware.Logger)

	router.Post("/secret", s.secretHandler)

	err := http.ListenAndServeTLS(s.Listen, s.Cert, s.Key, router)
	if err != nil {
		return fmt.Errorf("failed to start http server: %s", err)
	}

	return nil
}
