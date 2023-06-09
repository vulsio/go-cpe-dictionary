package db

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"os"
	"time"

	"github.com/cheggaaa/pb/v3"
	"github.com/glebarez/sqlite"
	"github.com/hbollon/go-edlib"
	"github.com/inconshreveable/log15"
	"github.com/spf13/viper"
	"golang.org/x/xerrors"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/vulsio/go-cpe-dictionary/config"
	"github.com/vulsio/go-cpe-dictionary/models"
)

// Supported DB dialects.
const (
	dialectSqlite3    = "sqlite3"
	dialectMysql      = "mysql"
	dialectPostgreSQL = "postgres"
)

// RDBDriver is Driver for RDB
type RDBDriver struct {
	name string
	conn *gorm.DB
}

// https://github.com/mattn/go-sqlite3/blob/edc3bb69551dcfff02651f083b21f3366ea2f5ab/error.go#L18-L66
type errNo int

type sqliteError struct {
	Code errNo /* The error code returned by SQLite */
}

// result codes from http://www.sqlite.org/c3ref/c_abort.html
var (
	errBusy   = errNo(5) /* The database file is locked */
	errLocked = errNo(6) /* A table in the database is locked */
)

// ErrDBLocked :
var ErrDBLocked = xerrors.New("database is locked")

// Name return db name
func (r *RDBDriver) Name() string {
	return r.name
}

// OpenDB opens Database
func (r *RDBDriver) OpenDB(dbType, dbPath string, debugSQL bool, _ Option) (err error) {
	gormConfig := gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
		Logger: logger.New(
			log.New(os.Stderr, "\r\n", log.LstdFlags),
			logger.Config{
				LogLevel: logger.Silent,
			},
		),
	}

	if debugSQL {
		gormConfig.Logger = logger.New(
			log.New(os.Stderr, "\r\n", log.LstdFlags),
			logger.Config{
				SlowThreshold: time.Second,
				LogLevel:      logger.Info,
				Colorful:      true,
			},
		)
	}

	switch r.name {
	case dialectSqlite3:
		r.conn, err = gorm.Open(sqlite.Open(dbPath), &gormConfig)
		if err != nil {
			parsedErr, marshalErr := json.Marshal(err)
			if marshalErr != nil {
				return xerrors.Errorf("Failed to marshal err. err: %w", marshalErr)
			}

			var errMsg sqliteError
			if unmarshalErr := json.Unmarshal(parsedErr, &errMsg); unmarshalErr != nil {
				return xerrors.Errorf("Failed to unmarshal. err: %w", unmarshalErr)
			}

			switch errMsg.Code {
			case errBusy, errLocked:
				return xerrors.Errorf("Failed to open DB. dbtype: %s, dbpath: %s, err: %w", dbType, dbPath, ErrDBLocked)
			default:
				return xerrors.Errorf("Failed to open DB. dbtype: %s, dbpath: %s, err: %w", dbType, dbPath, err)
			}
		}

		r.conn.Exec("PRAGMA foreign_keys = ON")
	case dialectMysql:
		r.conn, err = gorm.Open(mysql.Open(dbPath), &gormConfig)
		if err != nil {
			return xerrors.Errorf("Failed to open DB. dbtype: %s, dbpath: %s, err: %w", dbType, dbPath, err)
		}
	case dialectPostgreSQL:
		r.conn, err = gorm.Open(postgres.Open(dbPath), &gormConfig)
		if err != nil {
			return xerrors.Errorf("Failed to open DB. dbtype: %s, dbpath: %s, err: %w", dbType, dbPath, err)
		}
	default:
		return xerrors.Errorf("Not Supported DB dialects. r.name: %s", r.name)
	}
	return nil
}

// CloseDB close Database
func (r *RDBDriver) CloseDB() (err error) {
	if r.conn == nil {
		return
	}

	var sqlDB *sql.DB
	if sqlDB, err = r.conn.DB(); err != nil {
		return xerrors.Errorf("Failed to get DB Object. err: %w", err)
	}
	if err = sqlDB.Close(); err != nil {
		return xerrors.Errorf("Failed to close DB. Type: %s. err: %w", r.name, err)
	}
	return
}

// MigrateDB migrates Database
func (r *RDBDriver) MigrateDB() error {
	if err := r.conn.AutoMigrate(
		&models.FetchMeta{},
		&models.CategorizedCpe{},
	); err != nil {
		switch r.name {
		case dialectSqlite3:
			if r.name == dialectSqlite3 {
				parsedErr, marshalErr := json.Marshal(err)
				if marshalErr != nil {
					return xerrors.Errorf("Failed to marshal err. err: %w", marshalErr)
				}

				var errMsg sqliteError
				if unmarshalErr := json.Unmarshal(parsedErr, &errMsg); unmarshalErr != nil {
					return xerrors.Errorf("Failed to unmarshal. err: %w", unmarshalErr)
				}

				switch errMsg.Code {
				case errBusy, errLocked:
					return xerrors.Errorf("Failed to migrate. err: %w", ErrDBLocked)
				default:
					return xerrors.Errorf("Failed to migrate. err: %w", err)
				}
			}
		case dialectMysql, dialectPostgreSQL:
			if err != nil {
				return xerrors.Errorf("Failed to migrate. err: %w", err)
			}
		default:
			return xerrors.Errorf("Not Supported DB dialects. r.name: %s", r.name)
		}
	}
	return nil
}

// IsGoCPEDictModelV1 determines if the DB was created at the time of go-cpe-dictionary Model v1
func (r *RDBDriver) IsGoCPEDictModelV1() (bool, error) {
	if r.conn.Migrator().HasTable(&models.FetchMeta{}) {
		return false, nil
	}

	var (
		count int64
		err   error
	)
	switch r.name {
	case dialectSqlite3:
		err = r.conn.Table("sqlite_master").Where("type = ?", "table").Count(&count).Error
	case dialectMysql:
		err = r.conn.Table("information_schema.tables").Where("table_schema = ?", r.conn.Migrator().CurrentDatabase()).Count(&count).Error
	case dialectPostgreSQL:
		err = r.conn.Table("pg_tables").Where("schemaname = ?", "public").Count(&count).Error
	}

	if count > 0 {
		return true, nil
	}
	return false, err
}

// GetFetchMeta get FetchMeta from Database
func (r *RDBDriver) GetFetchMeta() (fetchMeta *models.FetchMeta, err error) {
	if err = r.conn.Take(&fetchMeta).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		return &models.FetchMeta{GoCPEDictRevision: config.Revision, SchemaVersion: models.LatestSchemaVersion, LastFetchedAt: time.Date(1000, time.January, 1, 0, 0, 0, 0, time.UTC)}, nil
	}

	return fetchMeta, nil
}

// UpsertFetchMeta upsert FetchMeta to Database
func (r *RDBDriver) UpsertFetchMeta(fetchMeta *models.FetchMeta) error {
	fetchMeta.GoCPEDictRevision = config.Revision
	fetchMeta.SchemaVersion = models.LatestSchemaVersion
	return r.conn.Save(fetchMeta).Error
}

// GetVendorProducts : GetVendorProducts
func (r *RDBDriver) GetVendorProducts() ([]models.VendorProduct, []models.VendorProduct, error) {
	cpes := []models.CategorizedCpe{}
	if err := r.conn.Distinct("vendor", "product", "deprecated").Find(&cpes).Error; err != nil {
		return nil, nil, xerrors.Errorf("Failed to select results. err: %w", err)
	}

	vendorProducts := []models.VendorProduct{}
	deprecated := []models.VendorProduct{}
	for _, c := range cpes {
		vp := models.VendorProduct{
			Vendor:  c.Vendor,
			Product: c.Product,
		}
		if c.Deprecated {
			deprecated = append(deprecated, vp)
		} else {
			vendorProducts = append(vendorProducts, vp)
		}
	}

	return vendorProducts, deprecated, nil
}

// GetCpesByVendorProduct : GetCpesByVendorProduct
func (r *RDBDriver) GetCpesByVendorProduct(vendor, product string) ([]string, []string, error) {
	results := []models.CategorizedCpe{}
	if err := r.conn.Distinct("cpe_uri", "deprecated").Find(&results, "vendor LIKE ? and product LIKE ?", vendor, product).Error; err != nil {
		return nil, nil, xerrors.Errorf("Failed to select results. err: %w", err)
	}
	cpeURIs, deprecated := []string{}, []string{}
	for _, r := range results {
		if r.Deprecated {
			deprecated = append(deprecated, r.CpeURI)
		} else {
			cpeURIs = append(cpeURIs, r.CpeURI)
		}
	}
	return cpeURIs, deprecated, nil
}

// GetSimilarCpesByTitle : GetSimilarCpesByTitle
func (r *RDBDriver) GetSimilarCpesByTitle(query string, n int, algorithm edlib.Algorithm) ([]models.FetchedCPE, error) {
	if query == "" || n <= 0 {
		return nil, nil
	}

	var titles []string
	if err := r.conn.Model(&models.CategorizedCpe{}).Distinct("title").Find(&titles).Error; err != nil {
		return nil, xerrors.Errorf("Failed to select title. err: %w", err)
	}

	if len(titles) < n {
		n = len(titles)
	}

	ss, err := edlib.FuzzySearchSet(query, titles, n, algorithm)
	if err != nil {
		return nil, xerrors.Errorf("Failed to fuzzy search. err: %w", err)
	}

	ranks := make([]models.FetchedCPE, 0, n)
	for _, s := range ss {
		c := models.FetchedCPE{Title: s}
		if err := r.conn.Model(&models.CategorizedCpe{}).Distinct("cpe_uri").Where("title = ?", s).Find(&c.CPEs).Error; err != nil {
			return nil, xerrors.Errorf("Failed to select cpe_uri. err: %w", err)
		}
		ranks = append(ranks, c)
	}

	return ranks, nil
}

// InsertCpes inserts Cpe Information into DB
func (r *RDBDriver) InsertCpes(fetchType models.FetchType, cpes models.FetchedCPEs) error {
	return r.deleteAndInsertCpes(r.conn, fetchType, cpes)
}

func (r *RDBDriver) deleteAndInsertCpes(conn *gorm.DB, fetchType models.FetchType, cpes models.FetchedCPEs) (err error) {
	tx := conn.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		tx.Commit()
	}()

	batchSize := viper.GetInt("batch-size")
	if batchSize < 1 {
		return xerrors.New("Failed to set batch-size. err: batch-size option is not set properly")
	}

	// Delete all old records
	oldIDs := []int64{}
	result := tx.Model(models.CategorizedCpe{}).Select("id").Where("fetch_type = ?", fetchType).Find(&oldIDs)
	if result.Error != nil {
		return xerrors.Errorf("Failed to select old defs: %w", result.Error)
	}

	if result.RowsAffected > 0 {
		log15.Info("Deleting old CPEs")
		bar := pb.StartNew(len(oldIDs))
		for idx := range chunkSlice(len(oldIDs), batchSize) {
			if err := tx.Where("id IN ?", oldIDs[idx.From:idx.To]).Delete(&models.CategorizedCpe{}).Error; err != nil {
				return xerrors.Errorf("Failed to delete: %w", err)
			}
			bar.Add(idx.To - idx.From)
		}
		bar.Finish()
	}

	log15.Info("Inserting new CPEs")
	bar := pb.StartNew(len(cpes.CPEs) + len(cpes.Deprecated))
	for _, in := range []struct {
		cpes       []models.FetchedCPE
		deprecated bool
	}{
		{
			cpes:       cpes.CPEs,
			deprecated: false,
		},
		{
			cpes:       cpes.Deprecated,
			deprecated: true,
		},
	} {
		for idx := range chunkSlice(len(in.cpes), batchSize) {
			if err := tx.Create(models.ConvertToModels(in.cpes[idx.From:idx.To], fetchType, in.deprecated)).Error; err != nil {
				return xerrors.Errorf("Failed to insert. err: %w", err)
			}
			bar.Add(idx.To - idx.From)
		}
	}
	bar.Finish()

	return nil
}

// IsDeprecated : IsDeprecated
func (r *RDBDriver) IsDeprecated(_ string) (bool, error) {
	// not implemented yet
	return false, nil
}
