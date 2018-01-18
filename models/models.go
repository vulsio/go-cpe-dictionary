package models

import (
	"github.com/jinzhu/gorm"
)

// CategorizedCpe :
// https://cpe.mitre.org/specification/CPE_2.3_for_ITSAC_Nov2011.pdf
type CategorizedCpe struct {
	gorm.Model      `json:"-" xml:"-"`
	Cpe22URI        string
	Cpe23URI        string
	Part            string
	Vendor          string
	Product         string
	Version         string
	Update          string
	Edition         string
	Language        string
	SoftwareEdition string
	TargetSoftware  string
	TargetHardware  string
	Other           string
}

// FilterableCategories :
type FilterableCategories struct {
	Part           []string
	VendorProduct  map[string][]string
	TargetSoftware []string
	TargetHardware []string
}
