package stor

import (
	"math/rand"
	"os"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/google/uuid"
	"syreclabs.com/go/faker"
)

// some global vars shared by all tests
var St Store
var Publications []Publication
var Licenses []LicenseInfo
var pubUUIDs []string

func TestMain(m *testing.M) {

	// generate random publications
	for i := 0; i < 10; i++ {
		pub := Publication{}
		pub.UUID = uuid.New().String()
		pub.Title = faker.Company().CatchPhrase()
		pub.EncryptionKey = make([]byte, 16)
		rand.Read(pub.EncryptionKey)
		pub.Location = faker.Internet().Url()
		if i == 5 || i == 7 {
			pub.ContentType = "application/epub+zip"
		} else {
			pub.ContentType = "application/unknown"
		}
		pub.Size = uint32(faker.Number().NumberInt(5))
		pub.Checksum = faker.Lorem().Characters(16)
		Publications = append(Publications, pub)
		// save the list of pub IDs
		pubUUIDs = append(pubUUIDs, pub.UUID)
	}

	// generate random licenses
	var randomIdx int
	for i := 0; i < 10; i++ {
		lic := LicenseInfo{}
		lic.UUID = uuid.New().String()
		if i == 2 || i == 3 {
			lic.UserID = "Morpheus"
		} else {
			lic.UserID = uuid.New().String()
		}
		// publication IDs must be existing ids
		randomIdx = rand.Intn(len(pubUUIDs))
		lic.PublicationID = pubUUIDs[randomIdx]
		lic.Provider = "http://edrlab.org"
		start := time.Now()
		lic.Start = &start
		end := start.AddDate(0, 0, 10)
		lic.End = &end
		if i == 2 || i == 3 {
			lic.Status = STATUS_REVOKED
		} else {
			lic.Status = STATUS_READY
		}
		lic.DeviceCount = i
		Licenses = append(Licenses, lic)
	}

	// create / open an sqlite db in memory
	dsn := "sqlite3://file::memory:?cache=shared"
	St, _ = Init(dsn)

	// store the publications in the db
	var err error
	for _, p := range Publications {
		err = St.Publication().Create(&p)
		if err != nil {
			log.Fatalf("Failed to create a publication: %v", err)
		}
	}
	// store the licenses in the db
	for _, l := range Licenses {
		err = St.License().Create(&l)
		if err != nil {
			log.Fatalf("Failed to create a license: %v", err)
		}
	}

	code := m.Run()
	os.Exit(code)
}

// TestPublications calls gorm functionalities related to Publications
func TestPublications(t *testing.T) {
	var err error

	// check a publication
	err = Publications[0].Validate()
	if err != nil {
		t.Fatalf("Invalid test publication: %v", err)
	}

	// count publications
	var cnt int64
	cnt, err = St.Publication().Count()
	if err != nil {
		t.Fatalf("Failed to count publications: %v", err)
	}
	if int(cnt) != len(Publications) {
		t.Fatalf("Incorrect publication count: %d", cnt)
	}

	// get publications by their format
	var publications *[]Publication
	contentType := "application/epub+zip"
	publications, err = St.Publication().FindByType(contentType)
	if err != nil {
		t.Fatalf("Failed to get publications by their format: %v", err)
	}
	if len(*publications) != 2 {
		t.Fatalf("Failed to get a 2 EPUB items: %v", err)
	}

	// list all publications
	publications, err = St.Publication().ListAll()
	if err != nil {
		t.Fatalf("Failed to list all publications: %v", err)
	}
	if len(*publications) == 0 {
		t.Fatal("Failed to get a list of publications: empty list")
	}

	// list publications per page (size 3, num 2)
	publications, err = St.Publication().List(3, 2)
	if err != nil {
		t.Fatalf("Failed to list some publications: %v", err)
	}
	if len(*publications) == 0 {
		t.Fatalf("Failed to get a list of publications: %v", err)
	}

	// get a publication by its id
	pubUUID := Publications[1].UUID
	var publication *Publication
	publication, err = St.Publication().Get(pubUUID)
	if err != nil {
		t.Fatalf("Failed to get a publication by uuid: %v", err)
	}

	// update the publication Title
	publication.Title = "La Peste (Camus)"
	err = St.Publication().Update(publication)
	if err != nil {
		t.Fatalf("Failed to update a publication property: %v", err)
	}

	// (soft) delete a publication
	err = St.Publication().Delete(publication)
	if err != nil {
		t.Fatalf("Failed to delete a publication: %v", err)
	}

	// check that the publication has been (soft) deleted
	_, err = St.Publication().Get(publication.UUID)
	if err == nil {
		t.Fatalf("Expected publication to be deleted")
	}

	// check that the creation of a new publication with the same UUID is disallowed
	publication = &Publications[1]
	publication.UUID = uuid.New().String()

	err = St.Publication().Create(publication)
	if err != nil {
		t.Fatalf("Failed to create a new publication: %v", err)
	}
	publication.ID = 0 // raz the gorm id
	err = St.Publication().Create(publication)
	if err == nil {
		t.Fatalf("Failed to disallow the creation of 2 publications with the same UUID: %v", err)
	} else {
		t.Logf("Test positive, it is not possible to create a publication with a already existing UUID: %v", err)
	}
}

// TestLicenses calls gorm functionalities related to License
func TestLicenses(t *testing.T) {
	var err error

	// check a license
	err = Licenses[0].Validate()
	if err != nil {
		t.Fatalf("Invalid test license: %v", err)
	}

	// count licenses
	var cnt int64
	cnt, err = St.License().Count()
	if err != nil {
		t.Fatalf("Failed to count licenses: %v", err)
	}
	if int(cnt) != len(Licenses) {
		t.Fatalf("Incorrect license count: %d", cnt)
	}

	// get licenses by their user
	var licenses *[]LicenseInfo
	licenses, err = St.License().FindByUser("Morpheus")
	if err != nil {
		t.Fatalf("Failed to get licenses by their user: %v", err)
	}
	if len(*licenses) != 2 {
		t.Fatal("Failed to get 2 licenses owned by Morpheus")
	}

	// get licenses by their publication id
	pubUUID := Licenses[5].PublicationID
	licenses, err = St.License().FindByPublication(pubUUID)
	if err != nil {
		t.Fatalf("Failed to get licenses by their publication id: %v", err)
	}
	if len(*licenses) == 0 {
		t.Fatalf("Failed to get at least one license with a specific publication id: %v", err)
	}

	// get licenses by their status
	licenses, err = St.License().FindByStatus(STATUS_REVOKED)
	if err != nil {
		t.Fatalf("Failed to get licenses by their status: %v", err)
	}
	if len(*licenses) != 2 {
		t.Fatal("Failed to get 2 revoked licenses")
	}

	// get licenses by their range of device count
	licenses, err = St.License().FindByDeviceCount(2, 4)
	if err != nil {
		t.Fatalf("Failed to get licenses by their range of device count: %v", err)
	}
	if len(*licenses) != 3 {
		t.Fatal("Failed to get at least one license with a specific range of device count")
	}

	// list all licenses
	licenses, err = St.License().ListAll()
	if err != nil {
		t.Fatalf("Failed to list all licenses: %v", err)
	}
	if len(*licenses) == 0 {
		t.Fatalf("Failed to list all licenses: empty list")
	}

	// list licenses per page (page size 2, num 1)
	licenses, err = St.License().List(2, 1)
	if err != nil {
		t.Fatalf("Failed to list some licenses: %v", err)
	}
	if len(*licenses) == 0 {
		t.Fatalf("Failed to list some licenses: empty list")
	}

	// get a license by its id
	licUUID := Licenses[1].UUID
	var license *LicenseInfo
	license, err = St.License().Get(licUUID)
	if err != nil {
		t.Fatalf("Failed to get a license by uuid: %v", err)
	}

	// update the license
	license.Status = STATUS_REVOKED
	now := time.Now()
	license.Updated = &now
	license.StatusUpdated = &now
	err = St.License().Update(license)
	if err != nil {
		t.Fatalf("Failed to update a license property: %v", err)
	}

	// (soft) delete the license
	err = St.License().Delete(license)
	if err != nil {
		t.Fatalf("Failed to delete a license: %v", err)
	}

	// check that the license has been (soft) deleted
	_, err = St.License().Get(license.UUID)
	if err == nil {
		t.Fatalf("Expected license to be deleted")
	}

	license = &Licenses[0]

	// check the license with an empty UUID
	licUUID = license.UUID
	license.UUID = ""
	err = license.Validate()
	if err == nil {
		t.Fatalf("Invalid UUID validation: %v", err)
	}
	license.UUID = licUUID

	// check the license with an empty user id
	userID := license.UserID
	license.UserID = ""
	err = license.Validate()
	if err == nil {
		t.Fatalf("Invalid UserID validation: %v", err)
	}
	license.UserID = userID

	// check the license with an empty publication id
	pubID := license.PublicationID
	license.PublicationID = ""
	err = license.Validate()
	if err == nil {
		t.Fatalf("Invalid PublicationID validation: %v", err)
	}
	license.PublicationID = pubID

	// check that the creation of a license with a publication id which
	// does not exist in the db is disallowed
	license.UUID = uuid.New().String()
	license.PublicationID = "unknown publication ID"
	err = St.License().Create(license)
	if err == nil {
		t.Fatal("Failed to disallow the creation of a license with a wrong publication id")
	} else {
		t.Logf("Test positive, it is not possible to create a license for a non-existent publication: %v", err)
	}

}
