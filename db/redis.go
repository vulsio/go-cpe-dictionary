package db

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/cheggaaa/pb"
	"github.com/go-redis/redis"
	"github.com/kotakanbe/go-cpe-dictionary/models"
	log "github.com/sirupsen/logrus"
)

const (
	dialectRedis     = "redis"
	hashKeyPrefix    = "CPE#"
	hashKeySeparator = "::"
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

// NewRedis return Redis driver
func NewRedis(dbType, dbpath string, debugSQL bool) (driver *RedisDriver, err error) {
	driver = &RedisDriver{
		name: dbType,
	}
	log.Debugf("Opening DB (%s).", driver.Name())
	if err = driver.OpenDB(dbType, dbpath, debugSQL); err != nil {
		return
	}

	return
}

// OpenDB opens Database
func (r *RedisDriver) OpenDB(dbType, dbPath string, debugSQL bool) (err error) {
	var option *redis.Options
	if option, err = redis.ParseURL(dbPath); err != nil {
		log.Error(err)
		return fmt.Errorf("Failed to Parse Redis URL. dbpath: %s, err: %s", dbPath, err)
	}
	r.conn = redis.NewClient(option)
	if err = r.conn.Ping().Err(); err != nil {
		return fmt.Errorf("Failed to open DB. dbtype: %s, dbpath: %s, err: %s", dbType, dbPath, err)
	}
	return nil
}

// CloseDB close Database
func (r *RedisDriver) CloseDB() (err error) {
	if err = r.conn.Close(); err != nil {
		log.Errorf("Failed to close DB. Type: %s. err: %s", r.name, err)
		return
	}
	return
}

// GetCpe Select Cve information from DB.
func (r *RedisDriver) GetCpe(cpeName string) models.Cpe {
	return models.Cpe{}
}

// InsertCpes Select Cve information from DB.
func (r *RedisDriver) InsertCpes(cpes []models.Cpe) (err error) {
	bar := pb.New(len(cpes))
	bar.Start()
	var uniqueVendor, uniqueProduct = map[string]bool{}, map[string]bool{}
	for chunked := range chunkSlice(cpes, 10) {
		var pipe redis.Pipeliner
		pipe = r.conn.Pipeline()
		for _, c := range chunked {
			bar.Increment()
			var cpeJSON []byte
			if cpeJSON, err = json.Marshal(c); err != nil {
				return fmt.Errorf("Failed to marshal json. err: %s", err)
			}
			var cpeParts = strings.Split(c.Name, ":")
			var part, vendor, product = cpeParts[1], cpeParts[2], cpeParts[3]
			if result := pipe.HSet(hashKeyPrefix+"CPE", c.Name, string(cpeJSON)); result.Err() != nil {
				return fmt.Errorf("Failed to HSet CPE. err: %s", result.Err())
			}

			if !uniqueVendor[vendor] {
				if result := pipe.ZAdd(
					hashKeyPrefix+"VENDOR",
					redis.Z{Score: 0, Member: vendor},
				); result.Err() != nil {
					return fmt.Errorf("Failed to ZAdd cpe name. err: %s", result.Err())
				}
				uniqueVendor[vendor] = true
			}

			if result := pipe.ZAdd(
				hashKeyPrefix+"VENDOR"+hashKeySeparator+vendor,
				redis.Z{Score: 0, Member: product},
			); result.Err() != nil {
				return fmt.Errorf("Failed to ZAdd cpe name. err: %s", result.Err())
			}

			if !uniqueProduct[product] {
				if result := pipe.ZAdd(
					hashKeyPrefix+"PRODUCT",
					redis.Z{Score: 0, Member: product},
				); result.Err() != nil {
					return fmt.Errorf("Failed to ZAdd cpe name. err: %s", result.Err())
				}
				uniqueProduct[product] = true
			}

			if result := pipe.ZAdd(
				hashKeyPrefix+"PRODUCT"+hashKeySeparator+product,
				redis.Z{Score: 0, Member: c.Name},
			); result.Err() != nil {
				return fmt.Errorf("Failed to ZAdd cpe name. err: %s", result.Err())
			}

			if result := pipe.ZAdd(
				hashKeyPrefix+"CPENAME",
				redis.Z{Score: 0, Member: c.Name},
			); result.Err() != nil {
				return fmt.Errorf("Failed to ZAdd cpe name. err: %s", result.Err())
			}

			if result := pipe.ZAdd(
				hashKeyPrefix+"PART"+hashKeySeparator+part[1:]+hashKeySeparator+"VENDOR",
				redis.Z{Score: 0, Member: vendor},
			); result.Err() != nil {
				return fmt.Errorf("Failed to ZAdd cpe name. err: %s", result.Err())
			}

			if result := pipe.ZAdd(
				hashKeyPrefix+"PART"+hashKeySeparator+part[1:]+hashKeySeparator+"PRODUCT",
				redis.Z{Score: 0, Member: product},
			); result.Err() != nil {
				return fmt.Errorf("Failed to ZAdd cpe name. err: %s", result.Err())
			}
		}
		if _, err = pipe.Exec(); err != nil {
			return fmt.Errorf("Failed to exec pipeline. err: %s", err)
		}
	}
	bar.Finish()
	log.Infof("Refreshed %d CPEs.", len(cpes))
	return nil
}
