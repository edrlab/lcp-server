package check

import (
	"testing"

	"github.com/edrlab/lcp-server/pkg/lic"
)

//func TestMain(m *testing.M) {
//}

func TestCheckLicense(t *testing.T) {

	goodProfiles := [4]string{
		"http://readium.org/lcp/basic-profile",
		"http://readium.org/lcp/1.0",
		"http://readium.org/lcp/2.5",
		"http://readium.org/lcp/2.x",
	}
	var license lic.License
	for _, s := range goodProfiles {
		license.Encryption.Profile = s
		err := checkLicenseProfile(license)
		if err != nil {
			t.Errorf("%v: %s", err, s)
		}
	}
	badProfiles := [2]string{
		"http://readium.org/lcp/3.0",
		"http://readium.org/lcp/2.y",
	}
	for _, s := range badProfiles {
		license.Encryption.Profile = s
		err := checkLicenseProfile(license)
		if err == nil {
			t.Errorf("%v: %s", err, s)
		}
	}

}
