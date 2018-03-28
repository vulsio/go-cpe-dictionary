package nvd

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/cheggaaa/pb"
	"github.com/knqyf263/go-cpe/common"
	"github.com/knqyf263/go-cpe/naming"
	c "github.com/kotakanbe/go-cpe-dictionary/config"
	"github.com/kotakanbe/go-cpe-dictionary/db"
	"github.com/kotakanbe/go-cpe-dictionary/models"
	"github.com/kotakanbe/go-cpe-dictionary/util"
	"github.com/labstack/gommon/log"
	"github.com/parnurzeal/gorequest"
	"github.com/pkg/errors"
)

// CpeDictionary has cpe-item list
// https://nvd.nist.gov/cpe.cfm
type CpeDictionary struct {
	Items []struct {
		Name      string `xml:"name,attr"`
		Cpe23Item struct {
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
					Cpe22URI string `json:"cpe22Uri"`
					Cpe23URI string `json:"cpe23Uri"`
				} `json:"cpe"`
			} `json:"nodes"`
		} `json:"configurations"`
	} `json:"CVE_Items"`
}

// FetchAndInsertCPE : FetchAndInsertCPE
func FetchAndInsertCPE(driver db.DB) (err error) {
	if err = FetchAndInsertCpeDictioanry(driver); err != nil {
		return fmt.Errorf("Failed to fetch cpe dictionary. err : %s", err)
	}

	if err = FetchAndInsertV3Feed(driver); err != nil {
		return fmt.Errorf("Failed to fetch nvd v3 feed. err : %s", err)
	}

	return nil
}

// FetchAndInsertCpeDictioanry : FetchCPE
func FetchAndInsertCpeDictioanry(driver db.DB) (err error) {
	var cpeDictionary CpeDictionary
	var body string
	var errs []error
	var resp *http.Response
	url := "http://static.nvd.nist.gov/feeds/xml/cpe/dictionary/official-cpe-dictionary_v2.3.xml.gz"
	resp, body, errs = gorequest.New().Proxy(c.Conf.HTTPProxy).Get(url).End()
	if len(errs) > 0 || resp.StatusCode != 200 {
		return fmt.Errorf("HTTP error. errs: %v, url: %s", errs, url)
	}

	b := bytes.NewBufferString(body)
	reader, err := gzip.NewReader(b)
	defer reader.Close()
	if err != nil {
		return fmt.Errorf("Failed to decompress NVD feedfile. url: %s, err: %s", url, err)
	}
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("Failed to Read NVD feedfile. url: %s, err: %s", url, err)
	}
	if err = xml.Unmarshal(bytes, &cpeDictionary); err != nil {
		return fmt.Errorf("Failed to unmarshal. url: %s, err: %s", url, err)
	}

	var cpes []*models.CategorizedCpe
	if cpes, err = ConvertNvdCpeDictionaryToModel(cpeDictionary); err != nil {
		return err
	}

	if err = driver.InsertCpes(cpes); err != nil {
		return fmt.Errorf("Failed to insert cpes. err : %s", err)
	}
	return nil
}

// FetchAndInsertV3Feed : FetchAndInsertV3Feed
func FetchAndInsertV3Feed(driver db.DB) (err error) {
	startYear := 2002
	var years []int
	if years, err = GetYearsUntilThisYear(startYear); err != nil {
		return err
	}

	urlBlocks := MakeFeedURLBlocks(years, 3)
	for _, urls := range urlBlocks {
		var nvds []V3Feed
		if nvds, err = fetchFeedFileConcurrently(urls); err != nil {
			return fmt.Errorf("Failed to get feeds. err : %s", err)
		}
		var cpes []*models.CategorizedCpe
		if cpes, err = ConvertNvdV3FeedToModel(nvds); err != nil {
			return err
		}
		if err = driver.InsertCpes(cpes); err != nil {
			return fmt.Errorf("Failed to insert cpes. err : %s", err)
		}
	}
	return nil
}

// GetYearsUntilThisYear : GetYearsUntilThisYear
func GetYearsUntilThisYear(startYear int) (years []int, err error) {
	var thisYear int
	if thisYear, err = strconv.Atoi(time.Now().Format("2006")); err != nil {
		return years, fmt.Errorf("Failed to convert this year. err : %s", err)
	}
	years = make([]int, thisYear-startYear+1)
	for i := range years {
		years[i] = startYear + i
	}
	return years, nil
}

// MakeFeedURLBlocks : MakeFeedURLBlocks
func MakeFeedURLBlocks(years []int, n int) (urlBlocks [][]string) {
	//  http://static.nvd.nist.gov/feeds/xml/cve/nvdcve-2.0-2016.xml.gz
	formatTemplate := "https://static.nvd.nist.gov/feeds/json/cve/1.0/nvdcve-1.0-%d.json.gz"
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
				log.Infof("Fetching... %s", url)
				nvd, err := fetchFeedFile(url)
				if err != nil {
					errChan <- err
					return
				}
				resChan <- nvd
			}
		}
	}

	errs := []error{}
	bar := pb.New(len(urls))
	bar.Start()
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
		bar.Increment()
	}
	bar.Finish()
	//  bar.FinishPrint("Finished to fetch CVE information from JVN.")
	if 0 < len(errs) {
		return nvds, fmt.Errorf("%s", errs)
	}
	return nvds, nil
}

func fetchFeedFile(url string) (nvd V3Feed, err error) {
	var body string
	var errs []error
	var resp *http.Response

	resp, body, errs = gorequest.New().Proxy(c.Conf.HTTPProxy).Get(url).End()
	//  defer resp.Body.Close()
	if len(errs) > 0 || resp == nil || resp.StatusCode != 200 {
		return nvd, fmt.Errorf(
			"HTTP error. errs: %v, url: %s", errs, url)
	}

	b := bytes.NewBufferString(body)
	reader, err := gzip.NewReader(b)
	defer reader.Close()
	if err != nil {
		return nvd, fmt.Errorf(
			"Failed to decompress NVD feedfile. url: %s, err: %s", url, err)
	}

	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nvd, fmt.Errorf(
			"Failed to Read NVD feedfile. url: %s, err: %s", url, err)
	}

	if err = json.Unmarshal(bytes, &nvd); err != nil {
		return nvd, fmt.Errorf(
			"Failed to unmarshal. url: %s, err: %s", url, err)
	}
	return nvd, nil
}

// ConvertNvdCpeDictionaryToModel :
func ConvertNvdCpeDictionaryToModel(nvd CpeDictionary) (cpes []*models.CategorizedCpe, err error) {
	for _, item := range nvd.Items {
		var wfn common.WellFormedName
		if wfn, err = naming.UnbindFS(item.Cpe23Item.Name); err != nil {
			return nil, errors.Wrapf(err, "Failed to unbind cpe fs: %s", item.Cpe23Item.Name)
		}
		cpes = append(cpes, &models.CategorizedCpe{
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

// ConvertNvdV3FeedToModel :
func ConvertNvdV3FeedToModel(nvds []V3Feed) (cpes []*models.CategorizedCpe, err error) {
	for _, nvd := range nvds {
		for _, item := range nvd.CVEItems {
			for _, node := range item.Configurations.Nodes {
				for _, cpe := range node.Cpe {
					var wfn common.WellFormedName
					if wfn, err = naming.UnbindFS(cpe.Cpe23URI); err != nil {
						log.Warnf("Failed to unbind cpe fs: %s, err: %s", cpe.Cpe23URI, err)
						continue
					}
					cpes = append(cpes, &models.CategorizedCpe{
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
