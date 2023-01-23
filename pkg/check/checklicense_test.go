package check

import "testing"

func TestMain(m *testing.M) {
}

func TestCheckLicense(t *testing.T) {

	var license License
	license.Encryption.Profile = "http://readium.org/lcp/basic-profile"
	err := CheckLicense(license, "")
	if err != nil {
		t.Error("Checking license profile failed (1)")
	}
	license.Encryption.Profile = "http://readium.org/lcp/1.0"
	err = CheckLicense(license, "")
	if err != nil {
		t.Error("Checking license profile failed (1)")
	}
	license.Encryption.Profile = "http://readium.org/lcp/2.5"
	err = CheckLicense(license, "")
	if err != nil {
		t.Error("Checking license profile failed (1)")
	}
	license.Encryption.Profile = "http://readium.org/lcp/2.x"
	err = CheckLicense(license, "")
	if err != nil {
		t.Error("Checking license profile failed (1)")
	}
	license.Encryption.Profile = "http://readium.org/lcp/1.1"
	err = CheckLicense(license, "")
	if err == nil {
		t.Error("Checking license profile failed (2)")
	}
	license.Encryption.Profile = "http://readium.org/lcp/3.0"
	err = CheckLicense(license, "")
	if err == nil {
		t.Error("Checking license profile failed (3)")
	}
	license.Encryption.Profile = "http://readium.org/lcp/3.0"
	err = CheckLicense(license, "")
	if err == nil {
		t.Error("Checking license profile failed (3)")
	}

}
