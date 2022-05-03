package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/go-redis/redis/v8"
	"github.com/inconshreveable/log15"
	"github.com/spf13/viper"
	"golang.org/x/xerrors"

	"github.com/vulsio/go-cpe-dictionary/config"
	"github.com/vulsio/go-cpe-dictionary/models"
)

/**
# Redis Data Structure
- Sets
  ┌─────────────────────────────┬───────────────────────┬────────────────────────────────────┐
  │       KEY                   │  MEMBER               │              PURPOSE               │
  └─────────────────────────────┴───────────────────────┴────────────────────────────────────┘
  ┌─────────────────────────────┬───────────────────────┬────────────────────────────────────┐
  │ CPE#VendorProducts          │ ${vendor}##${product} │ Get ALL Vendor Products            │
  ├─────────────────────────────┼───────────────────────┼────────────────────────────────────┤
  │ CPE#DeprecatedVendorProducts│ ${vendor}##${product} │ Get ALL Deprecated Vendor Products │
  ├─────────────────────────────┼───────────────────────┼────────────────────────────────────┤
  │ CPE#VP#${vendor}##${product}│ CPEURI                │ Get CPEURI by vendor and product   │
  ├─────────────────────────────┼───────────────────────┼────────────────────────────────────┤
  │ CPE#DeprecatedCPEs          │ CPEURI                │ Get DeprecatedCPEs                 │
  └─────────────────────────────┴───────────────────────┴────────────────────────────────────┘

- Hash
  ┌───┬────────────────┬───────────────┬─────────────┬──────────────────────────────────────────────────┐
  │NO │    KEY         │   FIELD       │  VALUE      │                     PURPOSE                      │
  └───┴────────────────┴───────────────┴─────────────┴──────────────────────────────────────────────────┘
  ┌───┬────────────────┬───────────────┬─────────────┬──────────────────────────────────────────────────┐
  │ 1 │ CPE#DEP        │    NVD/JVN    │  JSON       │ TO DELETE OUTDATED AND UNNEEDED FIELD AND MEMBER │
  ├───┼────────────────┼───────────────┼─────────────┼──────────────────────────────────────────────────┤
  │ 2 │ CPE#FETCHMETA  │   Revision    │ string      │ GET Go-Cpe-Dictionary Binary Revision            │
  ├───┼────────────────┼───────────────┼─────────────┼──────────────────────────────────────────────────┤
  │ 3 │ CPE#FETCHMETA  │ SchemaVersion │  uint       │ GET Go-Cpe-Dictionary Schema Version             │
  ├───┼────────────────┼───────────────┼─────────────┼──────────────────────────────────────────────────┤
  │ 4 │ CPE#FETCHMETA  │ LastFetchedAt │  time.Time  │ GET Go-Cpe-Dictionary Last Fetched Time          │
  └───┴────────────────┴───────────────┴─────────────┴──────────────────────────────────────────────────┘
**/

const (
	dialectRedis        = "redis"
	vpKeyFormat         = "CPE#VP#%s##%s"
	vpListKey           = "CPE#VendorProducts"
	deprecatedVPListKey = "CPE#DeprecatedVendorProducts"
	vpSeparator         = "##"
	deprecatedCPEsKey   = "CPE#DeprecatedCPEs"
	depKey              = "CPE#DEP"
	fetchMetaKey        = "CPE#FETCHMETA"
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
func (r *RedisDriver) OpenDB(_, dbPath string, _ bool, option Option) (bool, error) {
	return false, r.connectRedis(dbPath, option)
}

func (r *RedisDriver) connectRedis(dbPath string, option Option) error {
	opt, err := redis.ParseURL(dbPath)
	if err != nil {
		return xerrors.Errorf("Failed to parse url. err: %w", err)
	}
	if 0 < option.RedisTimeout.Seconds() {
		opt.ReadTimeout = option.RedisTimeout
	}
	r.conn = redis.NewClient(opt)
	return r.conn.Ping(context.Background()).Err()
}

// CloseDB close Database
func (r *RedisDriver) CloseDB() error {
	if err := r.conn.Close(); err != nil {
		return xerrors.Errorf("Failed to close DB. Type: %s, err: %w", r.name, err)
	}
	return nil
}

// MigrateDB migrates Database
func (r *RedisDriver) MigrateDB() error {
	return nil
}

// IsGoCPEDictModelV1 determines if the DB was created at the time of go-cpe-dictionary Model v1
func (r *RedisDriver) IsGoCPEDictModelV1() (bool, error) {
	ctx := context.Background()

	exists, err := r.conn.Exists(ctx, fetchMetaKey).Result()
	if err != nil {
		return false, xerrors.Errorf("Failed to Exists. err: %w", err)
	}
	if exists == 0 {
		keys, _, err := r.conn.Scan(ctx, 0, "CPE#*", 1).Result()
		if err != nil {
			return false, xerrors.Errorf("Failed to Scan. err: %w", err)
		}
		if len(keys) == 0 {
			return false, nil
		}
		return true, nil
	}

	return false, nil
}

// GetFetchMeta get FetchMeta from Database
func (r *RedisDriver) GetFetchMeta() (*models.FetchMeta, error) {
	ctx := context.Background()

	exists, err := r.conn.Exists(ctx, fetchMetaKey).Result()
	if err != nil {
		return nil, xerrors.Errorf("Failed to Exists. err: %w", err)
	}
	if exists == 0 {
		return &models.FetchMeta{GoCPEDictRevision: config.Revision, SchemaVersion: models.LatestSchemaVersion, LastFetchedAt: time.Date(1000, time.January, 1, 0, 0, 0, 0, time.UTC)}, nil
	}

	revision, err := r.conn.HGet(ctx, fetchMetaKey, "Revision").Result()
	if err != nil {
		return nil, xerrors.Errorf("Failed to HGet Revision. err: %w", err)
	}

	verstr, err := r.conn.HGet(ctx, fetchMetaKey, "SchemaVersion").Result()
	if err != nil {
		return nil, xerrors.Errorf("Failed to HGet SchemaVersion. err: %w", err)
	}
	version, err := strconv.ParseUint(verstr, 10, 8)
	if err != nil {
		return nil, xerrors.Errorf("Failed to ParseUint. err: %w", err)
	}

	datestr, err := r.conn.HGet(ctx, fetchMetaKey, "LastFetchedAt").Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			return nil, xerrors.Errorf("Failed to HGet LastFetchedAt. err: %w", err)
		}
		datestr = time.Date(1000, time.January, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
	}
	date, err := time.Parse(time.RFC3339, datestr)
	if err != nil {
		return nil, xerrors.Errorf("Failed to Parse date. err: %w", err)
	}

	return &models.FetchMeta{GoCPEDictRevision: revision, SchemaVersion: uint(version), LastFetchedAt: date}, nil
}

// UpsertFetchMeta upsert FetchMeta to Database
func (r *RedisDriver) UpsertFetchMeta(fetchMeta *models.FetchMeta) error {
	return r.conn.HSet(context.Background(), fetchMetaKey, map[string]interface{}{"Revision": config.Revision, "SchemaVersion": models.LatestSchemaVersion, "LastFetchedAt": fetchMeta.LastFetchedAt}).Err()
}

// GetVendorProducts : GetVendorProducts
func (r *RedisDriver) GetVendorProducts() ([]models.VendorProduct, []models.VendorProduct, error) {
	vendorProducts, err := r.getVendorProducts(vpListKey)
	if err != nil {
		return nil, nil, xerrors.Errorf("Failed to get vendor products. err: %w", err)
	}
	deprecated, err := r.getVendorProducts(deprecatedVPListKey)
	if err != nil {
		return nil, nil, xerrors.Errorf("Failed to get deprecated vendor products. err: %w", err)
	}
	return vendorProducts, deprecated, nil
}

func (r *RedisDriver) getVendorProducts(redisKey string) ([]models.VendorProduct, error) {
	ctx := context.Background()
	result, err := r.conn.SMembers(ctx, redisKey).Result()
	if err != nil {
		return nil, xerrors.Errorf("Failed to SMembers. key: %s, err: %w", redisKey, err)
	}
	vendorProducts := []models.VendorProduct{}
	for _, vp := range result {
		vpParts := strings.Split(vp, vpSeparator)
		if len(vpParts) != 2 {
			continue
		}
		vendorProducts = append(vendorProducts, models.VendorProduct{
			Vendor:  vpParts[0],
			Product: vpParts[1],
		})
	}
	return vendorProducts, nil
}

// GetCpesByVendorProduct : GetCpesByVendorProduct
func (r *RedisDriver) GetCpesByVendorProduct(vendor, product string) ([]string, []string, error) {
	if vendor == "" || product == "" {
		return nil, nil, nil
	}
	result, err := r.conn.SMembers(context.Background(), fmt.Sprintf(vpKeyFormat, vendor, product)).Result()
	if err != nil {
		return nil, nil, xerrors.Errorf("Failed to SMembers CPE. err: %w", err)
	}

	cpeURIs, deprecated := []string{}, []string{}
	for _, cpeURI := range result {
		ok, err := r.IsDeprecated(cpeURI)
		if err != nil {
			return nil, nil, xerrors.Errorf("Failed to get deprecated CPE. err :%s", err)
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
func (r *RedisDriver) InsertCpes(fetchType models.FetchType, cpes []models.CategorizedCpe) (err error) {
	ctx := context.Background()
	batchSize := viper.GetInt("batch-size")
	if batchSize < 1 {
		return xerrors.Errorf("Failed to set batch-size. err: batch-size option is not set properly")
	}

	// newDeps, oldDeps: {"VP": {"${part}#${vendor}": {"CPEURI": {}}}, "VendorProducts": {"${part}#${vendor}": {}}, "DeprecatedVendorProducts": {"${part}#${vendor}": {}}, "DeprecatedCPEs": {"CPEURI": {}}}
	newDeps := map[string]map[string]map[string]struct{}{
		"VP":                       {},
		"VendorProducts":           {},
		"DeprecatedVendorProducts": {},
		"DeprecatedCPEs":           {},
	}
	oldDepsStr, err := r.conn.HGet(ctx, depKey, string(fetchType)).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			return xerrors.Errorf("Failed to Get key: %s. err: %w", depKey, err)
		}
		oldDepsStr = `{
			"VP": {},
			"VendorProducts": {},
			"DeprecatedVendorProducts": {},
			"DeprecatedCPEs": {}
		}`
	}
	var oldDeps map[string]map[string]map[string]struct{}
	if err := json.Unmarshal([]byte(oldDepsStr), &oldDeps); err != nil {
		return xerrors.Errorf("Failed to unmarshal JSON. err: %w", err)
	}

	bar := pb.StartNew(len(cpes))
	for idx := range chunkSlice(len(cpes), batchSize) {
		pipe := r.conn.Pipeline()
		for _, c := range cpes[idx.From:idx.To] {
			vendorProductStr := fmt.Sprintf("%s%s%s", c.Vendor, vpSeparator, c.Product)
			if c.Deprecated {
				_ = pipe.SAdd(ctx, deprecatedCPEsKey, c.CpeURI)
				newDeps["DeprecatedCPEs"][c.CpeURI] = map[string]struct{}{}
				delete(oldDeps["DeprecatedCPEs"], c.CpeURI)

				_ = pipe.SAdd(ctx, deprecatedVPListKey, vendorProductStr)
				newDeps["DeprecatedVendorProducts"][vendorProductStr] = map[string]struct{}{}
				delete(oldDeps["DeprecatedVendorProducts"], vendorProductStr)
			} else {
				_ = pipe.SAdd(ctx, vpListKey, vendorProductStr)
				newDeps["VendorProducts"][vendorProductStr] = map[string]struct{}{}
				delete(oldDeps["VendorProducts"], vendorProductStr)
			}
			_ = pipe.SAdd(ctx, fmt.Sprintf(vpKeyFormat, c.Vendor, c.Product), c.CpeURI)
			if _, ok := newDeps["VP"][vendorProductStr]; !ok {
				newDeps["VP"][vendorProductStr] = map[string]struct{}{}
			}
			newDeps["VP"][vendorProductStr][c.CpeURI] = struct{}{}
			if _, ok := oldDeps["VP"][vendorProductStr]; ok {
				delete(oldDeps["VP"][vendorProductStr], c.CpeURI)
				if len(oldDeps["VP"][vendorProductStr]) == 0 {
					delete(oldDeps["VP"], vendorProductStr)
				}
			}

		}
		if _, err = pipe.Exec(ctx); err != nil {
			return xerrors.Errorf("Failed to exec pipeline. err: %w", err)
		}
		bar.Add(idx.To - idx.From)
	}
	bar.Finish()
	log15.Info(fmt.Sprintf("Refreshed %d CPEs.", len(cpes)))

	pipe := r.conn.Pipeline()
	for vendorProductStr, cpeURIs := range oldDeps["VP"] {
		for cpeURI := range cpeURIs {
			ss := strings.Split(vendorProductStr, vpSeparator)
			_ = pipe.SRem(ctx, fmt.Sprintf(vpKeyFormat, ss[0], ss[1]), cpeURI)
		}
	}
	for vendorProductStr := range oldDeps["VendorProducts"] {
		_ = pipe.SRem(ctx, vpListKey, vendorProductStr)
	}
	for vendorProductStr := range oldDeps["DeprecatedVendorProducts"] {
		_ = pipe.SRem(ctx, deprecatedVPListKey, vendorProductStr)
	}
	for cpeURI := range oldDeps["DeprecatedCPEs"] {
		_ = pipe.SRem(ctx, deprecatedCPEsKey, cpeURI)
	}

	newDepsJSON, err := json.Marshal(newDeps)
	if err != nil {
		return xerrors.Errorf("Failed to Marshal JSON. err: %w", err)
	}
	_ = pipe.HSet(ctx, depKey, string(fetchType), string(newDepsJSON))
	if _, err = pipe.Exec(ctx); err != nil {
		return xerrors.Errorf("Failed to exec pipeline. err: %w", err)
	}

	return nil
}

// IsDeprecated : IsDeprecated
func (r *RedisDriver) IsDeprecated(cpeURI string) (bool, error) {
	result, err := r.conn.SIsMember(context.Background(), deprecatedCPEsKey, cpeURI).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil
		}
		return false, xerrors.Errorf("Failed to SIsMember. err :%s", err)
	}
	return result, nil
}
