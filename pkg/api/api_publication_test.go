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
	for i := 0; i < 10; i++ {
		pub, response := createPublication(t)
		if !checkResponseCode(t, http.StatusCreated, response) {
			t.Fatal("Failed to create publication")
		}
		t.Cleanup(func() {
			checkResponseCode(t, http.StatusOK, deletePublication(t, pub.UUID))
		})
		inPubs = append(inPubs, pub)
	}

	req, err := http.NewRequest("GET", "/publications/", nil)
	if err != nil {
		t.Fatal(err)
	}
	response := executeRequest(req)
	if !checkResponseCode(t, http.StatusOK, response) {
		return
	}

	var list []PublicationTest
	if err := json.Unmarshal(response.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}

	if len(list) != len(inPubs) {
		t.Fatalf("Expected %d publications, got %d", len(inPubs), len(list))
	}
	for idx, outPub := range list {
		expected := inPubs[len(inPubs)-1-idx]
		if !comparePublications(expected, &outPub) {
			t.Errorf("Publication at index %d: expected UUID %s and matching content, got UUID %s", idx, expected.UUID, outPub.UUID)
		}
	}
}

func TestSearchPublications(t *testing.T) {
	publications := make(map[string][]*PublicationTest)
	create := func(contentType string) {
		t.Helper()
		pub := newPublication()
		pub.ContentType = contentType
		data, err := json.Marshal(pub)
		if err != nil {
			t.Fatal(err)
		}
		req, err := http.NewRequest("POST", "/publications/", bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		response := executeRequest(req)
		if !checkResponseCode(t, http.StatusCreated, response) {
			t.Fatal("Failed to create publication")
		}
		t.Cleanup(func() {
			checkResponseCode(t, http.StatusOK, deletePublication(t, pub.UUID))
		})
		publications[contentType] = append(publications[contentType], pub)
	}

	for i := 0; i < 10; i++ {
		create("application/epub+zip")
	}
	create("application/pdf")
	create("application/pdf+lcp")
	create("application/audiobook+lcp")

	tests := []struct {
		format      string
		contentType string
		status      int
	}{
		{"epub", "application/epub+zip", http.StatusOK},
		{"pdf", "application/pdf", http.StatusOK},
		{"lcpdf", "application/pdf+lcp", http.StatusOK},
		{"lcpa", "application/audiobook+lcp", http.StatusOK},
		{"lcpau", "application/audiobook+lcp", http.StatusOK},
		{"lcpdi", "application/divina+lcp", http.StatusOK},
		{"unknown", "", http.StatusBadRequest},
		{"", "", http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.format, func(t *testing.T) {
			req, err := http.NewRequest("GET", "/publications/search", nil)
			if err != nil {
				t.Fatal(err)
			}
			if tt.format != "" {
				q := req.URL.Query()
				q.Set("format", tt.format)
				req.URL.RawQuery = q.Encode()
			}
			response := executeRequest(req)
			if !checkResponseCode(t, tt.status, response) || tt.status != http.StatusOK {
				return
			}

			var list []PublicationTest
			if err := json.Unmarshal(response.Body.Bytes(), &list); err != nil {
				t.Fatal(err)
			}
			expected := publications[tt.contentType]
			if len(list) != len(expected) {
				t.Fatalf("Expected %d publications, got %d", len(expected), len(list))
			}
			byID := make(map[string]*PublicationTest, len(expected))
			for _, pub := range expected {
				byID[pub.UUID] = pub
			}
			for _, pub := range list {
				inPub, ok := byID[pub.UUID]
				if !ok || !comparePublications(inPub, &pub) {
					t.Errorf("Unexpected publication: %+v", pub)
					continue
				}
				delete(byID, pub.UUID)
			}
			if len(byID) != 0 {
				t.Errorf("Missing %d expected publications", len(byID))
			}
		})
	}
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
