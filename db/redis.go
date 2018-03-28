package db

import (
	"fmt"

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

// InsertCpes Select Cve information from DB.
func (r *RedisDriver) InsertCpes(cpes []*models.CategorizedCpe) (err error) {
	bar := pb.New(len(cpes))
	bar.Start()
	//	var uniqueVendor, uniqueProduct = map[string]bool{}, map[string]bool{}
	for chunked := range chunkSlice(cpes, 10) {
		var pipe redis.Pipeliner
		pipe = r.conn.Pipeline()
		for _, c := range chunked {
			bar.Increment()
			if c.Version == "ANY" {
				continue
			}

			if result := pipe.ZAdd(
				hashKeyPrefix+"CpeURI",
				redis.Z{Score: 0, Member: c.CpeURI},
			); result.Err() != nil {
				return fmt.Errorf("Failed to ZAdd CpeURI and cpe name. err: %s", result.Err())
			}
			if result := pipe.ZAdd(
				hashKeyPrefix+"VendorProduct",
				redis.Z{Score: 0, Member: c.Vendor + hashKeySeparator + c.Product},
			); result.Err() != nil {
				return fmt.Errorf("Failed to ZAdd CpeURI and cpe name. err: %s", result.Err())
			}
			if result := pipe.ZAdd(
				hashKeyPrefix+c.Vendor+hashKeySeparator+c.Product,
				redis.Z{Score: 0, Member: c.CpeURI},
			); result.Err() != nil {
				return fmt.Errorf("Failed to ZAdd CpeURI and cpe name. err: %s", result.Err())
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

// GetVendorProducts : GetVendorProducts
func (r *RedisDriver) GetVendorProducts() (vendorProducts []string, err error) {
	var result *redis.StringSliceCmd
	if result = r.conn.ZRange(hashKeyPrefix+"VendorProduct", 0, -1); result.Err() != nil {
		return nil, result.Err()
	}
	return result.Val(), nil
}

// GetCpesByVendorProduct : GetCpesByVendorProduct
func (r *RedisDriver) GetCpesByVendorProduct(vendor, product string) (cpeURIs []string, err error) {
	if vendor == "" || product == "" {
		return nil, nil
	}
	var result *redis.StringSliceCmd
	if result = r.conn.ZRange(hashKeyPrefix+vendor+hashKeySeparator+product, 0, -1); result.Err() != nil {
		return nil, result.Err()
	}
	return result.Val(), nil
}
