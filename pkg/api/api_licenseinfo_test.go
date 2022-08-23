package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"syreclabs.com/go/faker"
)

// ---
// License utilities
// ---

// global license countre
var LicenseCounter int

// generates a random license info object
func newLicense(pubID string) *LicenseTest {
	lic := &LicenseTest{}
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
	tsu := faker.Time().Backward(3600)
	lic.StatusUpdated = &tsu
	if LicenseCounter%5 == 0 {
		lic.DeviceCount = LicenseCounter
	} else {
		lic.DeviceCount = faker.Number().NumberInt(3)
	}

	LicenseCounter++

	return lic
}

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
// License Info Tests
// ---

func TestEmptyLicenseInfoTable(t *testing.T) {

	// get a list of publications in an empty db
	req, _ := http.NewRequest("GET", "/licenseinfo", nil)
	response := executeRequest(req)

	if checkResponseCode(t, http.StatusOK, response) {
		if body := response.Body.String(); strings.TrimSpace(body) != "[]" {
			t.Errorf("Expected an empty array. Got %s", body)

		}

	}
}

func TestCreateLicenseInfo(t *testing.T) {

	// create a license
	inLic, response := createLicense(t)

	// check the response
	if checkResponseCode(t, http.StatusOK, response) {
		var outLic LicenseTest

		if err := json.Unmarshal((response.Body.Bytes()), &outLic); err != nil {
			t.Fatal(err)
		}
	}

	// delete the license
	deleteLicense(t, inLic.UUID)
}

func TestGetLicenseInfo(t *testing.T) {

	// create a license
	inLic, _ := createLicense(t)

	// get the license
	path := "/licenseinfo/" + inLic.UUID
	req, _ := http.NewRequest("GET", path, nil)
	response := executeRequest(req)

	// check the response
	if checkResponseCode(t, http.StatusOK, response) {
		var outLic LicenseTest

		if err := json.Unmarshal(response.Body.Bytes(), &outLic); err != nil {
			t.Fatal(err)
		}
	}

	// delete the license
	deleteLicense(t, inLic.UUID)
}

func TestUpdateLicenseInfo(t *testing.T) {

	// create a license
	inLic, _ := createLicense(t)

	// update a field
	inLic.Status = "revoked"

	data, err := json.Marshal((inLic))
	if err != nil {
		t.Error("Marshaling License failed.")
	}

	path := "/licenseinfo/" + inLic.UUID
	req, _ := http.NewRequest("PUT", path, bytes.NewReader(data))
	response := executeRequest(req)

	if checkResponseCode(t, http.StatusOK, response) {
		var outLic LicenseTest

		if err := json.Unmarshal(response.Body.Bytes(), &outLic); err != nil {
			t.Fatal(err)
		}
	} else {
		t.Error("Updating License failed.")
	}

	// delete the license
	deleteLicense(t, inLic.UUID)

}

func TestDeleteLicense(t *testing.T) {

	// create a license
	inLic, _ := createLicense(t)

	// delete the license
	path := "/licenseinfo/" + inLic.UUID
	req, _ := http.NewRequest("DELETE", path, nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusOK, response)

	// delete the corresponding publication
	deletePublication(t, inLic.PublicationID)
}

func TestListLicenses(t *testing.T) {

	var inLics []*LicenseTest
	// create some licenses
	for i := 0; i < 10; i++ {
		lic, _ := createLicense(t)
		inLics = append(inLics, lic)
	}

	// get the list of licenses
	path := "/licenseinfo/"
	req, _ := http.NewRequest("GET", path, nil)
	response := executeRequest(req)

	if checkResponseCode(t, http.StatusOK, response) {
		var list []LicenseTest

		if err := json.Unmarshal(response.Body.Bytes(), &list); err != nil {
			t.Fatal(err)
		}
	}

	// delete the licenses
	for _, lic := range inLics {
		deleteLicense(t, lic.UUID)
	}
}
func TestSearchLicensesByUser(t *testing.T) {

	var inLics []*LicenseTest
	// create some licenses
	for i := 0; i < 2; i++ {
		lic, _ := createLicense(t)
		inLics = append(inLics, lic)
	}

	// get the list of licenses
	path := "/licenseinfo/search"
	req, _ := http.NewRequest("GET", path, nil)
	q := req.URL.Query()
	q.Add("user", inLics[1].UserID)
	req.URL.RawQuery = q.Encode()
	response := executeRequest(req)

	if checkResponseCode(t, http.StatusOK, response) {
		var list []LicenseTest

		if err := json.Unmarshal(response.Body.Bytes(), &list); err != nil {
			t.Fatal(err)
		}

		if len(list) != 1 {
			t.Errorf("Expected 1 license back, got %d", len(list))
		}
	}

	// delete the licenses
	for _, lic := range inLics {
		deleteLicense(t, lic.UUID)
	}
}

func TestSearchLicensesByPublication(t *testing.T) {

	var inLics []*LicenseTest
	// create some licenses
	for i := 0; i < 2; i++ {
		lic, _ := createLicense(t)
		inLics = append(inLics, lic)
	}

	// get the list of licenses
	path := "/licenseinfo/search"
	req, _ := http.NewRequest("GET", path, nil)
	q := req.URL.Query()
	q.Add("pub", inLics[0].PublicationID)
	req.URL.RawQuery = q.Encode()
	response := executeRequest(req)

	if checkResponseCode(t, http.StatusOK, response) {
		var list []LicenseTest

		if err := json.Unmarshal(response.Body.Bytes(), &list); err != nil {
			t.Fatal(err)
		}

		if len(list) != 1 {
			t.Errorf("Expected 1 license back, got %d", len(list))
		}
	}

	// delete the licenses
	for _, lic := range inLics {
		deleteLicense(t, lic.UUID)
	}
}

func TestSearchLicensesByStatus(t *testing.T) {

	var inLics []*LicenseTest
	// create some licenses
	for i := 0; i < 3; i++ {
		lic, _ := createLicense(t)
		inLics = append(inLics, lic)
	}

	// get the list of licenses
	path := "/licenseinfo/search"
	req, _ := http.NewRequest("GET", path, nil)
	q := req.URL.Query()
	q.Add("status", "ready")
	req.URL.RawQuery = q.Encode()
	response := executeRequest(req)

	if checkResponseCode(t, http.StatusOK, response) {
		var list []LicenseTest

		if err := json.Unmarshal(response.Body.Bytes(), &list); err != nil {
			t.Fatal(err)
		}

		if len(list) != 3 {
			t.Errorf("Expected 3 licenses back, got %d", len(list))
		}
	}

	// delete the licenses
	for _, lic := range inLics {
		deleteLicense(t, lic.UUID)
	}
}

func TestSearchLicensesByCount(t *testing.T) {

	var inLics []*LicenseTest
	// create some licenses
	for i := 0; i < 20; i++ {
		lic, _ := createLicense(t)
		inLics = append(inLics, lic)
	}

	// get the list of licenses
	path := "/licenseinfo/search"
	req, _ := http.NewRequest("GET", path, nil)
	q := req.URL.Query()
	q.Add("count", "1:50")
	req.URL.RawQuery = q.Encode()
	response := executeRequest(req)

	if checkResponseCode(t, http.StatusOK, response) {
		var list []LicenseTest

		if err := json.Unmarshal(response.Body.Bytes(), &list); err != nil {
			t.Fatal(err)
		}

		if len(list) < 3 {
			t.Errorf("Expected 3 or 4 licenses back, got %d", len(list))
		}
	}

	// delete the licenses
	for _, lic := range inLics {
		deleteLicense(t, lic.UUID)
	}
}

func TestSearchLicensesByCountWithError(t *testing.T) {

	var inLics []*LicenseTest
	// create some licenses
	for i := 0; i < 2; i++ {
		lic, _ := createLicense(t)
		inLics = append(inLics, lic)
	}

	// get the list of licenses
	path := "/licenseinfo/search"
	req, _ := http.NewRequest("GET", path, nil)
	q := req.URL.Query()
	q.Add("count", "1-50")
	req.URL.RawQuery = q.Encode()
	response := executeRequest(req)

	checkResponseCode(t, http.StatusBadRequest, response)

	// delete the licenses
	for _, lic := range inLics {
		deleteLicense(t, lic.UUID)
	}
}

func TestDeleteNoExistingLicense(t *testing.T) {

	path := "/licenseinfo/" + uuid.New().String()

	req, _ := http.NewRequest("DELETE", path, nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusNotFound, response)
}
