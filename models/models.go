package models

import (
	"time"

	"gorm.io/gorm"
)

// LatestSchemaVersion manages the Schema version used in the latest go-cpe-dictionary.
const LatestSchemaVersion = 2

// FetchMeta has meta information about fetched data
type FetchMeta struct {
	gorm.Model        `json:"-"`
	GoCPEDictRevision string
	SchemaVersion     uint
	LastFetchedAt     time.Time
}

// OutDated checks whether last fetched feed is out dated
func (f FetchMeta) OutDated() bool {
	return f.SchemaVersion != LatestSchemaVersion
}

// FetchType :
type FetchType string

const (
	// NVD :
	NVD FetchType = "nvd"
	// JVN :
	JVN FetchType = "jvn"
)

// CategorizedCpe :
// https://cpe.mitre.org/specification/CPE_2.3_for_ITSAC_Nov2011.pdf
type CategorizedCpe struct {
	ID              int64     `json:"-"`
	FetchType       FetchType `gorm:"type:varchar(3)"`
	CpeURI          string    `gorm:"type:varchar(255);index:idx_categorized_cpe_cpe_uri"`
	CpeFS           string    `gorm:"type:varchar(255)"`
	Part            string    `gorm:"type:varchar(255)"`
	Vendor          string    `gorm:"type:varchar(255);index:idx_categorized_cpe_vendor"`
	Product         string    `gorm:"type:varchar(255);index:idx_categorized_cpe_product"`
	Version         string    `gorm:"type:varchar(255)"`
	Update          string    `gorm:"type:varchar(255)"`
	Edition         string    `gorm:"type:varchar(255)"`
	Language        string    `gorm:"type:varchar(255)"`
	SoftwareEdition string    `gorm:"type:varchar(255)"`
	TargetSoftware  string    `gorm:"type:varchar(255)"`
	TargetHardware  string    `gorm:"type:varchar(255)"`
	Other           string    `gorm:"type:varchar(255)"`
	Deprecated      bool
}

type VendorProduct struct {
	Vendor  string
	Product string
}
