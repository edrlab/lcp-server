// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

// LCP Server generates LCP licenses.
package main

import (
	"crypto/tls"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	"github.com/edrlab/lcp-server/pkg/api"
	"github.com/edrlab/lcp-server/pkg/conf"
	"github.com/edrlab/lcp-server/pkg/stor"
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

	configFile := os.Getenv("EDRLAB_LCPSERVER_CONFIG")
	if configFile == "" {
		panic("Failed to retrieve the configuration file path.")
	}

	c, err := conf.ReadConfig(configFile)
	if err != nil {
		panic("Failed to read the configuration.")
	}

	s.Config = c

	s.Initialize()

	log.Printf("The server is ready.")

	if c.Port == 0 {
		c.Port = 8081
	}

	s.Run(":" + strconv.Itoa(c.Port))
}

// Initialize sets up the database and routes
func (s *Server) Initialize() {
	var err error

	// Setup the database
	s.Store, err = stor.DBSetup(s.Config.Dsn)
	if err != nil {
		panic("Database setup failed.")
	}

	// Setup the X509 certificate
	var certFile, privKeyFile string
	if certFile = s.Config.Certificate.Cert; certFile == "" {
		panic("Must specify a certificate.")
	}
	if privKeyFile = s.Config.Certificate.PrivateKey; privKeyFile == "" {
		panic("Must specify a private key.")
	}
	cert, err := tls.LoadX509KeyPair(certFile, privKeyFile)
	if err != nil {
		panic(err)
	}
	s.Cert = &cert

	// Setup the routes
	s.Router = s.setRoutes()
}

func (s *Server) setRoutes() *chi.Mux {

	// Set a context for controllers
	h := api.NewAPICtrl(s.Config, s.Store, s.Cert)

	// Define the router
	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.NotFound(notFoundProblemDetail)

	// Public routes
	// Heartbeat
	r.Group(func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("This is the LCP Server running!"))
		})
	})

	// Status document management
	r.Group(func(r chi.Router) {
		r.Use(render.SetContentType(render.ContentTypeJSON))
		r.Get("/status/{licenseID}", h.StatusDoc)   // Get /status/123
		r.Post("/register/{licenseID}", h.Register) // POST /register/123
		r.Put("/renew/{licenseID}", h.Renew)        // PUT /renew/123
		r.Put("/return/{licenseID}", h.Return)      // PUT /return/123
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
			r.With(paginate).Get("/", h.ListPublications)
			r.With(paginate).Get("/search", h.SearchPublications) // GET /publication/search{?format}
			r.Post("/", h.CreatePublication)                      // POST /publications

			r.Route("/{publicationID}", func(r chi.Router) {
				r.Get("/", h.GetPublication)       // GET /publications/123
				r.Put("/", h.UpdatePublication)    // PUT /publications/123
				r.Delete("/", h.DeletePublication) // DELETE /publications/123
			})
		})

		// LicenseInfo, CRUD
		r.Route("/licenseinfo", func(r chi.Router) {
			r.With(paginate).Get("/", h.ListLicenses)
			r.With(paginate).Get("/search", h.SearchLicenses) // GET /licenses/search{?pub,user,status,count}
			r.Post("/", h.CreateLicense)                      // POST /licenses

			r.Route("/{licenseID}", func(r chi.Router) {
				r.Get("/", h.GetLicense)       // GET /licenses/123
				r.Put("/", h.UpdateLicense)    // PUT /licenses/123
				r.Delete("/", h.DeleteLicense) // DELETE /licenses/123
			})
		})

		// License generation
		r.Route("/licenses/", func(r chi.Router) {
			r.Post("/", h.GenerateLicense) // POST /licenses

			r.Route("/{licenseID}", func(r chi.Router) {
				r.Post("/", h.GetFreshLicense) // POST /licenses/123
			})
		})

		// License revocation
		r.Put("/revoke/{licenseID}", h.Revoke) // PUT /revoke/123

	})

	return r
}

// Run starts the server
func (s *Server) Run(port string) {
	log.Fatal(http.ListenAndServe(port, s.Router))

	//  TODO sort of db.Close()
}

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
