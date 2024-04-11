package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/edrlab/lcp-server/pkg/lic"
	"github.com/google/uuid"
	"syreclabs.com/go/faker"
)

// ---
// License utilities
// ---

// newLicenseRequest generates a random license payload
func newLicenseRequest(pubID string) *LicenseRequest {
	lr := &LicenseRequest{}
	lr.PublicationID = pubID
	lr.UserID = uuid.New().String()
	lr.UserName = faker.Name().Name()
	lr.UserEmail = faker.Internet().Email()
	lr.UserEncrypted = []string{"email", "name"}

	start := time.Now()
	end := start.AddDate(0, 0, 10)
	//print := int32(0)
	//copy := int32(0)

	lr.Start = &start
	lr.End = &end
	//lr.Print = &print
	//lr.Copy = &copy
	lr.Print = nil
	lr.Copy = nil

	lr.Profile = lic.LCP_Basic_Profile
	lr.TextHint = faker.Company().CatchPhrase()
	lr.PassHash = "FAEB00CA518BEA7CB11A7EF31FB6183B489B1B6EADB792BEC64A03B3F6FF80A8"

	return lr
}

// ---
// License Tests
// ---

func TestGenerateLicense(t *testing.T) {

	// create a publication
	inPub, _ := createPublication(t)

	payload := newLicenseRequest(inPub.UUID)
	data, err := json.Marshal((payload))
	if err != nil {
		t.Error("Marshaling payload failed.")
	}

	// visual clue
	//log.Printf("%s \n", string(data))

	// generate a license
	path := "/licenses"
	req, _ := http.NewRequest("POST", path, bytes.NewReader(data))
	response := executeRequest(req)

	// check the response
	if checkResponseCode(t, http.StatusOK, response) {
		var outLic lic.License

		if err := json.Unmarshal(response.Body.Bytes(), &outLic); err != nil {
			t.Fatal(err)
		}

		// pick a data test
		if outLic.User.ID != payload.UserID {
			t.Fatal("Failed to get the same user id.")
		}

		// visual clue
		//log.Printf("%s \n", response.Body.String())

		// delete the license
		deleteLicense(t, outLic.UUID)
	}
}
func TestGetFreshLicense(t *testing.T) {

	// create a license
	inLic, _ := createLicense(t)

	payload := newLicenseRequest(inLic.PublicationID)
	data, err := json.Marshal((payload))
	if err != nil {
		t.Error("Marshaling payload failed.")
	}

	// visual clue
	//log.Printf("%s \n", string(data))

	// get the license
	path := "/licenses/" + inLic.UUID
	req, _ := http.NewRequest("POST", path, bytes.NewReader(data))
	response := executeRequest(req)

	// check the response
	if checkResponseCode(t, http.StatusOK, response) {
		var outLic lic.License

		if err := json.Unmarshal(response.Body.Bytes(), &outLic); err != nil {
			t.Fatal(err)
		}

		if outLic.UUID != inLic.UUID {
			t.Fatal("Failed to get the same uuid.")
		}

		// visual clue
		//log.Printf("%s \n", response.Body.String())
	}

	// delete the license
	deleteLicense(t, inLic.UUID)
}
