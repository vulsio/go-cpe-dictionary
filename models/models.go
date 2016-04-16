package models

import (
	"github.com/jinzhu/gorm"
	"github.com/kotakanbe/go-cpe-dictionary/cpe"
)

// CpeItem has CPE information
type Cpe struct {
	gorm.Model

	Name      string
	NameCpe23 string
	Title     string
	TitleJa   string
}

func ConvertToModel(cpeList cpe.CpeList) (cpes []Cpe) {
	for _, item := range cpeList.CpeItems {
		cpes = append(cpes, Cpe{
			Name:      item.Name,
			NameCpe23: item.Cpe23Item.Name,
			Title:     item.GetTitleEn(),
			TitleJa:   item.GetTitleJa(),
		})
	}
	return
}
