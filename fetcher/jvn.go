package fetcher

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/knqyf263/go-cpe/common"
	"github.com/knqyf263/go-cpe/naming"
	"github.com/kotakanbe/go-cpe-dictionary/models"
	"github.com/kotakanbe/go-cpe-dictionary/util"
)

type rdf struct {
	Items []Item `xml:"item"`
}

// Item ... http://jvndb.jvn.jp/apis/getVulnOverviewList_api.html
type Item struct {
	Cpes []cpe `xml:"cpe"`
}

type cpe struct {
	Version string `xml:"version,attr"` // cpe:/a:mysql:mysql
	Vendor  string `xml:"vendor,attr"`
	Product string `xml:"product,attr"`
	Value   string `xml:",chardata"`
}

// FetchJVN JVN feeds
func FetchJVN() ([]models.CategorizedCpe, error) {
	years, err := util.GetYearsUntilThisYear(2002)
	if err != nil {
		return nil, err
	}
	urls := makeJvnURLs(years)

	cpeURIs := map[string]models.CategorizedCpe{}
	for _, url := range urls {
		bytes, err := util.FetchFeedFile(url, false)
		if err != nil {
			return nil, fmt.Errorf("Failed to fetch. url: %s, err: %s", url, err)
		}
		var rdf rdf
		if err = xml.Unmarshal(bytes, &rdf); err != nil {
			return nil, fmt.Errorf("Failed to unmarshal. url: %s, err: %s", url, err)
		}

		for _, item := range rdf.Items {
			cpes, err := convertJvnCpesToModel(item.Cpes)
			if err != nil {
				return nil, fmt.Errorf("Failed to convert. err: %s", err)
			}

			for _, c := range cpes {
				if _, ok := cpeURIs[c.CpeURI]; !ok {
					cpeURIs[c.CpeURI] = c
				}
			}
		}
	}

	allCpes := []models.CategorizedCpe{}
	for _, c := range cpeURIs {
		allCpes = append(allCpes, c)
	}

	return allCpes, nil
}

func makeJvnURLs(years []int) (urls []string) {
	latestFeeds := []string{
		"https://jvndb.jvn.jp/ja/rss/jvndb_new.rdf",
		"https://jvndb.jvn.jp/ja/rss/jvndb.rdf",
	}

	if len(years) == 0 {
		return latestFeeds
	}

	urlFormat := "https://jvndb.jvn.jp/ja/rss/years/jvndb_%d.rdf"
	for _, year := range years {
		urls = append(urls, fmt.Sprintf(urlFormat, year))

		thisYear := time.Now().Year()
		if year == thisYear {
			urls = append(urls, latestFeeds...)
		}
	}
	return
}

func convertJvnCpesToModel(jvnCpes []cpe) (cpes []models.CategorizedCpe, err error) {
	for _, c := range jvnCpes {
		var wfn common.WellFormedName
		if wfn, err = naming.UnbindURI(c.Value); err != nil {
			// Logging only
			log15.Warn("Failed to unbind", c.Value, err)
			continue
		}
		cpes = append(cpes, models.CategorizedCpe{
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
		})
	}
	return cpes, nil
}
