package util

import (
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"time"

	logger "github.com/inconshreveable/log15"
	"golang.org/x/xerrors"
)

// GenWorkers generate workers
func GenWorkers(num, wait int) chan<- func() {
	tasks := make(chan func())
	for i := 0; i < num; i++ {
		go func() {
			for f := range tasks {
				f()
				time.Sleep(time.Duration(wait) * time.Second)
			}
		}()
	}
	return tasks
}

// GetDefaultLogDir returns default log directory
func GetDefaultLogDir() string {
	defaultLogDir := "/var/log/go-cpe-dictionary"
	if runtime.GOOS == "windows" {
		defaultLogDir = filepath.Join(os.Getenv("APPDATA"), "go-cpe-dictionary")
	}
	return defaultLogDir
}

// SetLogger set logger
func SetLogger(logToFile bool, logDir string, debug, logJSON bool) error {
	stderrHandler := logger.StderrHandler
	logFormat := logger.LogfmtFormat()
	if logJSON {
		logFormat = logger.JsonFormatEx(false, true)
		stderrHandler = logger.StreamHandler(os.Stderr, logFormat)
	}

	lvlHandler := logger.LvlFilterHandler(logger.LvlInfo, stderrHandler)
	if debug {
		lvlHandler = logger.LvlFilterHandler(logger.LvlDebug, stderrHandler)
	}

	var handler logger.Handler
	if logToFile {
		if _, err := os.Stat(logDir); err != nil {
			if os.IsNotExist(err) {
				if err := os.Mkdir(logDir, 0700); err != nil {
					return xerrors.Errorf("Failed to create log directory. err: %w", err)
				}
			} else {
				return xerrors.Errorf("Failed to check log directory. err: %w", err)
			}
		}

		logPath := filepath.Join(logDir, "go-cpe-dictionary.log")
		if _, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err != nil {
			return xerrors.Errorf("Failed to open a log file. err: %w", err)
		}
		handler = logger.MultiHandler(
			logger.Must.FileHandler(logPath, logFormat),
			lvlHandler,
		)
	} else {
		handler = lvlHandler
	}
	logger.Root().SetHandler(handler)
	return nil
}

// Unique return unique elements
func Unique[T comparable](s []T) []T {
	m := map[T]struct{}{}
	for _, v := range s {
		m[v] = struct{}{}
	}
	return slices.Collect(maps.Keys(m))
}
