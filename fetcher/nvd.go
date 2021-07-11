package fetcher

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/inconshreveable/log15"
	"github.com/knqyf263/go-cpe/common"
	"github.com/knqyf263/go-cpe/naming"
	"github.com/kotakanbe/go-cpe-dictionary/models"
	"github.com/kotakanbe/go-cpe-dictionary/util"
	"github.com/parnurzeal/gorequest"
	"github.com/spf13/viper"
)

// CpeDictionary has cpe-item list
// https://nvd.nist.gov/cpe.cfm
type CpeDictionary struct {
	Items []CpeDictionaryItem `xml:"cpe-item"`
}

// CpeDictionaryItem :
type CpeDictionaryItem struct {
	Name       string `xml:"name,attr"`
	Deprecated string `xml:"deprecated,attr"`
	Cpe23Item  struct {
		Name string `xml:"name,attr"`
	} `xml:"cpe23-item"`
}

// V3Feed : NvdV3Feed
// https://scap.nist.gov/schema/nvd/feed/0.1/nvd_cve_feed_json_0.1_beta.schema
type V3Feed struct {
	CVEItems []struct {
		Configurations struct {
			Nodes []struct {
				Cpe []V3FeedCpe `json:"cpe_match"`
			} `json:"nodes"`
		} `json:"configurations"`
	} `json:"CVE_Items"`
}

// V3FeedCpe :
type V3FeedCpe struct {
	Cpe23URI string `json:"cpe23Uri"`
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
	if cpes, err = convertNvdCpeDictionaryToModel(cpeDictionary, viper.GetInt("threads"), viper.GetInt("wait")); err != nil {
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
	urls := makeFeedURLBlocks(years)
	nvds, err := fetchNVDFeedFileConcurrently(urls, viper.GetInt("threads"), viper.GetInt("wait"))
	if err != nil {
		return nil, fmt.Errorf("Failed to get feeds. err : %s", err)
	}
	cpes, err := convertNvdV3FeedToModel(nvds, viper.GetInt("threads"), viper.GetInt("wait"))
	if err != nil {
		return nil, err
	}
	allCpes = append(allCpes, cpes...)
	return allCpes, nil
}

// makeFeedURLBlocks : makeFeedURLBlocks
func makeFeedURLBlocks(years []int) (urls []string) {
	formatTemplate := "https://nvd.nist.gov/feeds/json/cve/1.1/nvdcve-1.1-%d.json.gz"
	for _, year := range years {
		urls = append(urls, fmt.Sprintf(formatTemplate, year))
	}
	return urls
}

func fetchNVDFeedFileConcurrently(urls []string, concurrency, wait int) (nvds []V3Feed, err error) {
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

	tasks := util.GenWorkers(concurrency, wait)
	for range urls {
		tasks <- func() {
			select {
			case url := <-reqChan:
				nvd, err := fetchNVDFeedFile(url)
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

func fetchNVDFeedFile(url string) (nvd *V3Feed, err error) {
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
func convertNvdCpeDictionaryToModel(nvd CpeDictionary, concurrency, wait int) (cpes []models.CategorizedCpe, err error) {
	blockItems := [][]CpeDictionaryItem{}
	for i := 0; i < len(nvd.Items); i += concurrency {
		end := i + concurrency
		if len(nvd.Items) < end {
			end = len(nvd.Items)
		}
		blockItems = append(blockItems, nvd.Items[i:end])
	}

	reqChan := make(chan []CpeDictionaryItem, len(nvd.Items))
	resChan := make(chan []models.CategorizedCpe, len(nvd.Items))
	errChan := make(chan error)
	defer close(reqChan)
	defer close(resChan)
	defer close(errChan)

	go func() {
		for _, item := range blockItems {
			reqChan <- item
		}
	}()

	tasks := util.GenWorkers(concurrency, wait)
	for range blockItems {
		tasks <- func() {
			req := <-reqChan
			cpes, err := convertNvdCpeDictionary(req)
			if err != nil {
				errChan <- err
				return
			}
			resChan <- cpes
		}
	}

	errs := []error{}
	timeout := time.After(10 * 60 * time.Second)
	for range blockItems {
		select {
		case res := <-resChan:
			cpes = append(cpes, res...)
		case err := <-errChan:
			errs = append(errs, err)
		case <-timeout:
			return nil, fmt.Errorf("Timeout Converting")
		}
	}
	if 0 < len(errs) {
		return nil, fmt.Errorf("%s", errs)
	}
	return cpes, nil
}

func convertNvdCpeDictionary(items []CpeDictionaryItem) (cpes []models.CategorizedCpe, err error) {
	for _, item := range items {
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
func convertNvdV3FeedToModel(nvds []V3Feed, concurrency, wait int) (cpes []models.CategorizedCpe, err error) {
	allCpes := []V3FeedCpe{}
	for _, nvd := range nvds {
		for _, item := range nvd.CVEItems {
			for _, node := range item.Configurations.Nodes {
				allCpes = append(allCpes, node.Cpe...)
			}
		}
	}

	blockItems := [][]V3FeedCpe{}
	for i := 0; i < len(allCpes); i += concurrency {
		end := i + concurrency
		if len(allCpes) < end {
			end = len(allCpes)
		}
		blockItems = append(blockItems, allCpes[i:end])
	}

	reqChan := make(chan []V3FeedCpe, len(allCpes))
	resChan := make(chan []models.CategorizedCpe, len(allCpes))
	errChan := make(chan error)
	defer close(reqChan)
	defer close(resChan)
	defer close(errChan)

	go func() {
		for _, item := range blockItems {
			reqChan <- item
		}
	}()

	tasks := util.GenWorkers(concurrency, wait)
	for range blockItems {
		tasks <- func() {
			req := <-reqChan
			cpes, err := convertNvdV3Feed(req)
			if err != nil {
				errChan <- err
				return
			}
			resChan <- cpes
		}
	}

	errs := []error{}
	timeout := time.After(10 * 60 * time.Second)
	for range blockItems {
		select {
		case res := <-resChan:
			cpes = append(cpes, res...)
		case err := <-errChan:
			errs = append(errs, err)
		case <-timeout:
			return nil, fmt.Errorf("Timeout Converting")
		}
	}
	if 0 < len(errs) {
		return nil, fmt.Errorf("%s", errs)
	}
	return cpes, nil
}

// convertNvdV3Feed :
func convertNvdV3Feed(v3FeedCpes []V3FeedCpe) (cpes []models.CategorizedCpe, err error) {
	for _, cpe := range v3FeedCpes {
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
	return cpes, nil
}
