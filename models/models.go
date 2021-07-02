package models

// CategorizedCpe :
// https://cpe.mitre.org/specification/CPE_2.3_for_ITSAC_Nov2011.pdf
type CategorizedCpe struct {
	ID              int64  `json:"-"`
	CpeURI          string `gorm:"index:idx_categorized_cpe_cpe_uri"`
	CpeFS           string
	Part            string
	Vendor          string `gorm:"index:idx_categorized_cpe_vendor"`
	Product         string `gorm:"index:idx_categorized_cpe_product"`
	Version         string
	Update          string
	Edition         string
	Language        string
	SoftwareEdition string
	TargetSoftware  string
	TargetHardware  string
	Other           string
	Deprecated      bool
}
