package nvd

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/inconshreveable/log15"
	"github.com/klauspost/compress/zstd"
	"github.com/knqyf263/go-cpe/common"
	"github.com/knqyf263/go-cpe/naming"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"golang.org/x/xerrors"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/registry/remote"

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
	log15.Info("Fetching vuls-data-raw-nvd-api-cpe ...")

	dir, err := os.MkdirTemp("", "go-cpe-dictionary")
	if err != nil {
		return xerrors.Errorf("Failed to create temp directory. err: %w", err)
	}
	defer os.RemoveAll(dir)

	if err := fetch(dir, "vuls-data-raw-nvd-api-cpe"); err != nil {
		return xerrors.Errorf("Failed to fetch vuls-data-raw-nvd-api-cpe. err: %w", err)
	}

	if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return xerrors.Errorf("Failed to open %s. err: %w", path, err)
		}
		defer f.Close()

		var item cpeDictionaryItem
		if err := json.NewDecoder(f).Decode(&item); err != nil {
			return xerrors.Errorf("Failed to decode %s. err: %w", path, err)
		}

		if _, err := naming.UnbindFS(item.Name); err != nil {
			log15.Warn("Failed to unbind", item.Name, err)
			return nil
		}

		var title string
		for _, t := range item.Titles {
			title = t.Title
			if t.Lang == "en" {
				break
			}
		}
		if item.Deprecated {
			deprecated[item.Name] = title
		} else {
			cpes[item.Name] = title
		}

		return nil
	}); err != nil {
		return xerrors.Errorf("Failed to walk %s. err: %w", dir, err)
	}

	return nil
}

func fetchCpeMatch(cpes map[string]string) error {
	log15.Info("Fetching vuls-data-raw-nvd-api-cpematch ...")

	dir, err := os.MkdirTemp("", "go-cpe-dictionary")
	if err != nil {
		return xerrors.Errorf("Failed to create temp directory. err: %w", err)
	}
	defer os.RemoveAll(dir)

	if err := fetch(dir, "vuls-data-raw-nvd-api-cpematch"); err != nil {
		return xerrors.Errorf("Failed to fetch vuls-data-raw-nvd-api-cpematch. err: %w", err)
	}

	if err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return xerrors.Errorf("Failed to open %s. err: %w", path, err)
		}
		defer f.Close()

		var cpeMatch cpeMatchElement
		if err := json.NewDecoder(f).Decode(&cpeMatch); err != nil {
			return xerrors.Errorf("Failed to decode %s. err: %w", path, err)
		}
		wfn, err := naming.UnbindFS(cpeMatch.Criteria)
		if err != nil {
			log15.Warn("Failed to unbind", cpeMatch.Criteria, err)
			return nil
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

		return nil
	}); err != nil {
		return xerrors.Errorf("Failed to walk %s. err: %w", dir, err)
	}

	return nil
}

func fetch(dir, tag string) error {
	ctx := context.TODO()
	repo, err := remote.NewRepository(fmt.Sprintf("ghcr.io/vulsio/vuls-data-db:%s", tag))
	if err != nil {
		return xerrors.Errorf("Failed to create client for %s. err: %w", fmt.Sprintf("ghcr.io/vulsio/vuls-data-db:%s", tag), err)
	}

	_, r, err := oras.Fetch(ctx, repo, repo.Reference.Reference, oras.DefaultFetchOptions)
	if err != nil {
		return xerrors.Errorf("Failed to fetch manifest. err: %w", err)
	}
	defer r.Close()

	var manifest ocispec.Manifest
	if err := json.NewDecoder(r).Decode(&manifest); err != nil {
		return xerrors.Errorf("Failed to decode manifest. err: %w", err)
	}

	l := func() *ocispec.Descriptor {
		for _, l := range manifest.Layers {
			if l.MediaType == "application/vnd.vulsio.vuls-data-db.dotgit.layer.v1.tar+zstd" {
				return &l
			}
		}
		return nil
	}()
	if l == nil {
		return xerrors.Errorf("Failed to find digest and filename from layers, actual layers: %#v", manifest.Layers)
	}

	r, err = repo.Fetch(ctx, *l)
	if err != nil {
		return xerrors.Errorf("Failed to fetch content. err: %w", err)
	}
	defer r.Close()

	zr, err := zstd.NewReader(r)
	if err != nil {
		return xerrors.Errorf("Failed to new zstd reader. err: %w", err)
	}
	defer zr.Close()

	tr := tar.NewReader(zr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return xerrors.Errorf("Failed to next tar reader. err: %w", err)
		}

		p := filepath.Join(dir, hdr.Name)

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(p, 0755); err != nil {
				return xerrors.Errorf("Failed to mkdir %s. err: %w", p, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
				return xerrors.Errorf("Failed to mkdir %s. err: %w", p, err)
			}

			if err := func() error {
				f, err := os.Create(p)
				if err != nil {
					return xerrors.Errorf("Failed to create %s. err: %w", p, err)
				}
				defer f.Close()

				if _, err := io.Copy(f, tr); err != nil {
					return xerrors.Errorf("Failed to copy to %s. err: %w", p, err)
				}

				return nil
			}(); err != nil {
				return xerrors.Errorf("Failed to create %s. err: %w", p, err)
			}
		}
	}

	cmd := exec.Command("git", "-C", filepath.Join(dir, tag), "restore", ".")
	if err := cmd.Run(); err != nil {
		return xerrors.Errorf("Failed to exec %q. err: %w", cmd.String(), err)
	}

	return nil
}
