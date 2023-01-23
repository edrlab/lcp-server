package api

import (
	"encoding/json"
	"log"
	"net/http"
	"testing"

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
