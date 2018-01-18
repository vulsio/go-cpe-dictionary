package db

import (
	"reflect"
	"strings"
	"testing"

	"github.com/sadayuki-matsuno/go-cpe-dictionary/models"
	"github.com/sadayuki-matsuno/go-cpe-dictionary/testutil"
)

func TestGetCpeFromCpe22(t *testing.T) {
	var err error

	type Expected struct {
		CategorizedCpe models.CategorizedCpe
		ErrString      string
	}

	cases := map[string]struct {
		Cpe22Name string
		Expected  Expected
	}{
		"OK": {
			Cpe22Name: "cpe:/a:vendorName1:productName1-1:1.1",
			Expected: Expected{
				CategorizedCpe: models.CategorizedCpe{
					Cpe22URI:        "cpe:/a:vendorName1:productName1-1:1.1",
					Cpe23URI:        "cpe:2.3:a:vendorName1:productName1-1:1.1:*:*:*:*:targetSoftware1:targetHardware1:*",
					Part:            "a",
					Vendor:          "vendorName1",
					Product:         "productName1-1",
					Version:         "1.0",
					Update:          "*",
					Edition:         "*",
					Language:        "*",
					SoftwareEdition: "*",
					TargetSoftware:  "targetSoftware1",
					TargetHardware:  "targetHardware1",
					Other:           "*",
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
		var cpe models.CategorizedCpe
		if cpe, err = driver.GetCpeFromCpe22(tc.Cpe22Name); err != nil {
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
		if !reflect.DeepEqual(cpe, tc.Expected.CategorizedCpe) {
			t.Errorf("%s: actual %#v, expected %#v", k, cpe, tc.Expected.CategorizedCpe)
		}
	}
}

func TestGetCpeFromCpe23(t *testing.T) {
	var err error

	type Expected struct {
		CategorizedCpe models.CategorizedCpe
		ErrString      string
	}

	cases := map[string]struct {
		Cpe23Name string
		Expected  Expected
	}{
		"OK": {
			Cpe23Name: "cpe:2.3:a:vendorName1:productName1-1:1.1:*:*:*:*:targetSoftware1:targetHardware1:*",
			Expected: Expected{
				CategorizedCpe: models.CategorizedCpe{
					Cpe22URI:        "cpe:/a:vendorName1:productName1-1:1.1",
					Cpe23URI:        "cpe:2.3:a:vendorName1:productName1-1:1.1:*:*:*:*:targetSoftware1:targetHardware1:*",
					Part:            "a",
					Vendor:          "vendorName1",
					Product:         "productName1-1",
					Version:         "1.0",
					Update:          "*",
					Edition:         "*",
					Language:        "*",
					SoftwareEdition: "*",
					TargetSoftware:  "targetSoftware1",
					TargetHardware:  "targetHardware1",
					Other:           "*",
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
		var cpe models.CategorizedCpe
		if cpe, err = driver.GetCpeFromCpe23(tc.Cpe23Name); err != nil {
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
		if !reflect.DeepEqual(cpe, tc.Expected.CategorizedCpe) {
			t.Errorf("%s: actual %#v, expected %#v", k, cpe, tc.Expected.CategorizedCpe)
		}
	}
}

func TestGetCategories(t *testing.T) {
	var err error

	type Expected struct {
		FilterableCategories models.FilterableCategories
		ErrString            string
	}

	cases := map[string]struct {
		Expected Expected
	}{
		"OK": {
			Expected: Expected{
				FilterableCategories: models.FilterableCategories{
					Part: []string{"a", "o", "h"},
					VendorProduct: map[string][]string{
						"vendorName1": []string{
							"productName1-1",
							"productName1-2",
						},
						"vendorName2": []string{
							"productName2",
						},
						"vendorName3": []string{
							"productName3",
						},
						"vendorName4": []string{
							"productName4",
						},
						"vendorName5": []string{
							"productName5",
						},
						"vendorName6": []string{
							"productName6",
						},
					},
					TargetSoftware: []string{
						"targetSoftware1",
						"targetSoftware2",
						"targetSoftware3",
						"targetSoftware4",
						"targetSoftware5",
						"targetSoftware6",
					},
					TargetHardware: []string{
						"targetHardware1",
						"targetHardware2",
						"targetHardware3",
						"targetHardware4",
						"targetHardware5",
						"targetHardware6",
					},
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
		var categories models.FilterableCategories
		if categories, err = driver.GetCategories(); err != nil {
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
		if !reflect.DeepEqual(categories, tc.Expected.FilterableCategories) {
			t.Errorf("%s: actual %#v, expected %#v", k, categories, tc.Expected.FilterableCategories)
		}
	}
}

func TestGetFilteredCpe(t *testing.T) {
	var err error

	type Expected struct {
		CategorizedCpes []models.CategorizedCpe
		ErrString       string
	}

	cases := map[string]struct {
		FilterableCategories models.FilterableCategories
		Expected             Expected
	}{
		"OK": {
			FilterableCategories: models.FilterableCategories{
				Part: []string{"a", "o", "h"},
				VendorProduct: map[string][]string{
					"vendorName1": []string{
						"productName1-1",
						"productName1-2",
					},
				},
				TargetSoftware: []string{
					"targetSoftware1",
				},
			},
			Expected: Expected{
				CategorizedCpes: []models.CategorizedCpe{
					{
						Cpe22URI:        "cpe:/a:vendorName1:productName1-1:1.1",
						Cpe23URI:        "cpe:2.3:a:vendorName1:productName1-1:1.1:*:*:*:*:targetSoftware1:targetHardware1:*",
						Part:            "a",
						Vendor:          "vendorName1",
						Product:         "productName1-1",
						Version:         "1.0",
						Update:          "*",
						Edition:         "*",
						Language:        "*",
						SoftwareEdition: "*",
						TargetSoftware:  "targetSoftware1",
						TargetHardware:  "targetHardware1",
						Other:           "*",
					},
					{
						Cpe22URI:        "cpe:/a:vendorName1:productName1-2:1.2",
						Cpe23URI:        "cpe:2.3:a:vendorName1:productName1-2:1.2:*:*:*:*:targetSoftware1:targetHardware1:*",
						Part:            "a",
						Vendor:          "vendorName1",
						Product:         "productName1-2",
						Version:         "1.0",
						Update:          "*",
						Edition:         "*",
						Language:        "*",
						SoftwareEdition: "*",
						TargetSoftware:  "targetSoftware1",
						TargetHardware:  "targetHardware1",
						Other:           "*",
					},
				},
			},
		},
		"NoOtherFilter": {
			FilterableCategories: models.FilterableCategories{
				VendorProduct: map[string][]string{
					"vendorName1": []string{
						"productName1-1",
					},
				},
			},
			Expected: Expected{
				CategorizedCpes: []models.CategorizedCpe{
					{
						Cpe22URI:        "cpe:/a:vendorName1:productName1-1:1.1",
						Cpe23URI:        "cpe:2.3:a:vendorName1:productName1-1:1.1:*:*:*:*:targetSoftware1:targetHardware1:*",
						Part:            "a",
						Vendor:          "vendorName1",
						Product:         "productName1-1",
						Version:         "1.0",
						Update:          "*",
						Edition:         "*",
						Language:        "*",
						SoftwareEdition: "*",
						TargetSoftware:  "targetSoftware1",
						TargetHardware:  "targetHardware1",
						Other:           "*",
					},
				},
			},
		},
		"MultiFilter": {
			FilterableCategories: models.FilterableCategories{
				Part: []string{"a"},
				VendorProduct: map[string][]string{
					"vendorName1": []string{
						"productName1-1",
					},
				},
				TargetSoftware: []string{
					"targetSoftware1",
				},
				TargetHardware: []string{
					"targetHardware1",
				},
			},
			Expected: Expected{
				CategorizedCpes: []models.CategorizedCpe{
					{
						Cpe22URI:        "cpe:/a:vendorName1:productName1-1:1.1",
						Cpe23URI:        "cpe:2.3:a:vendorName1:productName1-1:1.1:*:*:*:*:targetSoftware1:targetHardware1:*",
						Part:            "a",
						Vendor:          "vendorName1",
						Product:         "productName1-1",
						Version:         "1.0",
						Update:          "*",
						Edition:         "*",
						Language:        "*",
						SoftwareEdition: "*",
						TargetSoftware:  "targetSoftware1",
						TargetHardware:  "targetHardware1",
						Other:           "*",
					},
				},
			},
		},
		"WrongFilterPart": {
			FilterableCategories: models.FilterableCategories{
				Part: []string{"o"},
				VendorProduct: map[string][]string{
					"vendorName1": []string{
						"productName1-1",
					},
				},
			},
			Expected: Expected{},
		},
		"WrongFilterTargetSoftware": {
			FilterableCategories: models.FilterableCategories{
				VendorProduct: map[string][]string{
					"vendorName1": []string{
						"productName1-1",
					},
				},
				TargetSoftware: []string{
					"targetSoftware2",
				},
			},
			Expected: Expected{},
		},
		"WrongFilterTargetHardWare": {
			FilterableCategories: models.FilterableCategories{
				VendorProduct: map[string][]string{
					"vendorName1": []string{
						"productName1-1",
					},
				},
				TargetHardware: []string{
					"targetHardware2",
				},
			},
			Expected: Expected{},
		},
		"OneIsMatchAndAnotherIsNot": {
			FilterableCategories: models.FilterableCategories{
				VendorProduct: map[string][]string{
					"vendorName1": []string{
						"productName1-1",
					},
					"vendorName2": []string{
						"productName2",
					},
				},
				TargetHardware: []string{
					"targetHardware2",
				},
			},
			Expected: Expected{
				CategorizedCpes: []models.CategorizedCpe{
					{
						Cpe22URI:        "cpe:/o:vendorName2:productName2:2.0",
						Cpe23URI:        "cpe:2.3:o:vendorName2:productName2:2.0:*:*:*:*:targetSoftware2:targetHardware2:*",
						Part:            "o",
						Vendor:          "vendorName2",
						Product:         "productName2",
						Version:         "2.0",
						Update:          "*",
						Edition:         "*",
						Language:        "*",
						SoftwareEdition: "*",
						TargetSoftware:  "targetSoftware2",
						TargetHardware:  "targetHardware2",
						Other:           "*",
					},
				},
			},
		},
		"NoVendorProduct": {
			FilterableCategories: models.FilterableCategories{
				TargetHardware: []string{
					"targetHardware2",
				},
			},
			Expected: Expected{
				ErrString: "At least one Vendor and Product must be specified",
			},
		},
		"ManyMatch": {
			FilterableCategories: models.FilterableCategories{
				VendorProduct: map[string][]string{
					"vendorName1": []string{
						"productName1-1",
						"productName1-2",
					},
					"vendorName2": []string{
						"productName2",
					},
					"vendorName3": []string{
						"productName3",
					},
				},
				TargetSoftware: []string{
					"targetSoftware1",
					"targetSoftware2",
					"targetSoftware3",
				},
				TargetHardware: []string{
					"targetHardware1",
					"targetHardware2",
					"targetHardware3",
				},
			},
			Expected: Expected{
				CategorizedCpes: []models.CategorizedCpe{
					{
						Cpe22URI:        "cpe:/a:vendorName1:productName1-1:1.1",
						Cpe23URI:        "cpe:2.3:a:vendorName1:productName1-1:1.1:*:*:*:*:targetSoftware1:targetHardware1:*",
						Part:            "a",
						Vendor:          "vendorName1",
						Product:         "productName1-1",
						Version:         "1.0",
						Update:          "*",
						Edition:         "*",
						Language:        "*",
						SoftwareEdition: "*",
						TargetSoftware:  "targetSoftware1",
						TargetHardware:  "targetHardware1",
						Other:           "*",
					},
					{
						Cpe22URI:        "cpe:/a:vendorName1:productName1-2:1.2",
						Cpe23URI:        "cpe:2.3:a:vendorName1:productName1-2:1.2:*:*:*:*:targetSoftware1:targetHardware1:*",
						Part:            "a",
						Vendor:          "vendorName1",
						Product:         "productName1-2",
						Version:         "1.0",
						Update:          "*",
						Edition:         "*",
						Language:        "*",
						SoftwareEdition: "*",
						TargetSoftware:  "targetSoftware1",
						TargetHardware:  "targetHardware1",
						Other:           "*",
					},
					{
						Cpe22URI:        "cpe:/o:vendorName2:productName2:2.0",
						Cpe23URI:        "cpe:2.3:o:vendorName2:productName2:2.0:*:*:*:*:targetSoftware2:targetHardware2:*",
						Part:            "o",
						Vendor:          "vendorName2",
						Product:         "productName2",
						Version:         "2.0",
						Update:          "*",
						Edition:         "*",
						Language:        "*",
						SoftwareEdition: "*",
						TargetSoftware:  "targetSoftware2",
						TargetHardware:  "targetHardware2",
						Other:           "*",
					},
					{
						Cpe22URI:        "cpe:/h:vendorName3:productName3:3.0",
						Cpe23URI:        "cpe:2.3:h:vendorName3:productName3:3.0:*:*:*:*:targetSoftware3:targetHardware3:*",
						Part:            "h",
						Vendor:          "vendorName3",
						Product:         "productName3",
						Version:         "3.0",
						Update:          "*",
						Edition:         "*",
						Language:        "*",
						SoftwareEdition: "*",
						TargetSoftware:  "targetSoftware3",
						TargetHardware:  "targetHardware3",
						Other:           "*",
					},
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
		var cpes []models.CategorizedCpe
		if cpes, err = driver.GetFilteredCpe(tc.FilterableCategories); err != nil {
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
		if !reflect.DeepEqual(cpes, tc.Expected.CategorizedCpes) {
			t.Errorf("%s: actual %#v, expected %#v", k, cpes, tc.Expected.CategorizedCpes)
		}
	}
}
