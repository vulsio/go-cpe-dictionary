package fetcher

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"

	"github.com/inconshreveable/log15"
	"github.com/knqyf263/go-cpe/common"
	"github.com/knqyf263/go-cpe/naming"
	"github.com/parnurzeal/gorequest"
	"github.com/spf13/viper"
	"golang.org/x/xerrors"

	"github.com/vulsio/go-cpe-dictionary/models"
)

// cpeDictionary : https://csrc.nist.gov/schema/nvd/api/2.0/cpe_api_json_2.0.schema
type cpeDictionaryItem struct {
	Deprecated bool   `json:"deprecated"`
	Name       string `json:"cpeName"`
	Titles     []struct {
		Title string `json:"title"`
		Lang  string `json:"lang"`
	} `json:"titles,omitempty"`
}

// cpeMatch : https://csrc.nist.gov/schema/nvd/api/2.0/cpematch_api_json_2.0.schema
type cpeMatchElement struct {
	Criteria string `json:"criteria"`
}

// FetchNVD NVD feeds
func FetchNVD() (models.FetchedCPEs, error) {
	cpes := map[string]string{}
	deprecateds := map[string]string{}
	if err := fetchCpeDictionary(cpes, deprecateds); err != nil {
		return models.FetchedCPEs{}, xerrors.Errorf("Failed to fetch cpe dictionary. err: %w", err)
	}
	if err := fetchCpeMatch(cpes); err != nil {
		return models.FetchedCPEs{}, xerrors.Errorf("Failed to fetch cpe match. err: %w", err)
	}
	var fetched models.FetchedCPEs
	for c, t := range cpes {
		fetched.CPEs = append(fetched.CPEs, models.FetchedCPE{
			Title: t,
			CPEs:  []string{c},
		})
	}
	for c, t := range deprecateds {
		fetched.Deprecated = append(fetched.Deprecated, models.FetchedCPE{
			Title: t,
			CPEs:  []string{c},
		})
	}
	return fetched, nil
}

func fetchCpeDictionary(cpes, deprecated map[string]string) error {
	url := "https://github.com/vulsio/vuls-data-raw-nvd-api-cpe/archive/refs/heads/main.tar.gz"
	log15.Info("Fetching...", "URL", url)
	resp, bs, errs := gorequest.New().Proxy(viper.GetString("http-proxy")).Get(url).EndBytes()
	if len(errs) > 0 || resp == nil || resp.StatusCode != 200 {
		return xerrors.Errorf("HTTP error. errs: %v, url: %s", errs, url)
	}

	gr, err := gzip.NewReader(bytes.NewReader(bs))
	if err != nil {
		return xerrors.Errorf("Failed to create gzip reader. url: %s, err: %w", url, err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return xerrors.Errorf("Failed to next tar reader. err: %w", err)
		}

		if hdr.FileInfo().IsDir() {
			continue
		}

		if filepath.Ext(hdr.Name) != ".json" {
			continue
		}

		var item cpeDictionaryItem
		if err := json.NewDecoder(tr).Decode(&item); err != nil {
			return xerrors.Errorf("Failed to decode %s. err: %w", hdr.Name, err)
		}

		if _, err := naming.UnbindFS(item.Name); err != nil {
			log15.Warn("Failed to unbind", item.Name, err)
			continue
		}
		for _, t := range item.Titles {
			if item.Deprecated {
				deprecated[item.Name] = t.Title
			} else {
				cpes[item.Name] = t.Title
			}
		}
	}

	return nil
}

func fetchCpeMatch(cpes map[string]string) error {
	url := "https://github.com/vulsio/vuls-data-raw-nvd-api-cpematch/archive/refs/heads/main.tar.gz"
	log15.Info("Fetching...", "URL", url)
	resp, bs, errs := gorequest.New().Proxy(viper.GetString("http-proxy")).Get(url).EndBytes()
	if len(errs) > 0 || resp == nil || resp.StatusCode != 200 {
		return xerrors.Errorf("HTTP error. errs: %v, url: %s", errs, url)
	}

	gr, err := gzip.NewReader(bytes.NewReader(bs))
	if err != nil {
		return xerrors.Errorf("Failed to create gzip reader. url: %s, err: %w", url, err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return xerrors.Errorf("Failed to next tar reader. err: %w", err)
		}

		if hdr.FileInfo().IsDir() {
			continue
		}

		if filepath.Ext(hdr.Name) != ".json" {
			continue
		}

		var cpeMatch cpeMatchElement
		if err := json.NewDecoder(tr).Decode(&cpeMatch); err != nil {
			return xerrors.Errorf("Failed to decode %s. err: %w", hdr.Name, err)
		}
		wfn, err := naming.UnbindFS(cpeMatch.Criteria)
		if err != nil {
			log15.Warn("Failed to unbind", cpeMatch.Criteria, err)
			continue
		}
		if _, ok := cpes[cpeMatch.Criteria]; !ok {
			title := fmt.Sprintf("%s %s", wfn.GetString(common.AttributeVendor), wfn.GetString(common.AttributeProduct))
			if wfn.GetString(common.AttributeVersion) != "ANY" {
				title = fmt.Sprintf("%s %s", title, wfn.GetString(common.AttributeVersion))
			}
			if wfn.GetString(common.AttributeUpdate) != "ANY" {
				title = fmt.Sprintf("%s %s", title, wfn.GetString(common.AttributeUpdate))
			}
			if wfn.GetString(common.AttributeEdition) != "ANY" {
				title = fmt.Sprintf("%s %s", title, wfn.GetString(common.AttributeEdition))
			}
			if wfn.GetString(common.AttributeLanguage) != "ANY" {
				title = fmt.Sprintf("%s %s", title, wfn.GetString(common.AttributeLanguage))
			}
			if wfn.GetString(common.AttributeSwEdition) != "ANY" {
				title = fmt.Sprintf("%s %s", title, wfn.GetString(common.AttributeSwEdition))
			}
			if wfn.GetString(common.AttributeTargetSw) != "ANY" {
				title = fmt.Sprintf("%s %s", title, wfn.GetString(common.AttributeTargetSw))
			}
			if wfn.GetString(common.AttributeTargetHw) != "ANY" {
				title = fmt.Sprintf("%s %s", title, wfn.GetString(common.AttributeTargetHw))
			}
			if wfn.GetString(common.AttributeOther) != "ANY" {
				title = fmt.Sprintf("%s %s", title, wfn.GetString(common.AttributeOther))
			}
			cpes[cpeMatch.Criteria] = title
		}
	}

	return nil
}
