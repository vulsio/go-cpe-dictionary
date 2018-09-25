package db

import (
	"reflect"
	"strings"
	"testing"

	"github.com/kotakanbe/go-cpe-dictionary/testutil"
)

func TestGetVendorProducts(t *testing.T) {
	var err error

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
					"vendorName1::productName1-1",
					"vendorName1::productName1-2",
					"vendorName2::productName2",
					"vendorName3::productName3",
					"vendorName4::productName4",
					"vendorName5::productName5",
					"vendorName6::productName6",
				},
			},
		},
	}

	s, err := testutil.PrepareTestRedis()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	var driver DB
	if driver, err = NewDB("redis", "redis://"+s.Addr(), false); err != nil {
		t.Error(err)
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

func TestGetCpesByVendorProduct(t *testing.T) {
	var err error

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
			Product: "productName1-1",
			Expected: Expected{
				CpeURIs: []string{
					"cpe:/a:vendorName1:productName1-1:1.1:*:*:*:*:targetSoftware1:targetHardware1:*",
				},
			},
		},
		"OK2": {
			Vendor:  "vendorName1",
			Product: "productName1-2",
			Expected: Expected{
				CpeURIs: []string{
					"cpe:/a:vendorName1:productName1-2:1.2:*:*:*:*:targetSoftware1:targetHardware1:*",
				},
			},
		},
	}

	s, err := testutil.PrepareTestRedis()
	if err != nil {
		panic(err)
	}
	defer s.Close()

	var driver DB
	if driver, err = NewDB("redis", "redis://"+s.Addr(), false); err != nil {
		t.Error(err)
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
