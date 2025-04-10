package util

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/inconshreveable/log15"
	"github.com/spf13/viper"
	"golang.org/x/xerrors"
)

// FetchFeedFile fetches vulnerability feed file
func FetchFeedFile(urlstr string, gzip bool) (body []byte, err error) {
	log15.Info("Fetching...", "URL", urlstr)

	count, retryMax := 0, 20
	f := func() (err error) {
		if body, err = fetchFile(urlstr, gzip); err != nil {
			count++
			if count == retryMax {
				return nil
			}
			return xerrors.Errorf("HTTP GET error, url: %s, err: %w", urlstr, err)
		}
		return nil
	}
	notify := func(err error, t time.Duration) {
		log15.Warn(fmt.Sprintf("Failed to HTTP GET. retrying in %s seconds.", t), "err", err)
	}
	err = backoff.RetryNotify(f, backoff.NewExponentialBackOff(), notify)
	if err != nil {
		return nil, xerrors.Errorf("Failed to fetch file: %w", err)
	}

	if count == retryMax {
		return nil, xerrors.Errorf("Failed to fetch file. Retry count exceeded: %d", retryMax)
	}

	return body, nil
}

func fetchFile(urlstr string, isGzip bool) (body []byte, err error) {
	var proxyURL *url.URL
	httpClient := &http.Client{
		Timeout: time.Duration(180 * time.Second),
	}
	if viper.GetString("http-proxy") != "" {
		if proxyURL, err = url.Parse(viper.GetString("http-proxy")); err != nil {
			return nil, xerrors.Errorf("Failed to parse proxy url: %w", err)
		}
		httpClient = &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
			Timeout: time.Duration(180 * time.Second),
		}
	}
	req, err := http.NewRequest("GET", urlstr, nil)
	if err != nil {
		return nil, xerrors.Errorf("Failed to new request. url: %s, err: %w", urlstr, err)
	}
	req.Header.Set("User-Agent", "curl/7.58.0")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, xerrors.Errorf("Failed to GET. url: %s, err: %w", urlstr, err)
	}

	defer resp.Body.Close()
	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, xerrors.Errorf("Failed to read body. url: %s, err: %w", urlstr, err)
	}

	if isGzip {
		reader, err := gzip.NewReader(bytes.NewReader(buf))
		if err != nil {
			return nil, xerrors.Errorf("Failed to decompress NVD feedfile. url: %s, err: %w", urlstr, err)
		}
		defer reader.Close()

		bytes, err := io.ReadAll(reader)
		if err != nil {
			return nil, xerrors.Errorf("Failed to Read NVD feedfile. url: %s, err: %w", urlstr, err)
		}
		return bytes, nil
	}

	return buf, nil
}
