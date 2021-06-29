package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/inconshreveable/log15"
	"github.com/kotakanbe/go-cpe-dictionary/config"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
)

// Start starts CVE dictionary HTTP Server.
func Start(logDir string) error {
	e := echo.New()
	e.Debug = config.Conf.Debug

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// setup access logger
	logPath := filepath.Join(logDir, "access.log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		if _, err := os.Create(logPath); err != nil {
			log15.Error("Failed to create log dir", logPath, err)
		}
	}
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		log15.Error("Failed to open log file", logPath, err)
	}
	defer f.Close()
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Output: f,
	}))

	// Routes
	e.GET("/health", health())

	bindURL := fmt.Sprintf("%s:%s", config.Conf.Bind, config.Conf.Port)
	log15.Info("Listening...", "URL", bindURL)
	return e.Start(bindURL)
}

// Handler
func health() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.String(http.StatusOK, "")
	}
}
