package api

import (
	"fmt"
	"net/http"

	"github.com/Ouest-France/k8s-vault-webhook/vault"
	"github.com/go-chi/chi"
	"github.com/sirupsen/logrus"
)

type Server struct {
	Listen       string
	Cert         string
	Key          string
	Vault        vault.Client
	VaultPattern string
	Logger       *logrus.Logger
}

func (s *Server) Serve() error {
	router := chi.NewRouter()

	router.Post("/secret", s.secretHandler)
	router.Get("/status", s.statusHandler)

	s.Logger.Infof("webhook started, listening on %s", s.Listen)
	err := http.ListenAndServeTLS(s.Listen, s.Cert, s.Key, router)
	if err != nil {
		return fmt.Errorf("failed to start http server: %s", err)
	}

	return nil
}
