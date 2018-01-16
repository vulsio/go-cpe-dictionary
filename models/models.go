package models

import (
	"github.com/jinzhu/gorm"
	"github.com/kotakanbe/go-cpe-dictionary/cpe"
)

// Cpe has CPE information
type Cpe struct {
	gorm.Model `json:"-" xml:"-"`

	Name      string
	NameCpe23 string
	Title     string
	TitleJa   string
}

// ConvertToModel : ConvertToModel
func ConvertToModel(cpeList cpe.List) (cpes []Cpe) {
	for _, item := range cpeList.Items {
		cpes = append(cpes, Cpe{
			Name:      item.Name,
			NameCpe23: item.Cpe23Item.Name,
			Title:     item.GetTitleEn(),
			TitleJa:   item.GetTitleJa(),
		})
	}
	return
}
