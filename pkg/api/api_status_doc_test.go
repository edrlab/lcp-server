package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/edrlab/lcp-server/pkg/lic"
)

// ---
// StatusDoc Tests
// ---

func TestGetStatusDoc(t *testing.T) {

	// create a license
	inLic, _ := createLicense(t)

	// get the associated status doc
	path := "/status/" + inLic.UUID
	req, _ := http.NewRequest("GET", path, nil)
	response := executeRequest(req)

	// check the response
	if checkResponseCode(t, http.StatusOK, response) {
		var statusDoc lic.StatusDoc

		if err := json.Unmarshal((response.Body.Bytes()), &statusDoc); err != nil {
			t.Fatal(err)
		}
		// visual clue
		log.Printf("%s\n", response.Body.String())
	}

	// delete the license
	deleteLicense(t, inLic.UUID)
}

func TestRegister(t *testing.T) {

	// create a license
	inLic, _ := createLicense(t)

	// send a register command
	path := "/register/" + inLic.UUID + "?id=1&name=device1"
	req, _ := http.NewRequest("POST", path, nil)
	response := executeRequest(req)

	// check the response
	if checkResponseCode(t, http.StatusOK, response) {
		var statusDoc lic.StatusDoc

		if err := json.Unmarshal((response.Body.Bytes()), &statusDoc); err != nil {
			t.Fatal(err)
		}
		// visual clue
		log.Printf("%s\n", response.Body.String())
	}

	// delete the license
	deleteLicense(t, inLic.UUID)
}

func TestRenew(t *testing.T) {

	// create a license
	inLic, _ := createLicense(t)

	// send a register command
	path := "/register/" + inLic.UUID + "?id=1&name=device1"
	req, _ := http.NewRequest("POST", path, nil)
	response := executeRequest(req)

	// check the response
	if checkResponseCode(t, http.StatusOK, response) {
		var statusDoc lic.StatusDoc

		if err := json.Unmarshal((response.Body.Bytes()), &statusDoc); err != nil {
			t.Fatal(err)
		}
	}

	// calculate now + 13 days
	newEnd := time.Now().Add(13 * 24 * time.Hour).Format(time.RFC3339)

	// send a renew command
	path = "/renew/" + inLic.UUID + "?id=1&name=device1&end=" + url.QueryEscape(newEnd)

	req, _ = http.NewRequest("PUT", path, nil)
	response = executeRequest(req)

	// check the response
	if checkResponseCode(t, http.StatusOK, response) {
		var statusDoc lic.StatusDoc

		if err := json.Unmarshal((response.Body.Bytes()), &statusDoc); err != nil {
			t.Fatal(err)
		}
		// visual clue
		log.Printf("%s\n", response.Body.String())
	}

	// delete the license
	deleteLicense(t, inLic.UUID)
}

func TestReturn(t *testing.T) {

	// create a license
	inLic, _ := createLicense(t)

	// send a register command
	path := "/register/" + inLic.UUID + "?id=1&name=device1"
	req, _ := http.NewRequest("POST", path, nil)
	response := executeRequest(req)

	// check the response
	if checkResponseCode(t, http.StatusOK, response) {
		var statusDoc lic.StatusDoc

		if err := json.Unmarshal((response.Body.Bytes()), &statusDoc); err != nil {
			t.Fatal(err)
		}
	}

	// send a return command
	path = "/return/" + inLic.UUID + "?id=1&name=device1"
	req, _ = http.NewRequest("PUT", path, nil)
	response = executeRequest(req)

	// check the response
	if checkResponseCode(t, http.StatusOK, response) {
		var statusDoc lic.StatusDoc

		if err := json.Unmarshal((response.Body.Bytes()), &statusDoc); err != nil {
			t.Fatal(err)
		}
		// visual clue
		log.Printf("%s\n", response.Body.String())
	}

	// delete the license
	deleteLicense(t, inLic.UUID)
}

func TestRevoke(t *testing.T) {

	// create a license
	inLic, _ := createLicense(t)

	// send a register command
	path := "/register/" + inLic.UUID + "?id=1&name=device1"
	req, _ := http.NewRequest("POST", path, nil)
	response := executeRequest(req)

	// check the response
	if checkResponseCode(t, http.StatusOK, response) {
		var statusDoc lic.StatusDoc

		if err := json.Unmarshal((response.Body.Bytes()), &statusDoc); err != nil {
			t.Fatal(err)
		}
	}

	// send a return command
	path = "/revoke/" + inLic.UUID
	req, _ = http.NewRequest("PUT", path, nil)
	response = executeRequest(req)

	// check the response
	if checkResponseCode(t, http.StatusOK, response) {
		var statusDoc lic.StatusDoc

		if err := json.Unmarshal((response.Body.Bytes()), &statusDoc); err != nil {
			t.Fatal(err)
		}
		// visual clue
		log.Printf("%s\n", response.Body.String())
	}

	// delete the license
	deleteLicense(t, inLic.UUID)
}
