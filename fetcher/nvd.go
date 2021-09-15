package fetcher

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"math"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/knqyf263/go-cpe/common"
	"github.com/knqyf263/go-cpe/naming"
	"github.com/parnurzeal/gorequest"
	"github.com/spf13/viper"
	"github.com/vulsio/go-cpe-dictionary/models"
	"github.com/vulsio/go-cpe-dictionary/util"
)

// CpeDictionary has cpe-item list
// https://nvd.nist.gov/cpe.cfm
type CpeDictionary struct {
	Items []struct {
		Name       string `xml:"name,attr"`
		Deprecated string `xml:"deprecated,attr"`
		Cpe23Item  struct {
			Name string `xml:"name,attr"`
		} `xml:"cpe23-item"`
	} `xml:"cpe-item"`
}

// V3Feed : NvdV3Feed
// https://scap.nist.gov/schema/nvd/feed/0.1/nvd_cve_feed_json_0.1_beta.schema
type V3Feed struct {
	CVEItems []struct {
		Configurations struct {
			Nodes []struct {
				Cpe []struct {
					Cpe23URI string `json:"cpe23Uri"`
				} `json:"cpe_match"`
			} `json:"nodes"`
		} `json:"configurations"`
	} `json:"CVE_Items"`
}

// FetchNVD NVD feeds
func FetchNVD() ([]models.CategorizedCpe, error) {
	cpeURIs := map[string]models.CategorizedCpe{}

	dictCpes, err := FetchCpeDictionary()
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch cpe dictionary. err : %s", err)
	}
	for _, c := range dictCpes {
		if _, ok := cpeURIs[c.CpeURI]; !ok {
			cpeURIs[c.CpeURI] = c
		}
	}

	jsonCpes, err := FetchJSONFeed()
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch nvd JSON feed. err : %s", err)
	}
	for _, c := range jsonCpes {
		if _, ok := cpeURIs[c.CpeURI]; !ok {
			cpeURIs[c.CpeURI] = c
		}
	}

	allCpes := []models.CategorizedCpe{}
	for _, c := range cpeURIs {
		allCpes = append(allCpes, c)
	}

	return allCpes, nil
}

// FetchCpeDictionary : FetchCpeDictionary
func FetchCpeDictionary() ([]models.CategorizedCpe, error) {
	url := "http://nvd.nist.gov/feeds/xml/cpe/dictionary/official-cpe-dictionary_v2.3.xml.gz"
	log15.Info("Fetching...", "URL", url)
	resp, body, errs := gorequest.New().Proxy(viper.GetString("http-proxy")).Get(url).End()
	if len(errs) > 0 || resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP error. errs: %v, url: %s", errs, url)
	}

	b := bytes.NewBufferString(body)
	reader, err := gzip.NewReader(b)
	defer func() {
		_ = reader.Close()
	}()
	if err != nil {
		return nil, fmt.Errorf("Failed to decompress NVD feedfile. url: %s, err: %s", url, err)
	}
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("Failed to Read NVD feedfile. url: %s, err: %s", url, err)
	}

	var cpeDictionary CpeDictionary
	if err = xml.Unmarshal(bytes, &cpeDictionary); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal. url: %s, err: %s", url, err)
	}

	var cpes []models.CategorizedCpe
	if cpes, err = convertNvdCpeDictionaryToModel(cpeDictionary); err != nil {
		return nil, err
	}

	return cpes, nil
}

// FetchJSONFeed : FetchJSONFeed
func FetchJSONFeed() ([]models.CategorizedCpe, error) {
	startYear := 2002
	years, err := util.GetYearsUntilThisYear(startYear)
	if err != nil {
		return nil, err
	}

	allCpes := []models.CategorizedCpe{}
	urlBlocks := makeFeedURLBlocks(years, 2)
	for _, urls := range urlBlocks {
		nvds, err := fetchFeedFileConcurrently(urls)
		if err != nil {
			return nil, fmt.Errorf("Failed to get feeds. err : %s", err)
		}
		cpes, err := convertNvdV3FeedToModel(nvds)
		if err != nil {
			return nil, err
		}
		allCpes = append(allCpes, cpes...)
	}
	return allCpes, nil
}

// makeFeedURLBlocks : makeFeedURLBlocks
func makeFeedURLBlocks(years []int, n int) (urlBlocks [][]string) {
	//  http://nvd.nist.gov/feeds/xml/cve/nvdcve-2.0-2016.xml.gz
	formatTemplate := "https://nvd.nist.gov/feeds/json/cve/1.1/nvdcve-1.1-%d.json.gz"
	blockNum := int(math.Ceil(float64(len(years)) / float64(n)))
	urlBlocks = make([][]string, blockNum, blockNum)
	var i int
	for j := range urlBlocks {
		var urls []string
		for k := 0; k < n; k++ {
			urls = append(urls, fmt.Sprintf(formatTemplate, years[i]))
			i++
			if len(years) <= i {
				break
			}
		}
		urlBlocks[j] = urls
	}
	return urlBlocks
}

func fetchFeedFileConcurrently(urls []string) (nvds []V3Feed, err error) {
	reqChan := make(chan string, len(urls))
	resChan := make(chan V3Feed, len(urls))
	errChan := make(chan error, len(urls))
	defer close(reqChan)
	defer close(resChan)
	defer close(errChan)

	go func() {
		for _, url := range urls {
			reqChan <- url
		}
	}()

	concurrency := len(urls)
	tasks := util.GenWorkers(concurrency)
	for range urls {
		tasks <- func() {
			select {
			case url := <-reqChan:
				nvd, err := fetchFeedFile(url)
				if err != nil {
					errChan <- err
					return
				}
				resChan <- *nvd
			}
		}
	}

	errs := []error{}
	timeout := time.After(10 * 60 * time.Second)
	for range urls {
		select {
		case nvd := <-resChan:
			nvds = append(nvds, nvd)
		case err := <-errChan:
			errs = append(errs, err)
		case <-timeout:
			return nvds, fmt.Errorf("Timeout Fetching Nvd")
		}
	}
	if 0 < len(errs) {
		return nvds, fmt.Errorf("%s", errs)
	}
	return nvds, nil
}

func fetchFeedFile(url string) (nvd *V3Feed, err error) {
	bytes, err := util.FetchFeedFile(url, true)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch. url: %s, err: %s", url, err)
	}
	if err = json.Unmarshal(bytes, &nvd); err != nil {
		return nil, fmt.Errorf("Failed to unmarshal. url: %s, err: %s", url, err)
	}
	return nvd, nil
}

// convertNvdCpeDictionaryToModel :
func convertNvdCpeDictionaryToModel(nvd CpeDictionary) (cpes []models.CategorizedCpe, err error) {
	for _, item := range nvd.Items {
		var wfn common.WellFormedName
		if wfn, err = naming.UnbindFS(item.Cpe23Item.Name); err != nil {
			// Logging only
			log15.Warn("Failed to unbind", item.Cpe23Item.Name, err)
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
			Deprecated:      item.Deprecated == "true",
		})
	}
	return cpes, nil
}

// convertNvdV3FeedToModel :
func convertNvdV3FeedToModel(nvds []V3Feed) (cpes []models.CategorizedCpe, err error) {
	for _, nvd := range nvds {
		for _, item := range nvd.CVEItems {
			for _, node := range item.Configurations.Nodes {
				for _, cpe := range node.Cpe {
					var wfn common.WellFormedName
					if wfn, err = naming.UnbindFS(cpe.Cpe23URI); err != nil {
						log15.Warn("Failed to unbind cpe.", "CPE URI", cpe.Cpe23URI, "err", err)
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
			}
		}
	}
	return cpes, nil
}
