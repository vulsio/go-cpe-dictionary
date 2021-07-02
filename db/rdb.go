package db

import (
	"fmt"

	"github.com/cheggaaa/pb/v3"
	"github.com/jinzhu/gorm"
	"github.com/k0kubun/pp"
	"github.com/kotakanbe/go-cpe-dictionary/models"
	sqlite3 "github.com/mattn/go-sqlite3"
	"golang.org/x/xerrors"

	// Required MySQL.  See http://jinzhu.me/gorm/database.html#connecting-to-a-database
	_ "github.com/jinzhu/gorm/dialects/mysql"
	_ "github.com/jinzhu/gorm/dialects/postgres"

	// Required SQLite3.
	_ "github.com/jinzhu/gorm/dialects/sqlite"
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

// Name return db name
func (r *RDBDriver) Name() string {
	return r.name
}

// OpenDB opens Database
func (r *RDBDriver) OpenDB(dbType, dbPath string, debugSQL bool) (locked bool, err error) {
	r.conn, err = gorm.Open(dbType, dbPath)
	if err != nil {
		msg := fmt.Sprintf("Failed to open DB. dbtype: %s, dbpath: %s, err: %s", dbType, dbPath, err)
		if r.name == dialectSqlite3 {
			switch err.(sqlite3.Error).Code {
			case sqlite3.ErrLocked, sqlite3.ErrBusy:
				return true, fmt.Errorf(msg)
			}
		}
		return false, fmt.Errorf(msg)
	}
	r.conn.LogMode(debugSQL)
	if r.name == dialectSqlite3 {
		r.conn.Exec("PRAGMA foreign_keys = ON")
	}
	return false, nil
}

// CloseDB close Database
func (r *RDBDriver) CloseDB() (err error) {
	if r.conn == nil {
		return
	}
	if err = r.conn.Close(); err != nil {
		return xerrors.Errorf("Failed to close DB. Type: %s. err: %w", r.name, err)
	}
	return
}

// MigrateDB migrates Database
func (r *RDBDriver) MigrateDB() error {
	if err := r.conn.AutoMigrate(
		&models.CategorizedCpe{},
	).Error; err != nil {
		return fmt.Errorf("Failed to migrate. err: %s", err)
	}
	return nil
}

// GetVendorProducts : GetVendorProducts
func (r *RDBDriver) GetVendorProducts() (vendorProducts []string, err error) {
	var results []struct {
		Vendor  string
		Product string
	}

	// TODO Is there a better way to use distinct with GORM? Needing
	// explicit column names seems like an antipattern for an orm.
	if err = r.conn.Select("DISTINCT vendor, product").Find(&models.CategorizedCpe{}).Scan(&results).Error; err != nil && err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("Failed to select results. err: %s", err)
	}

	for _, vp := range results {
		vendorProducts = append(vendorProducts, fmt.Sprintf("%s::%s", vp.Vendor, vp.Product))
	}
	return
}

// GetCpesByVendorProduct : GetCpesByVendorProduct
func (r *RDBDriver) GetCpesByVendorProduct(vendor, product string) ([]string, []string, error) {
	results := []models.CategorizedCpe{}
	err := r.conn.Select("DISTINCT cpe_uri, deprecated").Find(&results, "vendor LIKE ? and product LIKE ?", vendor, product).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, nil, fmt.Errorf("Failed to select results. err: %s", err)
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

// InsertCpes inserts Cpe Information into DB
func (r *RDBDriver) InsertCpes(cpes []models.CategorizedCpe) error {
	return r.deleteAndInsertCpes(r.conn, cpes)
}

func (r *RDBDriver) deleteAndInsertCpes(conn *gorm.DB, cpes []models.CategorizedCpe) (err error) {
	bar := pb.StartNew(len(cpes))
	tx := conn.Begin()
	defer func() {
		if err != nil {
			tx.Rollback()
			return
		}
		tx.Commit()
	}()

	for _, c := range cpes {
		if err := tx.FirstOrCreate(&c, models.CategorizedCpe{CpeURI: c.CpeURI}).Error; err != nil {
			return fmt.Errorf("Failed to insert. cpe: %s, err: %s",
				pp.Sprintf("%v", c), err)
		}
		bar.Increment()
	}
	bar.Finish()

	return nil
}

// IsDeprecated : IsDeprecated
func (r *RDBDriver) IsDeprecated(cpeURI string) (bool, error) {
	// not implemented yet
	return false, nil
}
