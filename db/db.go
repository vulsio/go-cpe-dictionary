package db

import (
	"fmt"

	"github.com/inconshreveable/log15"
	"github.com/kotakanbe/go-cpe-dictionary/models"
)

// DB is interface for a database driver
type DB interface {
	Name() string
	OpenDB(dbType, dbPath string, debugSQL bool) (bool, error)
	CloseDB() error
	MigrateDB() error

	GetVendorProducts() ([]string, error)
	GetCpesByVendorProduct(string, string) ([]string, []string, error)
	InsertCpes([]models.CategorizedCpe) error
	IsDeprecated(string) (bool, error)
}

// NewDB returns db driver
func NewDB(dbType string, dbPath string, debugSQL bool) (driver DB, locked bool, err error) {
	if driver, err = newDB(dbType); err != nil {
		log15.Error("Failed to new db.", "err", err)
		return driver, false, err
	}

	if locked, err := driver.OpenDB(dbType, dbPath, debugSQL); err != nil {
		if locked {
			return nil, true, err
		}
		return nil, false, err
	}

	if err := driver.MigrateDB(); err != nil {
		log15.Error("Failed to migrate db.", "err", err)
		return driver, false, err
	}
	return driver, false, nil
}

func newDB(dbType string) (DB, error) {
	switch dbType {
	case dialectSqlite3, dialectMysql, dialectPostgreSQL:
		return &RDBDriver{name: dbType}, nil
	case dialectRedis:
		return &RedisDriver{name: dbType}, nil
	}
	return nil, fmt.Errorf("Invalid database dialect, %s", dbType)
}

func chunkSlice(l []models.CategorizedCpe, n int) chan []models.CategorizedCpe {
	ch := make(chan []models.CategorizedCpe)
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
