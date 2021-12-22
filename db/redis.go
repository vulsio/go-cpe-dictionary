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
	"github.com/vulsio/go-cpe-dictionary/config"
	"github.com/vulsio/go-cpe-dictionary/models"
	"golang.org/x/xerrors"
)

/**
# Redis Data Structure
- Sets
  ┌─────────────────────────────┬──────────────────────┬──────────────────────────────────┐
  │       KEY                   │  MEMBER              │             PURPOSE              │
  └─────────────────────────────┴──────────────────────┴──────────────────────────────────┘
  ┌─────────────────────────────┬──────────────────────┬──────────────────────────────────┐
  │ CPE#VendorProducts          │ ${vendor}#${product} │ Get ALL Vendor Products          │
  ├─────────────────────────────┼──────────────────────┼──────────────────────────────────┤
  │ CPE#VP#${vendor}#${product} │ CPEURI               │ Get CPEURI by vendor and product │
  ├─────────────────────────────┼──────────────────────┼──────────────────────────────────┤
  │ CPE#DeprecatedCPEs          │ CPEURI               │ Get DeprecatedCPEs               │
  └─────────────────────────────┴──────────────────────┴──────────────────────────────────┘

- Hash
  ┌───┬────────────────┬─────────────────┬─────────────┬──────────────────────────────────────────────────┐
  │NO │    KEY         │   FIELD         │  VALUE      │                     PURPOSE                      │
  └───┴────────────────┴─────────────────┴─────────────┴──────────────────────────────────────────────────┘
  ┌───┬────────────────┬─────────────────┬─────────────┬──────────────────────────────────────────────────┐
  │ 1 │ CPE#DEP        │    NVD/JVN      │  JSON       │ TO DELETE OUTDATED AND UNNEEDED FIELD AND MEMBER │
  ├───┼────────────────┼─────────────────┼─────────────┼──────────────────────────────────────────────────┤
  │ 2 │ CPE#FETCHMETA  │   Revision      │ string      │ GET Go-Cpe-Dictionary Binary Revision            │
  ├───┼────────────────┼─────────────────┼─────────────┼──────────────────────────────────────────────────┤
  │ 3 │ CPE#FETCHMETA  │ SchemaVersion   │  uint       │ GET Go-Cpe-Dictionary Schema Version             │
  ├───┼────────────────┼─────────────────┼─────────────┼──────────────────────────────────────────────────┤
  │ 4 │ CPE#FETCHMETA  │ LastFetchedDate │  time.Time  │ GET Go-Cpe-Dictionary Last Fetched Time          │
  └───┴────────────────┴─────────────────┴─────────────┴──────────────────────────────────────────────────┘
**/

const (
	dialectRedis      = "redis"
	vpKeyFormat       = "CPE#VP#%s#%s"
	vpListKey         = "CPE#VendorProducts"
	vpSeparator       = "##"
	deprecatedCPEsKey = "CPE#DeprecatedCPEs"
	depKey            = "CPE#DEP"
	fetchMetaKey      = "CPE#FETCHMETA"
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
func (r *RedisDriver) OpenDB(dbType, dbPath string, debugSQL bool, option Option) (bool, error) {
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
		return &models.FetchMeta{GoCPEDictRevision: config.Revision, SchemaVersion: models.LatestSchemaVersion, LastFetchedDate: time.Date(1000, time.January, 1, 0, 0, 0, 0, time.UTC)}, nil
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

	datestr, err := r.conn.HGet(ctx, fetchMetaKey, "LastFetchedDate").Result()
	if err != nil {
		return nil, xerrors.Errorf("Failed to HGet LastFetchedDate. err: %w", err)
	}
	date, err := time.Parse(time.RFC3339, datestr)
	if err != nil {
		return nil, xerrors.Errorf("Failed to Parse date. err: %w", err)
	}

	return &models.FetchMeta{GoCPEDictRevision: revision, SchemaVersion: uint(version), LastFetchedDate: date}, nil
}

// UpsertFetchMeta upsert FetchMeta to Database
func (r *RedisDriver) UpsertFetchMeta(fetchMeta *models.FetchMeta) error {
	return r.conn.HSet(context.Background(), fetchMetaKey, map[string]interface{}{"Revision": config.Revision, "SchemaVersion": models.LatestSchemaVersion, "LastFetchedDate": fetchMeta.LastFetchedDate}).Err()
}

// GetVendorProducts : GetVendorProducts
func (r *RedisDriver) GetVendorProducts() (vendorProducts []models.VendorProduct, err error) {
	ctx := context.Background()
	result, err := r.conn.SMembers(ctx, vpListKey).Result()
	if err != nil {
		return nil, err
	}
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

	// newDeps, oldDeps: {"VP": {"${part}#${vendor}": {"CPEURI": {}}}, "VendorProducts": {"${part}#${vendor}": {}}, "DeprecatedCPEs": {"CPEURI": {}}}
	newDeps := map[string]map[string]map[string]struct{}{
		"VP":             {},
		"VendorProducts": {},
		"DeprecatedCPEs": {},
	}
	oldDepsStr, err := r.conn.HGet(ctx, depKey, string(fetchType)).Result()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			return xerrors.Errorf("Failed to Get key: %s. err: %w", depKey, err)
		}
		oldDepsStr = `{
			"VP": {},
			"VendorProducts": {},
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
			bar.Increment()
			vendorProductStr := fmt.Sprintf("%s%s%s", c.Vendor, vpSeparator, c.Product)
			if err := pipe.SAdd(ctx, vpListKey, vendorProductStr).Err(); err != nil {
				return xerrors.Errorf("Failed to SAdd vendorProduct. err: %w", err)
			}
			if err := pipe.SAdd(ctx, fmt.Sprintf(vpKeyFormat, c.Vendor, c.Product), c.CpeURI).Err(); err != nil {
				return xerrors.Errorf("Failed to SAdd CpeURI. err: %w", err)
			}
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
			newDeps["VendorProducts"][vendorProductStr] = map[string]struct{}{}
			delete(oldDeps["VendorProducts"], vendorProductStr)

			if c.Deprecated {
				if err := pipe.SAdd(ctx, deprecatedCPEsKey, c.CpeURI).Err(); err != nil {
					return xerrors.Errorf("Failed to set to deprecated CPE. err: %w", err)
				}
				newDeps["DeprecatedCPEs"][c.CpeURI] = map[string]struct{}{}
				delete(oldDeps["DeprecatedCPEs"], c.CpeURI)
			}
		}
		if _, err = pipe.Exec(ctx); err != nil {
			return xerrors.Errorf("Failed to exec pipeline. err: %w", err)
		}
	}
	bar.Finish()
	log15.Info(fmt.Sprintf("Refreshed %d CPEs.", len(cpes)))

	pipe := r.conn.Pipeline()
	for vendorProductStr, cpeURIs := range oldDeps["VP"] {
		for cpeURI := range cpeURIs {
			ss := strings.Split(vendorProductStr, "#")
			if err := pipe.SRem(ctx, fmt.Sprintf(vpKeyFormat, ss[0], ss[1]), cpeURI).Err(); err != nil {
				return xerrors.Errorf("Failed to SRem. err: %w", err)
			}
		}
	}
	for vendorProductStr := range oldDeps["VendorProducts"] {
		if err := pipe.SRem(ctx, vpListKey, vendorProductStr).Err(); err != nil {
			return xerrors.Errorf("Failed to SRem. err: %w", err)
		}
	}
	for cpeURI := range oldDeps["DeprecatedCPEs"] {
		if err := pipe.SRem(ctx, deprecatedCPEsKey, cpeURI).Err(); err != nil {
			return xerrors.Errorf("Failed to SRem. err: %w", err)
		}
	}

	newDepsJSON, err := json.Marshal(newDeps)
	if err != nil {
		return xerrors.Errorf("Failed to Marshal JSON. err: %w", err)
	}
	if err := pipe.HSet(ctx, depKey, string(fetchType), string(newDepsJSON)).Err(); err != nil {
		return xerrors.Errorf("Failed to Set depkey. err: %w", err)
	}
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
