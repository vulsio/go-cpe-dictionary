package fetcher

import (
	"encoding/json"

	"github.com/vulsio/go-cpe-dictionary/models"
	"github.com/vulsio/go-cpe-dictionary/util"
	"golang.org/x/xerrors"
)

// FetchVuls Vuls Annotation feeds
func FetchVuls() (models.FetchedCPEs, error) {
	bs, err := util.FetchFeedFile("https://raw.githubusercontent.com/vulsio/go-cpe-dictionary/master/annotation/vuls.json", false)
	if err != nil {
		return models.FetchedCPEs{}, xerrors.Errorf("Failed to fetch. url: %s, err: %w", "https://raw.githubusercontent.com/vulsio/go-cpe-dictionary/master/annotation/vuls.json", err)
	}

	var cs []models.FetchedCPE
	if err := json.Unmarshal(bs, &cs); err != nil {
		return models.FetchedCPEs{}, xerrors.Errorf("Failed to unmarshal json. err: %w", err)
	}

	m := map[string][]string{}
	for _, c := range cs {
		m[c.Title] = append(m[c.Title], c.CPEs...)
	}

	var cpes models.FetchedCPEs
	for t, cs := range m {
		cpes.CPEs = append(cpes.CPEs, models.FetchedCPE{
			Title: t,
			CPEs:  util.Unique(cs),
		})
	}

	return cpes, nil
}
