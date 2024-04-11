// Copyright 2024 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

// The LCP Server generates LCP licenses.
package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/edrlab/lcp-server/pkg/conf"
	"github.com/edrlab/lcp-server/pkg/stor"

	"encoding/json"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	"github.com/edrlab/lcp-server/pkg/api"
)

// Server context
type Server struct {
	*conf.Config
	stor.Store
	Cert   *tls.Certificate
	Router *chi.Mux
}

func main() {

	s := Server{}

	// Initialize the configuration from a config file or/and environment variables
	c, err := conf.Init(os.Getenv("LCPSERVER_CONFIG"))
	if err != nil {
		log.Println("Configuration failed: " + err.Error())
		os.Exit(1)
	}
	s.Config = c

	s.initialize()

	// TODO: add a graceful shutdown like in PubStore

	log.Println("Server starting on port " + strconv.Itoa(c.Port))
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(c.Port), s.Router))
}

// Initialize sets the database, X509 certificate and routes
func (s *Server) initialize() {
	var err error

	// Init database
	s.Store, err = stor.Init(s.Config.Dsn)
	if err != nil {
		log.Println("Database setup failed: " + err.Error())
		os.Exit(1)
	}

	// Init X509 certificate
	var certFile, privKeyFile string
	if certFile = s.Config.Certificate.Cert; certFile == "" {
		log.Println("Provider certificate missing")
		os.Exit(1)

	}
	if privKeyFile = s.Config.Certificate.PrivateKey; privKeyFile == "" {
		log.Println("Private key missing")
		os.Exit(1)
	}
	cert, err := tls.LoadX509KeyPair(certFile, privKeyFile)
	if err != nil {
		log.Println("Loading X509 key pair failed: " + err.Error())
		os.Exit(1)

	}
	s.Cert = &cert

	// Init routes
	s.Router = s.setRoutes()
}

func (s *Server) setRoutes() *chi.Mux {

	// Set api controller dependencies
	a := api.NewAPICtrl(s.Config, s.Store, s.Cert)

	// Define the router
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.NotFound(notFoundProblemDetail)

	// Public routes
	// Heartbeat
	r.Group(func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("The LCP Server is running!"))
		})
	})

	// Static resources management (optional)
	if s.Config.Resources != "" {
		r.Group(func(r chi.Router) {
			resourceDir := s.Config.Resources
			path := "/resources/*"

			r.Get(path, func(w http.ResponseWriter, r *http.Request) {
				rctx := chi.RouteContext(r.Context())
				pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
				fs := http.StripPrefix(pathPrefix, http.FileServer(http.Dir(resourceDir)))
				fs.ServeHTTP(w, r)
			})
		})
	}

	// Status document management
	r.Group(func(r chi.Router) {
		r.Use(render.SetContentType(render.ContentTypeJSON))
		r.Get("/status/{licenseID}", a.StatusDoc)   // GET /status/123
		r.Post("/register/{licenseID}", a.Register) // POST /register/123
		r.Put("/renew/{licenseID}", a.Renew)        // PUT /renew/123
		r.Put("/return/{licenseID}", a.Return)      // PUT /return/123
	})

	// Private Routes
	// Require Authentication
	credentials := make(map[string]string)
	credentials[s.Config.Access.Username] = s.Config.Access.Password

	r.Group(func(r chi.Router) {
		r.Use(middleware.BasicAuth("restricted", credentials))
		r.Use(render.SetContentType(render.ContentTypeJSON))

		// Publications, CRUD
		r.Route("/publications", func(r chi.Router) {
			r.With(paginate).Get("/", a.ListPublications)
			r.With(paginate).Get("/search", a.SearchPublications) // GET /publication/search{?format}
			r.Post("/", a.CreatePublication)                      // POST /publications

			r.Route("/{publicationID}", func(r chi.Router) {
				r.Get("/", a.GetPublication)       // GET /publications/123
				r.Put("/", a.UpdatePublication)    // PUT /publications/123
				r.Delete("/", a.DeletePublication) // DELETE /publications/123
			})
		})

		// LicenseInfo, CRUD
		r.Route("/licenseinfo", func(r chi.Router) {
			r.With(paginate).Get("/", a.ListLicenses)
			r.With(paginate).Get("/search", a.SearchLicenses) // GET /licenses/search{?pub,user,status,count}
			r.Post("/", a.CreateLicense)                      // POST /licenses

			r.Route("/{licenseID}", func(r chi.Router) {
				r.Get("/", a.GetLicense)       // GET /licenses/123
				r.Put("/", a.UpdateLicense)    // PUT /licenses/123
				r.Delete("/", a.DeleteLicense) // DELETE /licenses/123
			})
		})

		// License generation
		r.Route("/licenses", func(r chi.Router) {
			r.Post("/", a.GenerateLicense) // POST /licenses

			r.Route("/{licenseID}", func(r chi.Router) {
				r.Post("/", a.FreshLicense) // POST /licenses/123
			})
		})

		// License revocation
		r.Put("/revoke/{licenseID}", a.Revoke) // PUT /revoke/123

	})

	return r
}

// TODO: add pagination
// paginate is a stub, but very possible to implement middleware logic
// to handle the request params for handling a paginated request.
func paginate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// just a stub.. some ideas are to look at URL query params for something like
		// the page number, or the limit, and send a query cursor down the chain
		next.ServeHTTP(w, r)
	})
}

// notFoundProblemDetail formats not found errors as problem details, for the sake of consistency.
func notFoundProblemDetail(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{"type": "about:blank", "title": "Endpoint not found."}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)

	json.NewEncoder(w).Encode(response)
}
