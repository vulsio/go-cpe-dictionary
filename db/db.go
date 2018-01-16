package db

import (
	"fmt"

	"github.com/kotakanbe/go-cpe-dictionary/models"
)

// DB is interface for a database driver
type DB interface {
	Name() string
	CloseDB() error
	GetCpe(string) models.Cpe
	InsertCpes([]models.Cpe) error
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
