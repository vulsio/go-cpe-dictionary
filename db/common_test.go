package db

import (
	"reflect"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/hbollon/go-edlib"
	"github.com/spf13/viper"

	"github.com/vulsio/go-cpe-dictionary/models"
)

func prepareTestData(driver DB) error {
	var testCpes = models.FetchedCPEs{
		CPEs: []models.FetchedCPE{
			{
				Title: "NTP NTP 4.2.5p48",
				CPEs:  []string{`cpe:2.3:a:ntp:ntp:4.2.5p48:*:*:*:*:*:*:*`},
			},
			{
				Title: "NTP NTP 4.2.8 p1-beta1",
				CPEs:  []string{`cpe:2.3:a:ntp:ntp:4.2.8:p1-beta1:*:*:*:*:*:*`},
			},
			{
				Title: "responsive_coming_soon_page_project responsive_coming_soon_page 1.1.18 wordpress",
				CPEs:  []string{`cpe:2.3:a:responsive_coming_soon_page_project:responsive_coming_soon_page:1.1.18:*:*:*:*:wordpress:*:*`},
			},
			{
				Title: "vendorName1 productName1-1 1.1 targetSoftware1 targetHardware1",
				CPEs:  []string{`cpe:2.3:a:vendorName1:productName1-1:1.1:*:*:*:*:targetSoftware1:targetHardware1:*`},
			},
			{
				Title: "vendorName1 productName1-2 1.2 targetSoftware1 targetHardware1",
				CPEs:  []string{`cpe:2.3:a:vendorName1:productName1-2:1.2:*:*:*:*:targetSoftware1:targetHardware1:*`},
			},
			{
				Title: "vendorName2 productName2 2.0 targetSoftware2 targetHardware2",
				CPEs:  []string{`cpe:2.3:a:vendorName2:productName2:2.0:*:*:*:*:targetSoftware2:targetHardware2:*`},
			},
			{
				Title: "vendorName3 productName3 3.0 targetSoftware3 targetHardware3",
				CPEs:  []string{`cpe:2.3:a:vendorName3:productName3:3.0:*:*:*:*:targetSoftware3:targetHardware3:*`},
			},
			{
				Title: "vendorName4 productName4 4.0 targetSoftware4 targetHardware4",
				CPEs:  []string{`cpe:2.3:a:vendorName4:productName4:4.0:*:*:*:*:targetSoftware4:targetHardware4:*`},
			},
			{
				Title: "vendorName5 productName5 5.0 targetSoftware5 targetHardware5",
				CPEs:  []string{`cpe:2.3:a:vendorName5:productName5:5.0:*:*:*:*:targetSoftware5:targetHardware5:*`},
			},
			{
				Title: "vendorName6 productName6 6.0 targetSoftware6 targetHardware6",
				CPEs:  []string{`cpe:2.3:a:vendorName6:productName6:6.0:*:*:*:*:targetSoftware6:targetHardware6:*`},
			},
			{
				Title: "MongoDB C# driver 1.10.0",
				CPEs:  []string{`cpe:2.3:a:mongodb:c\#_driver:1.10.0:-:*:*:*:mongodb:*:*`},
			},
		},
		Deprecated: []models.FetchedCPE{
			{
				Title: "vendorName6 productName6 6.1 targetSoftware6 targetHardware6",
				CPEs:  []string{`cpe:2.3:a:vendorName6:productName6:6.1:*:*:*:*:targetSoftware6:targetHardware6:*`},
			},
		},
	}
	viper.Set("threads", 1)
	viper.Set("batch-size", 1)
	return driver.InsertCpes(models.NVD, testCpes)
}

func testGetVendorProducts(t *testing.T, driver DB) {
	if err := prepareTestData(driver); err != nil {
		t.Errorf("Inserting CPEs: %s", err)
	}

	type Expected struct {
		VendorProduct []models.VendorProduct
		Deprecated    []models.VendorProduct
		ErrString     string
	}

	cases := map[string]struct {
		Expected Expected
	}{
		"OK": {
			Expected: Expected{
				VendorProduct: []models.VendorProduct{
					{Vendor: "mongodb", Product: "c\\#_driver"},
					{Vendor: "ntp", Product: "ntp"},
					{Vendor: "responsive_coming_soon_page_project", Product: "responsive_coming_soon_page"},
					{Vendor: "vendorName1", Product: "productName1\\-1"}, // TODO: what's with these slashes? Is it a bug?
					{Vendor: "vendorName1", Product: "productName1\\-2"}, // TODO: what's with these slashes? Is it a bug?
					{Vendor: "vendorName2", Product: "productName2"},
					{Vendor: "vendorName3", Product: "productName3"},
					{Vendor: "vendorName4", Product: "productName4"},
					{Vendor: "vendorName5", Product: "productName5"},
					{Vendor: "vendorName6", Product: "productName6"},
				},
				Deprecated: []models.VendorProduct{
					{Vendor: "vendorName6", Product: "productName6"},
				},
			},
		},
	}
	for k, tc := range cases {
		vendorProducts, deprecated, err := driver.GetVendorProducts()
		if err != nil {
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
		if diff := cmp.Diff(vendorProducts, tc.Expected.VendorProduct); diff != "" {
			t.Errorf("%s: vendor product diff %s", k, diff)
		}
		if diff := cmp.Diff(deprecated, tc.Expected.Deprecated); diff != "" {
			t.Errorf("%s: deprecated vendor product diff %s", k, diff)
		}
	}
}

func testGetCpesByVendorProduct(t *testing.T, driver DB) {
	if err := prepareTestData(driver); err != nil {
		t.Errorf("Inserting CPEs: %s", err)
	}

	type Expected struct {
		CpeURIs    []string
		Deprecated []string
		ErrString  string
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
				Deprecated: []string{},
			},
		},
		"OK2": {
			Vendor:  "vendorName1",
			Product: "productName1\\-2",
			Expected: Expected{
				CpeURIs: []string{
					`cpe:/a:vendorName1:productName1-2:1.2::~~~targetSoftware1~targetHardware1~`,
				},
				Deprecated: []string{},
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
				Deprecated: []string{},
			},
		},
		"deprecated": {
			Vendor:  "vendorName6",
			Product: "productName6",
			Expected: Expected{
				CpeURIs: []string{
					"cpe:/a:vendorName6:productName6:6.0::~~~targetSoftware6~targetHardware6~",
				},
				Deprecated: []string{
					"cpe:/a:vendorName6:productName6:6.1::~~~targetSoftware6~targetHardware6~",
				},
			},
		},
	}

	for k, tc := range cases {
		cpeURIs, deprecated, err := driver.GetCpesByVendorProduct(tc.Vendor, tc.Product)
		if err != nil {
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
		if !reflect.DeepEqual(deprecated, tc.Expected.Deprecated) {
			t.Errorf("actual %#v, expected %#v", deprecated, tc.Expected.Deprecated)
		}
	}
}

func testGetSimilarCpesByTitle(t *testing.T, driver DB) {
	if err := prepareTestData(driver); err != nil {
		t.Errorf("Inserting CPEs: %s", err)
	}

	type expected struct {
		cpes      []models.FetchedCPE
		ErrString string
	}

	cases := map[string]struct {
		query    string
		expected expected
	}{
		"OK": {
			query: "mongodb",
			expected: expected{
				cpes: []models.FetchedCPE{
					{
						Title: "MongoDB C# driver 1.10.0",
						CPEs:  []string{"cpe:/a:mongodb:c%23_driver:1.10.0:-:~~~mongodb~~"},
					},
				},
			},
		},
	}

	for k, tc := range cases {
		cs, err := driver.GetSimilarCpesByTitle(tc.query, 1, edlib.Jaro)
		if err != nil {
			if !strings.Contains(err.Error(), tc.expected.ErrString) {
				t.Errorf("%s : actual %s, expected %s", k, err, tc.expected.ErrString)
				continue
			}
			if len(tc.expected.ErrString) == 0 {
				t.Errorf("%s : actual %s, expected %s", k, err, tc.expected.ErrString)
				continue
			}
		} else if 0 < len(tc.expected.ErrString) {
			t.Errorf("%s : actual %s, expected %s", k, err, tc.expected.ErrString)
		}
		if !reflect.DeepEqual(cs, tc.expected.cpes) {
			t.Errorf("%s: actual %#v, expected %#v", k, cs, tc.expected.cpes)
		}
	}
}
