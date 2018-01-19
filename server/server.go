package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/kotakanbe/go-cpe-dictionary/config"
	"github.com/kotakanbe/go-cpe-dictionary/db"
	"github.com/kotakanbe/go-cpe-dictionary/models"
	log "github.com/sirupsen/logrus"
)

// Start starts CVE dictionary HTTP Server.
func Start(logDir string, driver db.DB) error {
	e := echo.New()
	e.Debug = config.Conf.Debug

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// setup access logger
	logPath := filepath.Join(logDir, "access.log")
	if _, err := os.Stat(logPath); os.IsNotExist(err) {
		if _, err := os.Create(logPath); err != nil {
			return err
		}
	}
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{
		Output: f,
	}))

	// Routes
	e.GET("/health", health())
	e.GET("/cpe/:cpename", getCpe(driver))

	bindURL := fmt.Sprintf("%s:%s", config.Conf.Bind, config.Conf.Port)
	log.Infof("Listening on %s", bindURL)

	e.Start(bindURL)
	return nil
}

// Handler
func health() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.String(http.StatusOK, "")
	}
}

// Handler
func getCpe(driver db.DB) echo.HandlerFunc {
	var err error
	return func(c echo.Context) error {
		cpeName := c.Param("cpename")
		var cpe models.CategorizedCpe
		if cpe, err = driver.GetCpeFromCpe23(cpeName); err != nil {
			return c.JSON(http.StatusBadRequest, nil)
		}
		return c.JSON(http.StatusOK, &cpe)
	}
}
