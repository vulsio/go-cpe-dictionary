package fetcher

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"

	"github.com/inconshreveable/log15"
	"github.com/knqyf263/go-cpe/naming"
	"github.com/parnurzeal/gorequest"
	"github.com/spf13/viper"
	"golang.org/x/exp/maps"
	"golang.org/x/xerrors"

	"github.com/vulsio/go-cpe-dictionary/models"
)

// cpeDictionary has cpe-item list
// https://nvd.nist.gov/cpe.cfm
type cpeDictionaryItem struct {
	Name       string `xml:"name,attr"`
	Deprecated string `xml:"deprecated,attr"`
	Cpe23Item  struct {
		Name string `xml:"name,attr"`
	} `xml:"cpe23-item"`
}

// cpeMatch : https://csrc.nist.gov/schema/cpematch/feed/1.0/nvd_cpematch_feed_json_1.0.schema
type cpeMatchElement struct {
	Cpe23URI string `json:"cpe23Uri"`
}

// FetchNVD NVD feeds
func FetchNVD() (models.FetchedCPEs, error) {
	cpes := map[string]struct{}{}
	deprecateds := map[string]struct{}{}
	if err := fetchCpeDictionary(cpes, deprecateds); err != nil {
		return models.FetchedCPEs{}, xerrors.Errorf("Failed to fetch cpe dictionary. err: %w", err)
	}
	if err := fetchCpeMatch(cpes); err != nil {
		return models.FetchedCPEs{}, xerrors.Errorf("Failed to fetch cpe match. err: %w", err)
	}
	return models.FetchedCPEs{CPEs: maps.Keys(cpes), Deprecated: maps.Keys(deprecateds)}, nil
}

func fetchCpeDictionary(cpes, deprecated map[string]struct{}) error {
	url := "http://nvd.nist.gov/feeds/xml/cpe/dictionary/official-cpe-dictionary_v2.3.xml.gz"
	log15.Info("Fetching...", "URL", url)
	resp, bs, errs := gorequest.New().Proxy(viper.GetString("http-proxy")).Get(url).EndBytes()
	if len(errs) > 0 || resp == nil || resp.StatusCode != 200 {
		return xerrors.Errorf("HTTP error. errs: %v, url: %s", errs, url)
	}

	r, err := gzip.NewReader(bytes.NewReader(bs))
	if err != nil {
		return xerrors.Errorf("Failed to decompress CPE Dictionary. url: %s, err: %w", url, err)
	}
	defer r.Close()

	d := xml.NewDecoder(r)
	for {
		t, err := d.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return xerrors.Errorf("Failed to return next XML token. err: %w", err)
		}
		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local != "cpe-item" {
				break
			}
			var item cpeDictionaryItem
			if err := d.DecodeElement(&item, &se); err != nil {
				return xerrors.Errorf("Failed to decode. url: %s, err: %w", url, err)
			}
			if _, err := naming.UnbindFS(item.Cpe23Item.Name); err != nil {
				// Logging only
				log15.Warn("Failed to unbind", item.Cpe23Item.Name, err)
				continue
			}
			if item.Deprecated == "true" {
				deprecated[item.Cpe23Item.Name] = struct{}{}
			} else {
				cpes[item.Cpe23Item.Name] = struct{}{}
			}
		default:
		}
	}
	return nil
}

func fetchCpeMatch(cpes map[string]struct{}) error {
	url := "https://nvd.nist.gov/feeds/json/cpematch/1.0/nvdcpematch-1.0.json.gz"
	log15.Info("Fetching...", "URL", url)
	resp, bs, errs := gorequest.New().Proxy(viper.GetString("http-proxy")).Get(url).EndBytes()
	if len(errs) > 0 || resp == nil || resp.StatusCode != 200 {
		return xerrors.Errorf("HTTP error. errs: %v, url: %s", errs, url)
	}

	r, err := gzip.NewReader(bytes.NewReader(bs))
	if err != nil {
		return xerrors.Errorf("Failed to decompress CPE Match. url: %s, err: %w", url, err)
	}
	defer r.Close()

	d := json.NewDecoder(r)
	if _, err := d.Token(); err != nil { // json.Delim: {
		return xerrors.Errorf("Failed to return next JSON token. err: %w", err)
	}
	if _, err := d.Token(); err != nil { // string: matches
		return xerrors.Errorf("Failed to return next JSON token. err: %w", err)
	}
	if _, err := d.Token(); err != nil { // json.Delim: [
		return xerrors.Errorf("Failed to return next JSON token. err: %w", err)
	}
	for d.More() {
		var cpeMatch cpeMatchElement
		if err := d.Decode(&cpeMatch); err != nil {
			return xerrors.Errorf("Failed to decode. url: %s, err: %w", url, err)
		}
		if _, err := naming.UnbindFS(cpeMatch.Cpe23URI); err != nil {
			// Logging only
			log15.Warn("Failed to unbind", cpeMatch.Cpe23URI, err)
			continue
		}
		cpes[cpeMatch.Cpe23URI] = struct{}{}
	}
	if _, err := d.Token(); err != nil { // json.Delim: ]
		return xerrors.Errorf("Failed to return next JSON token. err: %w", err)
	}
	if _, err := d.Token(); err != nil { // json.Delim: }
		return xerrors.Errorf("Failed to return next JSON token. err: %w", err)
	}

	return nil
}
