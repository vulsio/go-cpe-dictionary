package db

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"golang.org/x/xerrors"
)

func setupRedis() (*miniredis.Miniredis, DB, error) {
	s, err := miniredis.Run()
	if err != nil {
		return nil, nil, xerrors.Errorf("Failed to run miniredis: %w", err)
	}
	driver, err := NewDB("redis", "redis://"+s.Addr(), false, Option{})
	if err != nil {
		return nil, nil, xerrors.Errorf("Failed to new db: %w", err)
	}
	return s, driver, nil
}

func teardownRedis(s *miniredis.Miniredis, driver DB) {
	s.Close()
	_ = driver.CloseDB()
}

func TestGetVendorProductsRedis(t *testing.T) {
	s, driver, err := setupRedis()
	if err != nil {
		t.Errorf("Failed to parepare redis: %s", err)
	}
	defer teardownRedis(s, driver)

	testGetVendorProducts(t, driver)
}

func TestGetCpesByVendorProductRedis(t *testing.T) {
	s, driver, err := setupRedis()
	if err != nil {
		t.Errorf("Failed to parepare redis: %s", err)
	}
	defer teardownRedis(s, driver)

	testGetCpesByVendorProduct(t, driver)
}

func TestGetCpesByVendorProductRedisEdgeCases(t *testing.T) {
	cpes := []string{"cpe:/a:vendor:product:1.0", "cpe:/a:vendor:product:2.0"}
	tests := []struct {
		name           string
		vendor         string
		product        string
		members        []string
		deprecated     []string
		wantCpes       []string
		wantDeprecated []string
		wantCommands   int
	}{
		{
			name:           "all deprecated",
			vendor:         "vendor",
			product:        "product",
			members:        cpes,
			deprecated:     cpes,
			wantCpes:       []string{},
			wantDeprecated: cpes,
			wantCommands:   2,
		},
		{
			name:           "mixed",
			vendor:         "vendor",
			product:        "product",
			members:        cpes,
			deprecated:     []string{cpes[0]},
			wantCpes:       []string{cpes[1]},
			wantDeprecated: []string{cpes[0]},
			wantCommands:   2,
		},
		{
			name:           "missing product",
			vendor:         "vendor",
			product:        "product",
			wantCpes:       []string{},
			wantDeprecated: []string{},
			wantCommands:   1,
		},
		{
			name:           "missing deprecated set",
			vendor:         "vendor",
			product:        "product",
			members:        cpes,
			wantCpes:       cpes,
			wantDeprecated: []string{},
			wantCommands:   2,
		},
		{
			name:    "empty vendor",
			product: "product",
		},
		{
			name:   "empty product",
			vendor: "vendor",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, driver, err := setupRedis()
			if err != nil {
				t.Fatalf("Failed to prepare redis: %s", err)
			}
			defer teardownRedis(s, driver)
			if len(tt.members) > 0 {
				if _, err := s.SAdd(fmt.Sprintf(vpKeyFormat, tt.vendor, tt.product), tt.members...); err != nil {
					t.Fatal(err)
				}
			}
			if len(tt.deprecated) > 0 {
				if _, err := s.SAdd(deprecatedCPEsKey, tt.deprecated...); err != nil {
					t.Fatal(err)
				}
			}
			before := s.CommandCount()
			got, deprecated, err := driver.GetCpesByVendorProduct(tt.vendor, tt.product)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, tt.wantCpes) {
				t.Errorf("CPEs = %#v, want %#v", got, tt.wantCpes)
			}
			if !reflect.DeepEqual(deprecated, tt.wantDeprecated) {
				t.Errorf("deprecated = %#v, want %#v", deprecated, tt.wantDeprecated)
			}
			if count := s.CommandCount() - before; count != tt.wantCommands {
				t.Errorf("Redis commands = %d, want %d", count, tt.wantCommands)
			}
		})
	}
}

func TestGetSimilarCpesByTitle(t *testing.T) {
	s, driver, err := setupRedis()
	if err != nil {
		t.Errorf("Failed to parepare redis: %s", err)
	}
	defer teardownRedis(s, driver)

	testGetSimilarCpesByTitle(t, driver)
}

func TestRedisDriver_IsDeprecated(t *testing.T) {
	s, driver, err := setupRedis()
	if err != nil {
		t.Errorf("Failed to parepare redis: %s", err)
	}
	defer teardownRedis(s, driver)
	if err := prepareTestData(driver); err != nil {
		t.Errorf("Inserting CPEs: %s", err)
	}

	type args struct {
		cpeURI string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "test deprecated",
			args: args{
				cpeURI: "cpe:/a:vendorName6:productName6:6.1::~~~targetSoftware6~targetHardware6~",
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "test not deprecated",
			args: args{
				cpeURI: `cpe:/a:vendorName1:productName1-2:1.2::~~~targetSoftware1~targetHardware1~`,
			},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := driver.IsDeprecated(tt.args.cpeURI)
			if (err != nil) != tt.wantErr {
				t.Errorf("RedisDriver.IsDeprecated() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("RedisDriver.IsDeprecated() = %v, want %v", got, tt.want)
			}
		})
	}
}
