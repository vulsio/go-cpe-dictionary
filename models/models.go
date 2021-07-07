package models

// CategorizedCpe :
// https://cpe.mitre.org/specification/CPE_2.3_for_ITSAC_Nov2011.pdf
type CategorizedCpe struct {
	ID              int64  `json:"-"`
	CpeURI          string `gorm:"unique;type:varchar(255);index:idx_categorized_cpe_cpe_uri"`
	CpeFS           string `gorm:"type:varchar(255)"`
	Part            string `gorm:"type:varchar(255)"`
	Vendor          string `gorm:"type:varchar(255);index:idx_categorized_cpe_vendor"`
	Product         string `gorm:"type:varchar(255);index:idx_categorized_cpe_product"`
	Version         string `gorm:"type:varchar(255)"`
	Update          string `gorm:"type:varchar(255)"`
	Edition         string `gorm:"type:varchar(255)"`
	Language        string `gorm:"type:varchar(255)"`
	SoftwareEdition string `gorm:"type:varchar(255)"`
	TargetSoftware  string `gorm:"type:varchar(255)"`
	TargetHardware  string `gorm:"type:varchar(255)"`
	Other           string `gorm:"type:varchar(255)"`
	Deprecated      bool
}
