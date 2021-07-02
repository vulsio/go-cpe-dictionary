package db

import (
	"context"
	"fmt"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/go-redis/redis/v8"
	"github.com/inconshreveable/log15"
	"github.com/kotakanbe/go-cpe-dictionary/models"
)

const (
	dialectRedis     = "redis"
	hKeyPrefix       = "CPE#"
	deprecatedPrefix = hKeyPrefix + "dep#"
	sep              = "::"
)

// RedisDriver is Driver for Redis
type RedisDriver struct {
	name string
	conn *redis.Client
}

// Name return db name
func (r *RedisDriver) Name() string {
	return r.name
}

// OpenDB opens Database
func (r *RedisDriver) OpenDB(dbType, dbPath string, debugSQL bool) (locked bool, err error) {
	if err = r.connectRedis(dbPath); err != nil {
		err = fmt.Errorf("Failed to open DB. dbtype: %s, dbpath: %s, err: %s", dbType, dbPath, err)
	}
	return
}

func (r *RedisDriver) connectRedis(dbPath string) error {
	var err error
	var option *redis.Options
	if option, err = redis.ParseURL(dbPath); err != nil {
		log15.Error("Failed to parse url.", "err", err)
		return err
	}
	ctx := context.Background()
	r.conn = redis.NewClient(option)
	err = r.conn.Ping(ctx).Err()
	return err
}

// CloseDB close Database
func (r *RedisDriver) CloseDB() (err error) {
	if err = r.conn.Close(); err != nil {
		log15.Error("Failed to close DB.", "Type", r.name, "err", err)
		return
	}
	return
}

// MigrateDB migrates Database
func (r *RedisDriver) MigrateDB() error {
	return nil
}

// GetVendorProducts : GetVendorProducts
func (r *RedisDriver) GetVendorProducts() (vendorProducts []string, err error) {
	ctx := context.Background()
	var result *redis.StringSliceCmd
	if result = r.conn.ZRange(ctx, hKeyPrefix+"VendorProduct", 0, -1); result.Err() != nil {
		return nil, result.Err()
	}
	return result.Val(), nil
}

// GetCpesByVendorProduct : GetCpesByVendorProduct
func (r *RedisDriver) GetCpesByVendorProduct(vendor, product string) ([]string, []string, error) {
	if vendor == "" || product == "" {
		return nil, nil, nil
	}
	result := r.conn.ZRange(context.Background(), hKeyPrefix+vendor+sep+product, 0, -1)
	if result.Err() != nil {
		return nil, nil, fmt.Errorf("Failed to zrange CPE. err :%s", result.Err())
	}

	cpeURIs, deprecated := []string{}, []string{}
	for _, cpeURI := range result.Val() {
		ok, err := r.IsDeprecated(cpeURI)
		if err != nil {
			return nil, nil, fmt.Errorf("Failed to get deprecated CPE. err :%s", err)
		}
		if ok {
			deprecated = append(deprecated, cpeURI)
		} else {
			cpeURIs = append(cpeURIs, cpeURI)
		}
	}
	return cpeURIs, deprecated, nil
}

// InsertCpes Select Cve information from DB.
func (r *RedisDriver) InsertCpes(cpes []models.CategorizedCpe) (err error) {
	ctx := context.Background()
	bar := pb.New(len(cpes))
	bar.Start()
	for chunked := range chunkSlice(cpes, 10) {
		var pipe redis.Pipeliner
		pipe = r.conn.Pipeline()
		for _, c := range chunked {
			bar.Increment()
			if result := pipe.ZAdd(ctx, hKeyPrefix+"VendorProduct", &redis.Z{Score: 0, Member: c.Vendor + sep + c.Product}); result.Err() != nil {
				return fmt.Errorf("Failed to ZAdd vendorProduct. err: %s", result.Err())
			}
			if result := pipe.ZAdd(ctx, hKeyPrefix+c.Vendor+sep+c.Product, &redis.Z{Score: 0, Member: c.CpeURI}); result.Err() != nil {
				return fmt.Errorf("Failed to ZAdd CpeURI. err: %s", result.Err())
			}
			if c.Deprecated {
				if result := pipe.Set(ctx, fmt.Sprintf("%s%s", deprecatedPrefix, c.CpeURI), "true", time.Duration(0)); result.Err() != nil {
					return fmt.Errorf("Failed to set to deprecated CPE. err: %s", result.Err())
				}
			}
		}
		if _, err = pipe.Exec(ctx); err != nil {
			return fmt.Errorf("Failed to exec pipeline. err: %s", err)
		}
	}
	bar.Finish()
	log15.Info(fmt.Sprintf("Refreshed %d CPEs.", len(cpes)))
	return nil
}

// IsDeprecated : IsDeprecated
func (r *RedisDriver) IsDeprecated(cpeURI string) (bool, error) {
	cmd := r.conn.Get(context.Background(), fmt.Sprintf("%s%s", deprecatedPrefix, cpeURI))
	if cmd.Err() == redis.Nil {
		// key not found means the CPE is not deprecated
		return false, nil
	} else if cmd.Err() != nil {
		return false, fmt.Errorf("Failed to get deprecated CPE. err :%s", cmd.Err())
	}
	return cmd.Val() == "true", nil
}
