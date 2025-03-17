package jvn

import (
	"encoding/xml"
	"fmt"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/knqyf263/go-cpe/naming"
	"github.com/spf13/viper"
	"golang.org/x/xerrors"

	futil "github.com/vulsio/go-cpe-dictionary/fetcher/util"
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
func FetchJVN() (models.FetchedCPEs, error) {
	urls, err := makeJvnURLs()
	if err != nil {
		return models.FetchedCPEs{}, xerrors.Errorf("Failed to make URLs. err: %w", err)
	}

	cpeURIs := map[string]string{}
	rdfs, err := fetchJVNFeedFileConcurrently(urls, viper.GetInt("threads"), viper.GetInt("wait"))
	if err != nil {
		return models.FetchedCPEs{}, xerrors.Errorf("Failed to get feeds. err: %w", err)
	}
	for _, rdf := range rdfs {
		for _, item := range rdf.Items {
			for _, c := range item.Cpes {
				if _, err := naming.UnbindURI(c.Value); err != nil {
					// Logging only
					log15.Warn("Failed to unbind", c.Value, err)
					continue
				}
				cpeURIs[c.Value] = fmt.Sprintf("%s %s", c.Vendor, c.Product)
			}
		}
	}
	var fetched models.FetchedCPEs
	for c, t := range cpeURIs {
		fetched.CPEs = append(fetched.CPEs, models.FetchedCPE{
			Title: t,
			CPEs:  []string{c},
		})
	}
	return fetched, nil
}

func makeJvnURLs() ([]string, error) {
	var urls []string
	for y := 2002; y <= time.Now().Year(); y++ {
		urls = append(urls, fmt.Sprintf("https://jvndb.jvn.jp/ja/rss/years/jvndb_%d.rdf", y))
	}

	return append(urls,
		"https://jvndb.jvn.jp/ja/rss/jvndb_new.rdf",
		"https://jvndb.jvn.jp/ja/rss/jvndb.rdf",
	), nil
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
	bytes, err := futil.FetchFeedFile(url, false)
	if err != nil {
		return nil, xerrors.Errorf("Failed to fetch. url: %s, err: %w", url, err)
	}
	if err := xml.Unmarshal(bytes, &rdf); err != nil {
		return nil, xerrors.Errorf("Failed to unmarshal. url: %s, err: %w", url, err)
	}
	return rdf, nil
}
