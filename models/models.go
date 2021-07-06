package models

// CategorizedCpe :
// https://cpe.mitre.org/specification/CPE_2.3_for_ITSAC_Nov2011.pdf
type CategorizedCpe struct {
	ID              int64  `json:"-"`
	CpeURI          string `gorm:"unique;size:255;index:idx_categorized_cpe_cpe_uri"`
	CpeFS           string `gorm:"size:255"`
	Part            string `gorm:"size:255"`
	Vendor          string `gorm:"size:255;index:idx_categorized_cpe_vendor"`
	Product         string `gorm:"size:255;index:idx_categorized_cpe_product"`
	Version         string `gorm:"size:255"`
	Update          string `gorm:"size:255"`
	Edition         string `gorm:"size:255"`
	Language        string `gorm:"size:255"`
	SoftwareEdition string `gorm:"size:255"`
	TargetSoftware  string `gorm:"size:255"`
	TargetHardware  string `gorm:"size:255"`
	Other           string `gorm:"size:255"`
	Deprecated      bool
}
