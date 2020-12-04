package db

import (
	"testing"

	"github.com/alicebob/miniredis/v2"
)

func TestGetVendorProductsRedis(t *testing.T) {
	t.Parallel()

	s, err := miniredis.Run()
	if err != nil {
		t.Errorf("miniredis: %s", err)
	}
	defer s.Close()

	driver, err := NewDB("redis", "redis://"+s.Addr(), false)
	if err != nil {
		t.Errorf("newdb: %s", err)
	}
	defer func() {
		_ = driver.CloseDB()
	}()
	testGetVendorProducts(t, driver)
}

func TestGetCpesByVendorProductRedis(t *testing.T) {
	t.Parallel()

	s, err := miniredis.Run()
	if err != nil {
		t.Errorf("miniredis: %s", err)
	}
	defer s.Close()

	driver, err := NewDB("redis", "redis://"+s.Addr(), false)
	if err != nil {
		t.Errorf("newdb: %s", err)
	}
	defer func() {
		_ = driver.CloseDB()
	}()

	testGetCpesByVendorProduct(t, driver)
}
