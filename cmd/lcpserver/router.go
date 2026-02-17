// Copyright 2025 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/go-chi/render"

	"github.com/edrlab/lcp-server/pkg/api"
)

func (s *Server) setRoutes() *chi.Mux {

	// Set api controller dependencies
	a := api.NewAPICtrl(s.Config, s.Store, s.Cert)

	// Define the router
	r := chi.NewRouter()

	// Recovery middleware
	r.Use(middleware.Recoverer)

	// Heartbeat (excluded from logs)
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("The LCP Server is running!"))
	})

	// Group for all other routes
	r.Group(func(r chi.Router) {
		// Logger middleware
		r.Use(middleware.Logger)

		r.NotFound(notFoundProblemDetail)

		// CORS Configuration
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{"http://localhost:8090", "http://localhost:8091"}, // URLs of the React frontend
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: true,
			MaxAge:           300, // Maximum value not ignored by any of major browsers
		}))

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
				r.With(paginate).Get("/", a.ListPublications)         // GET /publications/
				r.With(paginate).Get("/search", a.SearchPublications) // GET /publications/search{?format}
				r.Post("/", a.CreatePublication)                      // POST /publications

				r.Route("/{publicationID}", func(r chi.Router) {
					r.Get("/", a.GetPublication)       // GET /publications/123
					r.Put("/", a.UpdatePublication)    // PUT /publications/123
					r.Delete("/", a.DeletePublication) // DELETE /publications/123
				})
				// get publication by AltID
				r.Get("/altid/{altID}", a.GetPublicationByAltID) // GET /publications/altid/alt123	
			})

			// LicenseInfo, CRUD
			r.Route("/licenseinfo", func(r chi.Router) {
				r.With(paginate).Get("/", a.ListLicenses)         // GET /licenseinfo/
				r.With(paginate).Get("/search", a.SearchLicenses) // GET /licenseinfo/search{?pub,user,status,count}
				r.Post("/", a.CreateLicense)                      // POST /licenseinfo

				r.Route("/{licenseID}", func(r chi.Router) {
					r.Get("/", a.GetLicense)       // GET /licenseinfo/123
					r.Put("/", a.UpdateLicense)    // PUT /licenseinfo/123
					r.Delete("/", a.DeleteLicense) // DELETE /licenseinfo	/123
				})
			})

			// License events
			r.Route("/license-events/{licenseID}", func(r chi.Router) {
				r.Get("/", a.ListLicenseEvents) // GET /license-events/123
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

		// Dashboard data
		r.Post("/dashdata/login", Login(s.Config)) // POST /dashdata/login
		// Require JWT Authentication
		r.Group(func(r chi.Router) {
			r.Use(AuthMiddleware(s.Config))
			r.Use(render.SetContentType(render.ContentTypeJSON))
			r.Route("/dashdata", func(r chi.Router) {
				r.Get("/data", a.GetDashboardData)            // GET /dashdata/data
				r.Get("/overshared", a.GetOversharedLicenses) // GET /dashdata/overshared
				r.Put("/revoke/{licenseID}", a.Revoke)        // PUT /dashdata/revoke/license123
				// these dashboard routes allow alt authentication before accessing crud functions
				r.With(paginate).Get("/publications", a.ListPublications) 		// GET /dashdata/publications
				r.Delete("/publications/{publicationID}", a.DeletePublication) // DELETE /dashdata/publication/publication123
				r.With(paginate).Get("/user-licenses/{userID}", a.ListUserLicenses) 			// GET /dashdata/user-licenses/user123
				r.Get("/license-events/{licenseID}", a.ListLicenseEvents) 			// GET /dashdata/license-events/license123
			})
		})

	})

	return r
}

// paginate middleware
func paginate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// default values
		page := 1
		perPage := 20

		// read query parameters
		q := r.URL.Query()
		if p := q.Get("page"); p != "" {
			if val, err := strconv.Atoi(p); err == nil && val > 0 {
				page = val
			}
		}
		if pp := q.Get("per_page"); pp != "" {
			if val, err := strconv.Atoi(pp); err == nil && val > 0 {
				perPage = val
			}
		}

		// add to context
		ctx := context.WithValue(r.Context(), api.PageKey, page)
		ctx = context.WithValue(ctx, api.PerPageKey, perPage)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// notFoundProblemDetail formats not found errors as problem details, for the sake of consistency.
func notFoundProblemDetail(w http.ResponseWriter, r *http.Request) {
	response := map[string]string{"type": "about:blank", "title": "Endpoint not found."}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotFound)

	json.NewEncoder(w).Encode(response)
}
