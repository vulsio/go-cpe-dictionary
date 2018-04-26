package util

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/inconshreveable/log15"
)

// GenWorkers generate workders
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
func SetLogger(logDir string, debug, logJSON bool) {
	stderrHundler := log15.StderrHandler
	logFormat := log15.LogfmtFormat()
	if logJSON {
		logFormat = log15.JsonFormatEx(false, true)
		stderrHundler = log15.StreamHandler(os.Stderr, logFormat)
	}

	lvlHundler := log15.LvlFilterHandler(log15.LvlInfo, stderrHundler)
	if debug {
		lvlHundler = log15.LvlFilterHandler(log15.LvlDebug, stderrHundler)
	}

	if _, err := os.Stat(logDir); os.IsNotExist(err) {
		if err := os.Mkdir(logDir, 0700); err != nil {
			log15.Error("Failed to create log directory", "err", err)
		}
	}
	var hundler log15.Handler
	if _, err := os.Stat(logDir); err == nil {
		logPath := filepath.Join(logDir, "go-cpe-dictionary.log")
		hundler = log15.MultiHandler(
			log15.Must.FileHandler(logPath, logFormat),
			lvlHundler,
		)
	} else {
		hundler = lvlHundler
	}
	log15.Root().SetHandler(hundler)
}
