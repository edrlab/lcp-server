// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

// LCP Server generates LCP licenses.
package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	"github.com/edrlab/lcp-server/pkg/api"
	"github.com/edrlab/lcp-server/pkg/stor"
)

// Server context
type Server struct {
	Config *Config
	Store  stor.Store
	Router *chi.Mux
}

// Server configuration
type Config struct {
	Dsn   string
	Login struct {
		User     string
		Password string
	}
}

func main() {

	s := Server{}

	// test config (later in a file)
	c := Config{}
	c.Dsn = "sqlite3://file::memory:?cache=shared"
	c.Login.User = "user"
	c.Login.Password = "password"
	s.Config = &c

	s.Initialize()

	s.Run(":8081")
}

// Initialize sets up the database and routes
func (s *Server) Initialize() {
	var err error

	// Setup the database
	s.Store, err = stor.DBSetup(s.Config.Dsn)
	if err != nil {
		panic("database setup failed.")
	}

	// Setup the routes
	s.Router = s.setRoutes()
}

func (s *Server) setRoutes() *chi.Mux {

	// Set a context for handlers
	h := api.NewHandlerCtx(s.Store)

	// Define the router
	r := chi.NewRouter()

	//r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	//r.Use(middleware.URLFormat)
	//r.Use(render.SetContentType(render.ContentTypeJSON))

	// Public routes
	r.Group(func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("This is the LCP Server running!"))
		})
	})

	// Private Routes
	// Require Authentication
	credentials := make(map[string]string)
	credentials[s.Config.Login.User] = s.Config.Login.Password

	r.Group(func(r chi.Router) {
		r.Use(middleware.BasicAuth("restricted", credentials))
		r.Use(render.SetContentType(render.ContentTypeJSON))

		// Publications
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

		// Licenses
		r.Route("/licenses", func(r chi.Router) {
			r.With(paginate).Get("/", h.ListLicenses)
			r.With(paginate).Get("/search", h.SearchLicenses) // GET /licenses/search{?pub,user,status,count}
			r.Post("/", h.CreateLicense)                      // POST /licenses

			r.Route("/{licenseID}", func(r chi.Router) {
				r.Get("/", h.GetLicense)       // GET /licenses/123
				r.Put("/", h.UpdateLicense)    // PUT /licenses/123
				r.Delete("/", h.DeleteLicense) // DELETE /licenses/123
			})
		})

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
