package models

import (
	"time"

	"github.com/knqyf263/go-cpe/common"
	"github.com/knqyf263/go-cpe/naming"
	"github.com/spf13/viper"
	"gorm.io/gorm"

	"github.com/vulsio/go-cpe-dictionary/util"
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
	// Vuls :
	Vuls FetchType = "vls"
)

// FetchedCPEs :
type FetchedCPEs struct {
	CPEs       []FetchedCPE
	Deprecated []FetchedCPE
}

// FetchedCPE :
type FetchedCPE struct {
	Title string   `json:"title,omitempty"`
	CPEs  []string `json:"cpes,omitempty"`
}

// CategorizedCpe :
// https://cpe.mitre.org/specification/CPE_2.3_for_ITSAC_Nov2011.pdf
type CategorizedCpe struct {
	ID              int64     `json:"-"`
	FetchType       FetchType `gorm:"type:varchar(3)"`
	Title           string    `gorm:"type:text;index:idx_categorized_cpe_title,length:255"`
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

// ConvertToModels :
func ConvertToModels(cpes []FetchedCPE, fetchType FetchType, deprecated bool) []CategorizedCpe {
	reqChan := make(chan FetchedCPE, len(cpes))
	resChan := make(chan *CategorizedCpe, len(cpes))
	defer close(reqChan)
	defer close(resChan)

	go func() {
		for _, cpe := range cpes {
			reqChan <- cpe
		}
	}()

	unbindFn := naming.UnbindFS
	if fetchType == JVN {
		unbindFn = naming.UnbindURI
	}
	tasks := util.GenWorkers(viper.GetInt("threads"), 0)
	for range cpes {
		tasks <- func() {
			cpe := <-reqChan
			for _, c := range cpe.CPEs {
				wfn, err := unbindFn(c)
				if err != nil {
					resChan <- nil
				}
				resChan <- &CategorizedCpe{
					FetchType:       fetchType,
					Title:           cpe.Title,
					CpeURI:          naming.BindToURI(wfn),
					CpeFS:           naming.BindToFS(wfn),
					Part:            wfn.GetString(common.AttributePart),
					Vendor:          wfn.GetString(common.AttributeVendor),
					Product:         wfn.GetString(common.AttributeProduct),
					Version:         wfn.GetString(common.AttributeVersion),
					Update:          wfn.GetString(common.AttributeUpdate),
					Edition:         wfn.GetString(common.AttributeEdition),
					Language:        wfn.GetString(common.AttributeLanguage),
					SoftwareEdition: wfn.GetString(common.AttributeSwEdition),
					TargetSoftware:  wfn.GetString(common.AttributeTargetSw),
					TargetHardware:  wfn.GetString(common.AttributeTargetHw),
					Other:           wfn.GetString(common.AttributeOther),
					Deprecated:      deprecated,
				}
			}
		}
	}

	var converted []CategorizedCpe
	for range cpes {
		cpe := <-resChan
		if cpe != nil {
			converted = append(converted, *cpe)
		}
	}
	return converted
}

// VendorProduct :
type VendorProduct struct {
	Vendor  string
	Product string
}
