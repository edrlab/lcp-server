package api

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/json"
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
	"github.com/google/uuid"
	"syreclabs.com/go/faker"
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

// LicenseTest data model, no gorm data, no join
type LicenseTest struct {
	CreatedAt     time.Time  `json:"created_at"`
	Updated       *time.Time `json:"updated,omitempty"`
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
// Utilities - Config
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

// ---
// Utilities - Publications
// ---

// generates a random publication object
func newPublication() *PublicationTest {
	pub := &PublicationTest{}
	pub.UUID = uuid.New().String()
	pub.Title = faker.Company().CatchPhrase()
	pub.EncryptionKey = make([]byte, 16)
	rand.Read(pub.EncryptionKey)
	pub.Location = faker.Internet().Url()
	pub.ContentType = "application/epub+zip"
	pub.Size = uint32(faker.Number().NumberInt(5))
	pub.Checksum = faker.Lorem().Characters(16)

	return pub
}

func createPublication(t *testing.T) (*PublicationTest, *httptest.ResponseRecorder) {

	pub := newPublication()
	data, err := json.Marshal((pub))
	if err != nil {
		t.Error("Marshaling Publication failed.")
	}

	// visual clue
	//log.Printf("%s \n", string(data))

	path := "/publications/"
	req, _ := http.NewRequest("POST", path, bytes.NewReader(data))
	return pub, executeRequest(req)
}

func deletePublication(t *testing.T, uuid string) *httptest.ResponseRecorder {

	// delete the publication
	path := "/publications/" + uuid
	req, _ := http.NewRequest("DELETE", path, nil)
	return executeRequest(req)
}

// ---
// Utilities - Licenses
// ---

// global license counter
var LicenseCounter int

// newLicense generates a random license info object
func newLicense(pubID string) *LicenseTest {
	lic := &LicenseTest{}
	now := time.Now()
	lic.CreatedAt = now
	lic.UUID = uuid.New().String()
	lic.UserID = uuid.New().String()
	lic.PublicationID = pubID
	lic.Provider = faker.Internet().Url()
	ts := faker.Time().Backward(3600)
	lic.Start = &ts
	te := faker.Time().Forward(3600 * 24)
	lic.End = &te
	lic.Copy = 10000
	lic.Print = 100
	lic.Status = stor.STATUS_READY
	if LicenseCounter%5 == 0 {
		lic.DeviceCount = LicenseCounter
	} else {
		lic.DeviceCount = faker.Number().NumberInt(3)
	}

	LicenseCounter++

	return lic
}

// createLicense generates a random license via the API
func createLicense(t *testing.T) (*LicenseTest, *httptest.ResponseRecorder) {

	// create a publication
	inPub, _ := createPublication(t)

	lic := newLicense(inPub.UUID)
	data, err := json.Marshal((lic))
	if err != nil {
		t.Error("Marshaling Publication failed.")
	}

	// visual clue
	//log.Printf("%s \n", string(data))

	path := "/licenseinfo"
	req, _ := http.NewRequest("POST", path, bytes.NewReader(data))
	return lic, executeRequest(req)
}

// deleteLicense suppresses a license via the API
func deleteLicense(t *testing.T, uuid string) {

	// delete the license
	path := "/licenseinfo/" + uuid
	req, _ := http.NewRequest("DELETE", path, nil)
	response := executeRequest(req)

	// check the response
	if checkResponseCode(t, http.StatusOK, response) {
		var outLic LicenseTest

		if err := json.Unmarshal(response.Body.Bytes(), &outLic); err != nil {
			t.Fatal(err)
		}
		// delete the corresponding publication
		deletePublication(t, outLic.PublicationID)
	}

}

// ---
// Utilities - Requests and Responses
// ---

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
	h := NewAPIHandler(s.Config, s.Store, s.Cert)

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

		// Status document management
		r.Group(func(r chi.Router) {
			r.Use(render.SetContentType(render.ContentTypeJSON))
			r.Get("/status/{licenseID}", h.StatusDoc)   // Get /status/123
			r.Post("/register/{licenseID}", h.Register) // POST /register/123
			r.Put("/return/{licenseID}", h.Return)      // PUT /return/123
			r.Put("/renew/{licenseID}", h.Renew)        // PUT /renew/123
		})

	})

	code := m.Run()
	os.Exit(code)
}
