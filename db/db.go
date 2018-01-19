package db

import (
	"fmt"

	"github.com/kotakanbe/go-cpe-dictionary/models"
)

// DB is interface for a database driver
type DB interface {
	Name() string
	CloseDB() error
	GetCpeFromCpe22(string) (models.CategorizedCpe, error)
	GetCpeFromCpe23(string) (models.CategorizedCpe, error)
	GetCategories() (models.FilterableCategories, error)
	GetFilteredCpe(models.FilterableCategories) ([]models.CategorizedCpe, error)
	InsertCpes([]models.CategorizedCpe) error
}

// NewDB return DB accessor.
func NewDB(dbType, dbpath string, debugSQL bool) (DB, error) {
	switch dbType {
	case dialectSqlite3, dialectMysql, dialectPostgreSQL:
		return NewRDB(dbType, dbpath, debugSQL)
	case dialectRedis:
		return NewRedis(dbType, dbpath, debugSQL)
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
