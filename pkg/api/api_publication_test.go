package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// ---
// Publication utilities
// ---

func comparePublications(inPub *PublicationTest, outPub *PublicationTest) bool {

	// check the values (we cannot directly compare the structs because of the bytes field)
	if outPub.UUID != inPub.UUID ||
		outPub.Title != inPub.Title ||
		!bytes.Equal(outPub.EncryptionKey, inPub.EncryptionKey) ||
		outPub.Href != inPub.Href ||
		outPub.ContentType != inPub.ContentType ||
		outPub.Size != inPub.Size ||
		outPub.Checksum != inPub.Checksum {
		return false
	}
	return true
}

// ---
// Publication Tests
// ---

func TestEmptyPublicationTable(t *testing.T) {

	// get a list of publications in an empty db
	req, _ := http.NewRequest("GET", "/publications", nil)
	response := executeRequest(req)

	if checkResponseCode(t, http.StatusOK, response) {
		if body := response.Body.String(); strings.TrimSpace(body) != "[]" {
			t.Errorf("Expected an empty array. Got %s", body)
		}
	}

}

func TestCreatePublication(t *testing.T) {

	// create a publication
	inPub, response := createPublication(t)

	// check the response
	if checkResponseCode(t, http.StatusCreated, response) {
		var outPub PublicationTest

		if err := json.Unmarshal((response.Body.Bytes()), &outPub); err != nil {
			t.Fatal(err)
		}

		same := comparePublications(inPub, &outPub)
		if !same {
			t.Error("Failed to get the same content back")
		}
	} else {
		t.Log(response.Body.String())
	}

	// delete the publication
	deletePublication(t, inPub.UUID)
}

func TestGetPublication(t *testing.T) {

	// create a publication
	inPub, _ := createPublication(t)

	// get the publication
	path := "/publications/" + inPub.UUID
	req, _ := http.NewRequest("GET", path, nil)
	response := executeRequest(req)

	// check the response
	if checkResponseCode(t, http.StatusOK, response) {
		var outPub PublicationTest

		if err := json.Unmarshal(response.Body.Bytes(), &outPub); err != nil {
			t.Fatal(err)
		}

		same := comparePublications(inPub, &outPub)
		if !same {
			t.Error("Failed to get the same content back")
		}
	}

	// delete the publication
	deletePublication(t, inPub.UUID)
}

func TestUpdatePublication(t *testing.T) {

	// create a publication
	inPub, _ := createPublication(t)

	// update a field
	inPub.Title = "Updated title"

	data, err := json.Marshal((inPub))
	if err != nil {
		t.Error("Marshaling Publication failed.")
	}

	path := "/publications/" + inPub.UUID
	req, _ := http.NewRequest("PUT", path, bytes.NewReader(data))
	response := executeRequest(req)

	if checkResponseCode(t, http.StatusOK, response) {

		var outPub PublicationTest

		if err := json.Unmarshal(response.Body.Bytes(), &outPub); err != nil {
			t.Fatal(err)
		}

		same := comparePublications(inPub, &outPub)
		if !same {
			t.Error("Failed to get the same content back")
		}
	}

	// delete the publication
	deletePublication(t, inPub.UUID)
}

func TestDeletePublication(t *testing.T) {

	// create a publication
	inPub, _ := createPublication(t)

	// delete the publication
	path := "/publications/" + inPub.UUID
	req, _ := http.NewRequest("DELETE", path, nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusOK, response)
}

func TestListPublications(t *testing.T) {

	var inPubs []*PublicationTest
	// create some publications
	for i := 0; i < 10; i++ {
		pub, _ := createPublication(t)
		inPubs = append(inPubs, pub)
	}

	// get the list of publications
	path := "/publications/"
	req, _ := http.NewRequest("GET", path, nil)
	response := executeRequest(req)

	if checkResponseCode(t, http.StatusOK, response) {
		var list []PublicationTest

		if err := json.Unmarshal(response.Body.Bytes(), &list); err != nil {
			t.Fatal(err)
		}

		if len(list) != len(inPubs) {
			t.Error("Failed to get the same list size")
			return
		}
		for idx, outPub := range list {
			same := comparePublications(inPubs[idx], &outPub)
			if !same {
				t.Error("Failed to get the same content back")
			}
		}
	}

	// delete the publications
	for _, pub := range inPubs {
		deletePublication(t, pub.UUID)
	}
}

func TestSearchPublications(t *testing.T) {

	var inPubs []*PublicationTest
	// create some epub publications
	for i := 0; i < 10; i++ {
		pub, _ := createPublication(t)
		inPubs = append(inPubs, pub)
	}
	// create an lcpdf publication
	lastpub := newPublication()
	lastpub.ContentType = "application/pdf+lcp"
	data, _ := json.Marshal((lastpub))
	// create the publication
	path := "/publications/"
	req, _ := http.NewRequest("POST", path, bytes.NewReader(data))
	executeRequest(req)

	// search publications by format
	formats := []string{"epub", "lcpdf", "lcpau", "lcpdi", "unknown"}
	for _, format := range formats {
		path = "/publications/search"
		req, _ = http.NewRequest("GET", path, nil)
		q := req.URL.Query()
		q.Add("format", format)
		req.URL.RawQuery = q.Encode()
		response := executeRequest(req)

		switch format {
		case "epub":
			if checkResponseCode(t, http.StatusOK, response) {
				var list []PublicationTest

				if err := json.Unmarshal(response.Body.Bytes(), &list); err != nil {
					t.Fatal(err)
				}

				if len(list) != len(inPubs) {
					t.Error("Failed to get the same list size")
					return
				}
				for idx, outPub := range list {
					same := comparePublications(inPubs[idx], &outPub)
					if !same {
						t.Error("Failed to get the same content back")
					}
				}

			}

		case "lcpdf":
			var list []PublicationTest
			if checkResponseCode(t, http.StatusOK, response) {
				if err := json.Unmarshal(response.Body.Bytes(), &list); err != nil {
					t.Fatal(err)
				}
				if len(list) != 1 {
					t.Error("Failed to get an lcpdf back")
				}
			}
		case "lcpau", "lcpdi":
			checkResponseCode(t, http.StatusOK, response)
		case "unknown":
			checkResponseCode(t, http.StatusBadRequest, response)
		}
	}

	// delete the publications
	for _, pub := range inPubs {
		deletePublication(t, pub.UUID)
	}
	deletePublication(t, lastpub.UUID)

}

func TestDeleteNoExistingPublication(t *testing.T) {

	path := "/publications/" + uuid.New().String()

	req, _ := http.NewRequest("DELETE", path, nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusNotFound, response)
}

func TestGetDeletedPublication(t *testing.T) {

	// create a publication
	inPub, _ := createPublication(t)

	// delete the publication
	deletePublication(t, inPub.UUID)

	// try to get the deleted publication
	path := "/publications/" + inPub.UUID
	req, _ := http.NewRequest("GET", path, nil)
	response := executeRequest(req)

	checkResponseCode(t, http.StatusNotFound, response)
}
