package fetcher

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/knqyf263/go-cpe/common"
	"github.com/knqyf263/go-cpe/naming"
	"github.com/spf13/viper"
	"golang.org/x/xerrors"

	"github.com/vulsio/go-cpe-dictionary/models"
	"github.com/vulsio/go-cpe-dictionary/util"
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
	rdfs, err := fetchJVNFeedFileConcurrently(urls, viper.GetInt("threads"), viper.GetInt("wait"))
	if err != nil {
		return nil, xerrors.Errorf("Failed to get feeds. err: %w", err)
	}
	for _, rdf := range rdfs {
		for _, item := range rdf.Items {
			cpes, err := convertJvnCpesToModel(item.Cpes)
			if err != nil {
				return nil, xerrors.Errorf("Failed to convert. err: %w", err)
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

func fetchJVNFeedFileConcurrently(urls []string, concurrency, wait int) (rdfs []rdf, err error) {
	reqChan := make(chan string, len(urls))
	resChan := make(chan rdf, len(urls))
	errChan := make(chan error, len(urls))
	defer close(reqChan)
	defer close(resChan)
	defer close(errChan)

	go func() {
		for _, url := range urls {
			reqChan <- url
		}
	}()

	tasks := util.GenWorkers(concurrency, wait)
	for range urls {
		tasks <- func() {
			select {
			case url := <-reqChan:
				rdf, err := fetchJVNFeedFile(url)
				if err != nil {
					errChan <- err
					return
				}
				resChan <- *rdf
			}
		}
	}

	errs := []error{}
	timeout := time.After(10 * 60 * time.Second)
	for range urls {
		select {
		case rdf := <-resChan:
			rdfs = append(rdfs, rdf)
		case err := <-errChan:
			errs = append(errs, err)
		case <-timeout:
			return rdfs, xerrors.Errorf("Timeout Fetching JVN")
		}
	}
	if 0 < len(errs) {
		return rdfs, xerrors.Errorf("%s", errs)
	}
	return rdfs, nil
}

func fetchJVNFeedFile(url string) (rdf *rdf, err error) {
	bytes, err := util.FetchFeedFile(url, false)
	if err != nil {
		return nil, xerrors.Errorf("Failed to fetch. url: %s, err: %w", url, err)
	}
	if err = xml.Unmarshal(bytes, &rdf); err != nil {
		return nil, xerrors.Errorf("Failed to unmarshal. url: %s, err: %w", url, err)
	}
	return rdf, nil
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
			FetchType:       models.JVN,
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
