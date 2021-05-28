package util

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/inconshreveable/log15"
	logger "github.com/inconshreveable/log15"
	"github.com/kotakanbe/go-cpe-dictionary/config"
	"github.com/parnurzeal/gorequest"
)

// GenWorkers generate workers
func GenWorkers(num int) chan<- func() {
	tasks := make(chan func())
	for i := 0; i < num; i++ {
		go func() {
			for f := range tasks {
				f()
			}
		}()
	}
	return tasks
}

// GetDefaultLogDir returns default log directory
func GetDefaultLogDir() string {
	defaultLogDir := "/var/log/vuls"
	if runtime.GOOS == "windows" {
		defaultLogDir = filepath.Join(os.Getenv("APPDATA"), "vuls")
	}
	return defaultLogDir
}

// SetLogger set logger
func SetLogger(logDir string, debug, logJSON, logToFile bool) {
	stderrHandler := log15.StderrHandler
	logFormat := log15.LogfmtFormat()
	if logJSON {
		logFormat = log15.JsonFormatEx(false, true)
		stderrHandler = log15.StreamHandler(os.Stderr, logFormat)
	}

	lvlHandler := log15.LvlFilterHandler(log15.LvlInfo, stderrHandler)
	if debug {
		lvlHandler = log15.LvlFilterHandler(log15.LvlDebug, stderrHandler)
	}

	var handler logger.Handler
	if logToFile {
		if _, err := os.Stat(logDir); os.IsNotExist(err) {
			if err := os.Mkdir(logDir, 0700); err != nil {
				logger.Error("Failed to create a log directory", "err", err)
			}
		}
		if _, err := os.Stat(logDir); err == nil {
			logPath := filepath.Join(logDir, "cve-dictionary.log")
			if err := ioutil.WriteFile(logPath, []byte{}, 0700); err != nil {
				logger.Error("Failed to create a log file", "err", err)
				handler = lvlHandler
			} else {
				handler = logger.MultiHandler(
					logger.Must.FileHandler(logPath, logFormat),
					lvlHandler,
				)
			}

		}
	} else {
		handler = lvlHandler
	}
	logger.Root().SetHandler(handler)
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

func FetchFeedFile(url string, compressed bool) ([]byte, error) {
	var body string
	var errs []error
	var resp *http.Response
	f := func() (err error) {
		log15.Info("Fetching...", "URL", url)
		resp, body, errs = gorequest.New().Timeout(60 * time.Second).Proxy(config.Conf.HTTPProxy).Get(url).End()
		defer func() {
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
			}
		}()
		if len(errs) > 0 || resp == nil || resp.StatusCode != 200 {
			return fmt.Errorf("HTTP error. errs: %v, url: %s", errs, url)
		}
		return nil
	}
	notify := func(err error, t time.Duration) {
		logger.Warn("Failed to HTTP GET", "retrying in", t)
	}
	err := backoff.RetryNotify(f, backoff.NewExponentialBackOff(), notify)
	if err != nil {
		return nil, err
	}

	b := bytes.NewBufferString(body)
	if !compressed {
		return b.Bytes(), nil
	}

	reader, err := gzip.NewReader(bytes.NewReader(b.Bytes()))
	defer func() {
		if reader != nil {
			_ = reader.Close()
		}
	}()
	if err != nil {
		return nil, fmt.Errorf("Failed to decompress NVD feedfile. url: %s, err: %s", url, err)
	}
	bytes, err := ioutil.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("Failed to Read NVD feedfile. url: %s, err: %s", url, err)
	}
	return bytes, nil
}
