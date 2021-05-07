package util

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"

	"github.com/inconshreveable/log15"
	logger "github.com/inconshreveable/log15"
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
