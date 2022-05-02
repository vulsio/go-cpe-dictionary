package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/inconshreveable/log15"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/spf13/viper"
	"github.com/vulsio/go-cpe-dictionary/db"
	"github.com/vulsio/go-cpe-dictionary/models"
	"golang.org/x/xerrors"
)

// Start starts CVE dictionary HTTP Server.
func Start(logToFile bool, logDir string, driver db.DB) error {
	e := echo.New()
	e.Debug = viper.GetBool("debug")

	// Middleware
	e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{Output: os.Stderr}))
	e.Use(middleware.Recover())

	// setup access logger
	if logToFile {
		logPath := filepath.Join(logDir, "access.log")
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return xerrors.Errorf("Failed to open a log file: %s", err)
		}
		defer f.Close()
		e.Use(middleware.LoggerWithConfig(middleware.LoggerConfig{Output: f}))
	}

	// Routes
	e.GET("/health", health())
	e.GET("/products", getVendorProducts(driver))
	e.GET("/cpes/:vendor/:product", getCpesByVendorProduct(driver))

	bindURL := fmt.Sprintf("%s:%s", viper.GetString("bind"), viper.GetString("port"))
	log15.Info("Listening...", "URL", bindURL)
	return e.Start(bindURL)
}

// Handler
func health() echo.HandlerFunc {
	return func(c echo.Context) error {
		return c.String(http.StatusOK, "")
	}
}

// Handler
func getVendorProducts(driver db.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		products, deprecated, err := driver.GetVendorProducts()
		if err != nil {
			log15.Error("Failed to GetVendorProducts", "err", err)
			return c.JSON(http.StatusInternalServerError, []string{})
		}

		return c.JSON(http.StatusOK, map[string][]models.VendorProduct{"vendorProducts": products, "deprecated": deprecated})
	}
}

// Handler
func getCpesByVendorProduct(driver db.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		vendor := c.Param("vendor")
		product := c.Param("product")
		log15.Debug("Params", "vendor", vendor, "product", product)

		cpeURIs, deprecated, err := driver.GetCpesByVendorProduct(vendor, product)
		if err != nil {
			log15.Error("Failed to GetVendorProducts", "err", err)
			return c.JSON(http.StatusInternalServerError, map[string][]string{"cpeURIs": {}, "deprecated": {}})
		}

		return c.JSON(http.StatusOK, map[string][]string{"cpeURIs": cpeURIs, "deprecated": deprecated})
	}
}
