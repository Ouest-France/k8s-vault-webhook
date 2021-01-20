package api

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

var (
	requestReceived = promauto.NewCounter(prometheus.CounterOpts{Name: "webhook_request_received", Help: "The total number of requests received"})
	secretMutated   = promauto.NewCounter(prometheus.CounterOpts{Name: "webhook_secret_mutated", Help: "The total number of secrets successfuly mutated"})
	secretIgnored   = promauto.NewCounter(prometheus.CounterOpts{Name: "webhook_secret_ignored", Help: "The total number of mutating requests ignored"})
	secretFailed    = promauto.NewCounter(prometheus.CounterOpts{Name: "webhook_secret_failed", Help: "The total number of mutating requests failed"})
)

// Server represent vault webhook server
type Server struct {
	Listen       string
	Cert         string
	Key          string
	Vault        VaultClient
	VaultPattern string
	Logger       *logrus.Logger
	BasicAuth    []string
}

// VaultClient interface validate a Vault read method
// and allow fake implementation for testing
type VaultClient interface {
	Read(path, key string) (string, error)
}

// Serve is the entrypoint of the API
func (s *Server) Serve() error {

	s.Logger.Infof("webhook started, listening on %s", s.Listen)
	err := http.ListenAndServeTLS(s.Listen, s.Cert, s.Key, s.Router())
	if err != nil {
		return fmt.Errorf("failed to start http server: %s", err)
	}

	return nil
}

// Router return a router with all API routes and middlewares
func (s *Server) Router() *chi.Mux {
	router := chi.NewRouter()
	router.Use(s.RequestLogger)
	router.Use(s.RequestCounter)

	router.Get("/status", s.statusHandler)
	router.Get("/metrics", promhttp.Handler().ServeHTTP)
	router.Group(func(router chi.Router) {
		router.Use(s.RequestAuth)
		router.Post("/secret", s.secretHandler)
	})

	return router
}

// RequestLogger is a middleware for logging request to stdout
func (s *Server) RequestLogger(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer s.Logger.WithFields(logrus.Fields{
			"type":   "http_request",
			"method": r.Method,
			"url":    r.URL,
			"host":   r.Host,
		}).Debug("request processed")

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// RequestCounter is a middleware counting requests for Prometheus metrics
func (s *Server) RequestCounter(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		// Increment Prometheus requestReceived counter
		requestReceived.Inc()
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}

// RequestAuth is a middleware checking API servers authentication through basicauth
func (s *Server) RequestAuth(next http.Handler) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {

		// Only check for authentication if basicauth is set via flags or config
		if len(s.BasicAuth) > 0 {

			// Extract user/password from request
			reqUser, reqPass, ok := r.BasicAuth()
			if !ok {
				s.Logger.Error("authentification failed, missing credentials")
				http.Error(w, http.StatusText(403), 403)
				return
			}

			// Check if user/password is one of defined by flags or config
			valid := func() bool {
				for _, cfgBasicauth := range s.BasicAuth {
					// Extract brcypt hashed password from basicauth config
					cfgUserPass := strings.Split(cfgBasicauth, ":")
					if cfgUserPass[0] != reqUser {
						continue
					}

					// Check password
					err := bcrypt.CompareHashAndPassword([]byte(cfgUserPass[1]), []byte(reqPass))
					if err == nil {
						return true
					}
				}
				return false
			}()
			if !valid {
				s.Logger.Error("authentication failed, invalid credentials")
				http.Error(w, http.StatusText(403), 403)
				return
			}
		}

		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
