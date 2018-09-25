package db

import (
	"fmt"

	"github.com/cheggaaa/pb"
	"github.com/inconshreveable/log15"
	"github.com/jinzhu/gorm"
	"github.com/k0kubun/pp"
	"github.com/kotakanbe/go-cpe-dictionary/models"
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

// NewRDB return RDB driver
func NewRDB(dbType, dbpath string, debugSQL bool) (driver *RDBDriver, err error) {
	driver = &RDBDriver{
		name: dbType,
	}

	log15.Debug("Opening DB", "db", driver.Name())
	if err = driver.OpenDB(dbType, dbpath, debugSQL); err != nil {
		return
	}

	log15.Debug("Migrating DB.", "db", driver.Name())
	if err = driver.MigrateDB(); err != nil {
		return
	}
	return
}

// OpenDB opens Database
func (r *RDBDriver) OpenDB(dbType, dbPath string, debugSQL bool) (err error) {
	r.conn, err = gorm.Open(dbType, dbPath)
	if err != nil {
		err = fmt.Errorf("Failed to open DB. dbtype: %s, dbpath: %s, err: %s", dbType, dbPath, err)
		return
	}
	r.conn.LogMode(debugSQL)
	if r.name == dialectSqlite3 {
		r.conn.Exec("PRAGMA journal_mode=WAL;")
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

	errMsg := "Failed to create index. err: %s"
	if err := r.conn.Model(&models.CategorizedCpe{}).
		AddUniqueIndex("idx_cpes_uri", "cpe_uri").Error; err != nil {
		return fmt.Errorf(errMsg, err)
	}
	return nil
}

// InsertCpes inserts Cpe Information into DB
func (r *RDBDriver) InsertCpes(cpes []*models.CategorizedCpe) error {
	insertedCpes := []string{}
	bar := pb.StartNew(len(cpes))

	for chunked := range chunkSlice(cpes, 100) {
		tx := r.conn.Begin()
		for _, c := range chunked {
			bar.Increment()

			// select old record.
			old := models.CategorizedCpe{}
			r := tx.Where(&models.CategorizedCpe{CpeURI: c.CpeURI}).First(&old)
			if r.RecordNotFound() || old.ID == 0 {
				if err := tx.Create(&c).Error; err != nil {
					tx.Rollback()
					return fmt.Errorf("Failed to insert. cve: %s, err: %s",
						pp.Sprintf("%v", c),
						err,
					)
				}
				insertedCpes = append(insertedCpes, c.CpeURI)
			}
		}
		tx.Commit()
	}
	bar.Finish()

	log15.Info(fmt.Sprintf("Inserted %d CPEs", len(insertedCpes)))
	//  log.Debugf("%v", refreshedNvds)
	return nil
}

// GetVendorProducts : GetVendorProducts
func (r *RDBDriver) GetVendorProducts() (vendorProducts []string, err error) {
	// TODO
	return nil, nil
}

// GetCpesByVendorProduct : GetCpesByVendorProduct
func (r *RDBDriver) GetCpesByVendorProduct(vendor, product string) (cpeURIs []string, err error) {
	// TODO
	return nil, nil
}

// CloseDB close Database
func (r *RDBDriver) CloseDB() (err error) {
	if err = r.conn.Close(); err != nil {
		log15.Error("Failed to close DB.", "Type", r.name, "err", err)
		return
	}
	return
}
