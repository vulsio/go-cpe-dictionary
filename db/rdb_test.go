package db

import (
	"reflect"
	"testing"
)

// Notes:
//
// gorm seems to be doing something odd with these :memory:
// databases. It seems a bit like a shared connection. So we can't
// support parallel tests. We get weird go concurrency issues.

func TestGetVendorProductsSqlite(t *testing.T) {
	driver, err := NewDB("sqlite3", ":memory:", false)
	if err != nil {
		t.Error(err)
	}
	defer func() {
		_ = driver.CloseDB()
	}()
	testGetVendorProducts(t, driver)
}

func TestGetCpesByVendorProductSqlite(t *testing.T) {
	driver, err := NewDB("sqlite3", ":memory:", false)
	if err != nil {
		t.Error(err)
	}
	defer func() {
		_ = driver.CloseDB()
	}()

	testGetCpesByVendorProduct(t, driver)
}

// TestGetCpesByVendorProductSqliteFuzzy includes a % for some simple fuzzy matches not supported by all drivers.
func TestGetCpesByVendorProductSqliteFuzzy(t *testing.T) {

	driver, err := NewDB("sqlite3", ":memory:", false)
	if err != nil {
		t.Error(err)
	}
	defer func() {
		_ = driver.CloseDB()
	}()

	if err := prepareTestDB(driver); err != nil {
		t.Errorf("Inserting CPEs: %s", err)
	}

	expected := []string{
		"cpe:/a:vendorName1:productName1-1:1.1::~~~targetSoftware1~targetHardware1~",
		"cpe:/a:vendorName1:productName1-2:1.2::~~~targetSoftware1~targetHardware1~",
		"cpe:/a:vendorName2:productName2:2.0::~~~targetSoftware2~targetHardware2~",
		"cpe:/a:vendorName3:productName3:3.0::~~~targetSoftware3~targetHardware3~",
		"cpe:/a:vendorName4:productName4:4.0::~~~targetSoftware4~targetHardware4~",
		"cpe:/a:vendorName5:productName5:5.0::~~~targetSoftware5~targetHardware5~",
		"cpe:/a:vendorName6:productName6:6.0::~~~targetSoftware6~targetHardware6~",
	}

	var cpeURIs []string
	if cpeURIs, err = driver.GetCpesByVendorProduct("vendor%", "product%"); err != nil {
		t.Errorf("GetCpesByVendorProduct: %s", err)
	}

	if len(cpeURIs) != len(expected) {
		t.Errorf("actual count %d, expected count %d", len(cpeURIs), len(expected))

	}
	if !reflect.DeepEqual(cpeURIs, expected) {
		t.Errorf("actual %#v, expected %#v", cpeURIs, expected)
	}

}
