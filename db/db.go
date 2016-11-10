package db

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/cheggaaa/pb"
	"github.com/k0kubun/pp"

	"github.com/jinzhu/gorm"
	"github.com/kotakanbe/go-cpe-dictionary/config"
	"github.com/kotakanbe/go-cpe-dictionary/models"

	// Required SQLite3.  See http://jinzhu.me/gorm/database.html#connecting-to-a-database
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

var db *gorm.DB

// Init open DB connection
func Init(conf config.Config) error {
	if err := OpenDB(conf); err != nil {
		return err
	}
	if err := MigrateDB(); err != nil {
		return err
	}
	return nil
}

// OpenDB opens Database
func OpenDB(conf config.Config) (err error) {
	db, err = gorm.Open("sqlite3", conf.DBPath)
	if err != nil {
		err = fmt.Errorf("Failed to open DB. datafile: %s, err: %s", conf.DBPath, err)
		return

	}
	db.LogMode(conf.DebugSQL)
	return
}

func recconectDB(conf config.Config) error {
	var err error
	if err = db.Close(); err != nil {
		return fmt.Errorf("Failed to close DB. datafile: %s, err: %s", conf.DBPath, err)
	}
	return OpenDB(conf)
}

// MigrateDB migrates Database
func MigrateDB() error {
	log.Info("Migrating Tables")
	if err := db.AutoMigrate(
		&models.Cpe{},
	).Error; err != nil {
		return fmt.Errorf("Failed to migrate. err: %s", err)
	}

	errMsg := "Failed to create index. err: %s"
	if err := db.Model(&models.Cpe{}).
		AddUniqueIndex("idx_cpes_name", "name").Error; err != nil {
		return fmt.Errorf(errMsg, err)
	}
	return nil
}

func chunkSlice(l []models.Cpe, n int) chan []models.Cpe {
	ch := make(chan []models.Cpe)

	go func() {
		for i := 0; i < len(l); i += n {
			fromIdx := i
			toIdx := i + n
			if toIdx > len(l) {
				toIdx = len(l)
			}
			ch <- l[fromIdx:toIdx]
		}
		close(ch)
	}()
	return ch
}

// InsertCpes inserts Cpe Information into DB
func InsertCpes(cpes []models.Cpe, conf config.Config) error {
	insertedCpes := []string{}
	bar := pb.StartNew(len(cpes))

	for chunked := range chunkSlice(cpes, 100) {
		tx := db.Begin()
		for _, c := range chunked {
			bar.Increment()

			// select old record.
			old := models.Cpe{}
			r := tx.Where(&models.Cpe{Name: c.Name}).First(&old)
			if r.RecordNotFound() || old.ID == 0 {
				if err := tx.Create(&c).Error; err != nil {
					tx.Rollback()
					return fmt.Errorf("Failed to insert. cve: %s, err: %s",
						pp.Sprintf("%v", c),
						err,
					)
				}
				insertedCpes = append(insertedCpes, c.Name)
			}
		}
		tx.Commit()
	}
	bar.Finish()

	log.Infof("Inserted %d CPEs", len(insertedCpes))
	//  log.Debugf("%v", refreshedNvds)
	return nil
}

// GetCpe Select Cpe information from DB.
func GetCpe(name string) models.Cpe {
	c := models.Cpe{}
	//TODO parameter
	db.Where(&models.Cpe{Name: name}).First(&c)
	return c
}
