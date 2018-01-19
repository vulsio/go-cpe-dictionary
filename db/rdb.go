package db

import (
	"fmt"

	"github.com/cheggaaa/pb"
	"github.com/jinzhu/gorm"
	"github.com/k0kubun/pp"
	"github.com/kotakanbe/go-cpe-dictionary/models"
	log "github.com/sirupsen/logrus"
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

	log.Debugf("Opening DB (%s).", driver.Name())
	if err = driver.OpenDB(dbType, dbpath, debugSQL); err != nil {
		return
	}

	log.Debugf("Migrating DB (%s).", driver.Name())
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
		AddUniqueIndex("idx_cpes_name23", "cpe23_uri").Error; err != nil {
		return fmt.Errorf(errMsg, err)
	}
	return nil
}

// InsertCpes inserts Cpe Information into DB
func (r *RDBDriver) InsertCpes(cpes []models.CategorizedCpe) error {
	insertedCpes := []string{}
	bar := pb.StartNew(len(cpes))

	for chunked := range chunkSlice(cpes, 100) {
		tx := r.conn.Begin()
		for _, c := range chunked {
			bar.Increment()

			// select old record.
			old := models.CategorizedCpe{}
			r := tx.Where(&models.CategorizedCpe{Cpe22URI: c.Cpe22URI}).First(&old)
			if r.RecordNotFound() || old.ID == 0 {
				if err := tx.Create(&c).Error; err != nil {
					tx.Rollback()
					return fmt.Errorf("Failed to insert. cve: %s, err: %s",
						pp.Sprintf("%v", c),
						err,
					)
				}
				insertedCpes = append(insertedCpes, c.Cpe22URI)
			}
		}
		tx.Commit()
	}
	bar.Finish()

	log.Infof("Inserted %d CPEs", len(insertedCpes))
	//  log.Debugf("%v", refreshedNvds)
	return nil
}

// GetCpeFromCpe22 Select Cpe information from DB.
func (r *RDBDriver) GetCpeFromCpe22(name string) (cpe models.CategorizedCpe, err error) {
	c := models.CategorizedCpe{}
	//TODO parameter
	r.conn.Where(&models.CategorizedCpe{Cpe22URI: name}).First(&c)
	return c, nil
}

// GetCpeFromCpe23 Select Cpe information from DB.
func (r *RDBDriver) GetCpeFromCpe23(name string) (cpe models.CategorizedCpe, err error) {
	c := models.CategorizedCpe{}
	//TODO parameter
	r.conn.Where(&models.CategorizedCpe{Cpe23URI: name}).First(&c)
	return c, nil
}

// GetCategories : GetCategories
func (r *RDBDriver) GetCategories() (cpe models.FilterableCategories, err error) {
	// TODO
	return models.FilterableCategories{}, nil
}

// GetFilteredCpe : GetFilteredCpe
func (r *RDBDriver) GetFilteredCpe(filters models.FilterableCategories) (cpes []models.CategorizedCpe, err error) {
	// TODO
	return cpes, nil
}

// CloseDB close Database
func (r *RDBDriver) CloseDB() (err error) {
	if err = r.conn.Close(); err != nil {
		log.Errorf("Failed to close DB. Type: %s. err: %s", r.name, err)
		return
	}
	return
}
