package db

import (
	"fmt"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis/v8"
)

func setupRedis() (*miniredis.Miniredis, DB, error) {
	s, err := miniredis.Run()
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to run miniredis: %s", err)
	}
	driver, err := NewDB("redis", "redis://"+s.Addr(), false)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to new db: %s", err)
	}
	return s, driver, nil
}

func teardownRedis(s *miniredis.Miniredis, driver DB) {
	s.Close()
	driver.CloseDB()
}

func TestGetVendorProductsRedis(t *testing.T) {
	t.Parallel()
	s, driver, err := setupRedis()
	if err != nil {
		t.Errorf("Failed to parepare redis: %s", err)
	}
	defer teardownRedis(s, driver)

	testGetVendorProducts(t, driver)
}

func TestGetCpesByVendorProductRedis(t *testing.T) {
	t.Parallel()
	s, driver, err := setupRedis()
	if err != nil {
		t.Errorf("Failed to parepare redis: %s", err)
	}
	defer teardownRedis(s, driver)

	testGetCpesByVendorProduct(t, driver)
}

func TestRedisDriver_IsDeprecated(t *testing.T) {
	t.Parallel()
	s, driver, err := setupRedis()
	if err != nil {
		t.Errorf("Failed to parepare redis: %s", err)
	}
	defer teardownRedis(s, driver)
	if err := prepareTestData(driver); err != nil {
		t.Errorf("Inserting CPEs: %s", err)
	}

	type fields struct {
		name string
		conn *redis.Client
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
				cpeURI: "cpe:/a:vendorName6:productName6:6.0::~~~targetSoftware6~targetHardware6~",
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
