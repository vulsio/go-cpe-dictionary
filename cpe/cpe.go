package cpe

import (
	"bytes"
	"compress/gzip"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/parnurzeal/gorequest"
)

// CpeList has cpe-item list
// https://nvd.nist.gov/cpe.cfm
type CpeList struct {
	CpeItems []CpeItem `xml:"cpe-item"`
}

// CpeItem has CPE information
type CpeItem struct {
	Name      string    `xml:"name,attr"`
	Cpe23Item Cpe23Item `xml:"cpe23-item"`
	Titles    []Title   `xml:"title"`

	// each items
	//  Part     string
	//  Vendor   string
	//  Product  string
	//  Version  string
	//  Update   string
	//  Edition  string
	//  Language string
}

func (item CpeItem) GetTitleEn() string {
	for _, t := range item.Titles {
		if t.Lang == "en-US" {
			return t.Value
		}
	}
	return ""
}

func (item CpeItem) GetTitleJa() string {
	for _, t := range item.Titles {
		if t.Lang == "ja-JP" {
			return t.Value
		}
	}
	return ""
}

type Cpe23Item struct {
	Name string `xml:"name,attr"`
}

// Title has title, lang
type Title struct {
	Lang  string `xml:"lang,attr"`
	Value string `xml:",chardata"`
}

func FetchCPE(httpProxy string) (cpeList CpeList, err error) {
	var body string
	var errs []error
	var resp *http.Response
	url := "http://static.nvd.nist.gov/feeds/xml/cpe/dictionary/official-cpe-dictionary_v2.3.xml.gz"
	resp, body, errs = gorequest.New().Proxy(httpProxy).Get(url).End()
	if len(errs) > 0 || resp.StatusCode != 200 {
		return cpeList, fmt.Errorf("HTTP error. errs: %v, url: %s", errs, url)
	}

	b := bytes.NewBufferString(body)
	reader, err := gzip.NewReader(b)
	defer reader.Close()
	if err != nil {
		return cpeList,
			fmt.Errorf("Failed to decompress NVD feedfile. url: %s, err: %s", url, err)
	}
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return cpeList,
			fmt.Errorf("Failed to Read NVD feedfile. url: %s, err: %s", url, err)
	}
	if err = xml.Unmarshal(bytes, &cpeList); err != nil {
		return cpeList, fmt.Errorf("Failed to unmarshal. url: %s, err: %s", url, err)
	}

	return
}
