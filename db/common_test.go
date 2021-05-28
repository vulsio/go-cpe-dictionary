package db

import (
	"reflect"
	"strings"
	"testing"

	"github.com/knqyf263/go-cpe/common"
	"github.com/knqyf263/go-cpe/naming"
	"github.com/kotakanbe/go-cpe-dictionary/models"
)

func prepareTestDB(driver DB) error {
	var testCpeStrings = []string{
		`cpe:2.3:a:ntp:ntp:4.2.5p48:*:*:*:*:*:*:*`,
		`cpe:2.3:a:ntp:ntp:4.2.8:p1-beta1:*:*:*:*:*:*`,
		`cpe:2.3:a:responsive_coming_soon_page_project:responsive_coming_soon_page:1.1.18:*:*:*:*:wordpress:*:*`,

		`cpe:2.3:a:vendorName1:productName1-1:1.1:*:*:*:*:targetSoftware1:targetHardware1:*`,
		`cpe:2.3:a:vendorName1:productName1-2:1.2:*:*:*:*:targetSoftware1:targetHardware1:*`,
		"cpe:2.3:a:vendorName2:productName2:2.0:*:*:*:*:targetSoftware2:targetHardware2:*",
		"cpe:2.3:a:vendorName3:productName3:3.0:*:*:*:*:targetSoftware3:targetHardware3:*",
		"cpe:2.3:a:vendorName4:productName4:4.0:*:*:*:*:targetSoftware4:targetHardware4:*",
		"cpe:2.3:a:vendorName5:productName5:5.0:*:*:*:*:targetSoftware5:targetHardware5:*",
		"cpe:2.3:a:vendorName6:productName6:6.0:*:*:*:*:targetSoftware6:targetHardware6:*",
	}

	testCpes := make([]models.CategorizedCpe, len(testCpeStrings))

	for i, cpeString := range testCpeStrings {
		wfn, err := naming.UnbindFS(cpeString)
		if err != nil {
			return err
		}

		testCpes[i] = models.CategorizedCpe{
			CpeURI:          naming.BindToURI(wfn),
			CpeFS:           naming.BindToFS(wfn),
			Part:            wfn.GetString(common.AttributePart),
			Vendor:          wfn.GetString(common.AttributeVendor),
			Product:         wfn.GetString(common.AttributeProduct),
			Version:         wfn.GetString(common.AttributeVersion),
			Update:          wfn.GetString(common.AttributeUpdate),
			Edition:         wfn.GetString(common.AttributeEdition),
			Language:        wfn.GetString(common.AttributeLanguage),
			SoftwareEdition: wfn.GetString(common.AttributeSwEdition),
			TargetSoftware:  wfn.GetString(common.AttributeTargetSw),
			TargetHardware:  wfn.GetString(common.AttributeTargetHw),
			Other:           wfn.GetString(common.AttributeOther),
		}
	}

	return driver.InsertCpes(testCpes)
}

func testGetVendorProducts(t *testing.T, driver DB) {
	var err error

	if err := prepareTestDB(driver); err != nil {
		t.Errorf("Inserting CPEs: %s", err)
	}

	type Expected struct {
		VendorProduct []string
		ErrString     string
	}

	cases := map[string]struct {
		Expected Expected
	}{
		"OK": {
			Expected: Expected{
				VendorProduct: []string{
					"ntp::ntp",
					"responsive_coming_soon_page_project::responsive_coming_soon_page",
					"vendorName1::productName1\\-1", // TODO: what's with these slashes? Is it a bug?
					"vendorName1::productName1\\-2", // TODO: what's with these slashes? Is it a bug?
					"vendorName2::productName2",
					"vendorName3::productName3",
					"vendorName4::productName4",
					"vendorName5::productName5",
					"vendorName6::productName6",
				},
			},
		},
	}
	for k, tc := range cases {
		var vendorProducts []string
		if vendorProducts, err = driver.GetVendorProducts(); err != nil {
			if !strings.Contains(err.Error(), tc.Expected.ErrString) {
				t.Errorf("%s : actual %s, expected %s", k, err, tc.Expected.ErrString)
				continue
			}
			if len(tc.Expected.ErrString) == 0 {
				t.Errorf("%s : actual %s, expected %s", k, err, tc.Expected.ErrString)
				continue
			}
		} else if 0 < len(tc.Expected.ErrString) {
			t.Errorf("%s : actual %s, expected %s", k, err, tc.Expected.ErrString)
		}
		if !reflect.DeepEqual(vendorProducts, tc.Expected.VendorProduct) {
			t.Errorf("%s: actual %#v, expected %#v", k, vendorProducts, tc.Expected.VendorProduct)
		}
	}
}

func testGetCpesByVendorProduct(t *testing.T, driver DB) {
	var err error

	if err := prepareTestDB(driver); err != nil {
		t.Errorf("Inserting CPEs: %s", err)
	}

	type Expected struct {
		CpeURIs   []string
		ErrString string
	}

	cases := map[string]struct {
		Vendor   string
		Product  string
		Expected Expected
	}{
		"OK": {
			Vendor:  "vendorName1",
			Product: "productName1\\-1",
			Expected: Expected{
				CpeURIs: []string{
					`cpe:/a:vendorName1:productName1-1:1.1::~~~targetSoftware1~targetHardware1~`,
				},
			},
		},
		"OK2": {
			Vendor:  "vendorName1",
			Product: "productName1\\-2",
			Expected: Expected{
				CpeURIs: []string{
					`cpe:/a:vendorName1:productName1-2:1.2::~~~targetSoftware1~targetHardware1~`,
				},
			},
		},
		"OK3": {
			Vendor:  "ntp",
			Product: "ntp",
			Expected: Expected{
				CpeURIs: []string{
					`cpe:/a:ntp:ntp:4.2.5p48`,
					`cpe:/a:ntp:ntp:4.2.8:p1-beta1`,
				},
			},
		},
	}

	for k, tc := range cases {
		var cpeURIs []string
		if cpeURIs, err = driver.GetCpesByVendorProduct(tc.Vendor, tc.Product); err != nil {
			if !strings.Contains(err.Error(), tc.Expected.ErrString) {
				t.Errorf("%s : actual %s, expected %s", k, err, tc.Expected.ErrString)
				continue
			}
			if len(tc.Expected.ErrString) == 0 {
				t.Errorf("%s : actual %s, expected %s", k, err, tc.Expected.ErrString)
				continue
			}
		} else if 0 < len(tc.Expected.ErrString) {
			t.Errorf("%s : actual %s, expected %s", k, err, tc.Expected.ErrString)
		}
		if !reflect.DeepEqual(cpeURIs, tc.Expected.CpeURIs) {
			t.Errorf("%s: actual %#v, expected %#v", k, cpeURIs, tc.Expected.CpeURIs)
		}
	}
}
