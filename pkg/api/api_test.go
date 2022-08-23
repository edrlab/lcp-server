package api

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/edrlab/lcp-server/pkg/conf"
	"github.com/edrlab/lcp-server/pkg/stor"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"
)

// Server context
type Server struct {
	Config *conf.Config
	stor.Store
	Cert   *tls.Certificate
	Router *chi.Mux
}

// s is the server variable shared by all tests
var s Server

// PublicationTest data model
type PublicationTest struct {
	UUID          string `json:"uuid"`
	Title         string `json:"title"`
	EncryptionKey []byte `json:"encryption_key"`
	Location      string `json:"location"`
	ContentType   string `json:"content_type"`
	Size          uint32 `json:"size"`
	Checksum      string `json:"checksum"`
}

// LicenseTest data model
type LicenseTest struct {
	Issued        *time.Time `json:"issued,omitempty"`
	UUID          string     `json:"uuid"`
	UserID        string     `json:"user_id"`
	PublicationID string     `json:"publication_id"`
	Provider      string     `json:"provider"`
	Start         *time.Time `json:"start,omitempty"`
	End           *time.Time `json:"end,omitempty"`
	Copy          int32      `json:"copy,omitempty"`
	Print         int32      `json:"print,omitempty"`
	Status        string     `json:"status"`
	StatusUpdated *time.Time `json:"status_updated,omitempty"`
	DeviceCount   int        `json:"device_count"`
}

// ---
// Utilities
// ---
func setConfig() *conf.Config {

	c := conf.Config{
		Dsn: "sqlite3://file::memory:?cache=shared",
		Login: conf.Login{
			User:     "user",
			Password: "password",
		},
		Certificate: conf.Certificate{
			Cert:       "../test/cert/cert-edrlab-test.pem",
			PrivateKey: "../test/cert/privkey-edrlab-test.pem",
		},
		License: conf.License{
			Provider: "http://edrlab.org",
			Profile:  "http://readium.org/lcp/basic-profile",
			Links: map[string]string{
				"status": "http://localhost/status/{license_id}",
				"hint":   "https://www.edrlab.org/lcp-help/{license_id}",
			},
		},
	}

	return &c
}

func executeRequest(req *http.Request) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	s.Router.ServeHTTP(rr, req)

	return rr
}

func checkResponseCode(t *testing.T, expected int, response *httptest.ResponseRecorder) bool {
	ok := true
	if expected != response.Code {
		t.Errorf("Expected response code %d. Got %d\n", expected, response.Code)
		t.Log(response.Body.String())
		ok = false
	}
	return ok
}

// ---
// Main Test
// ---

func TestMain(m *testing.M) {

	s.Config = setConfig()

	// Setup the database
	var err error
	s.Store, err = stor.DBSetup(s.Config.Dsn)
	if err != nil {
		panic("Database setup failed")
	}

	// Setup the X509 certificate
	var certFile, privKeyFile string
	if certFile = s.Config.Certificate.Cert; certFile == "" {
		panic("Must specify a certificate")
	}
	if privKeyFile = s.Config.Certificate.PrivateKey; privKeyFile == "" {
		panic("Must specify a private key")
	}
	cert, err := tls.LoadX509KeyPair(certFile, privKeyFile)
	if err != nil {
		panic(err)
	}
	s.Cert = &cert

	// Set a context for handlers
	h := NewHandlerCtx(s.Config, s.Store, s.Cert)

	// Define the router
	r := chi.NewRouter()

	s.Router = r

	r.Use(middleware.RequestID)
	//r.Use(middleware.Logger)
	r.Use(middleware.URLFormat)

	// Only public routes for these tests
	r.Group(func(r chi.Router) {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("This is the LCP Server running!"))
		})
	})

	r.Group(func(r chi.Router) {
		r.Use(render.SetContentType(render.ContentTypeJSON))

		// Publications
		r.Route("/publications", func(r chi.Router) {
			r.Get("/", h.ListPublications)
			r.Get("/search", h.SearchPublications) // GET /publication/search{?format}
			r.Post("/", h.CreatePublication)       // POST /publications

			r.Route("/{publicationID}", func(r chi.Router) {
				r.Get("/", h.GetPublication)       // GET /publications/123
				r.Put("/", h.UpdatePublication)    // PUT /publications/123
				r.Delete("/", h.DeletePublication) // DELETE /publications/123
			})
		})

		// LicenseInfo, CRUD
		r.Route("/licenseinfo", func(r chi.Router) {
			r.Get("/", h.ListLicenses)
			r.Get("/search", h.SearchLicenses) // GET /licenses/search{?pub,user,status,count}
			r.Post("/", h.CreateLicense)       // POST /licenses

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

	})

	code := m.Run()
	os.Exit(code)
}
