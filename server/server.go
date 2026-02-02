package server

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/hbollon/go-edlib"
	"github.com/inconshreveable/log15"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"
	"golang.org/x/xerrors"

	"github.com/vulsio/go-cpe-dictionary/db"
	"github.com/vulsio/go-cpe-dictionary/models"
)

// requestLogger creates a request logger middleware with the given output writer
func requestLogger(output io.Writer) echo.MiddlewareFunc {
	return middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogError:    true,
		LogMethod:   true,
		LogLatency:  true,
		LogRemoteIP: true,
		LogValuesFunc: func(_ echo.Context, v middleware.RequestLoggerValues) error {
			_, _ = fmt.Fprintf(output, "%s %s %d %v\n", v.Method, v.URI, v.Status, v.Latency)
			return nil
		},
	})
}

// Start starts CVE dictionary HTTP Server.
func Start(logToFile bool, logDir string, driver db.DB) error {
	e := echo.New()
	e.Debug = viper.GetBool("debug")

	// Middleware
	e.Use(requestLogger(os.Stderr))
	e.Use(middleware.Recover())

	// setup access logger
	if logToFile {
		logPath := filepath.Join(logDir, "access.log")
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return xerrors.Errorf("Failed to open a log file: %s", err)
		}
		defer f.Close()
		e.Use(requestLogger(f))
	}

	// Routes
	e.GET("/health", health())
	e.GET("/products", getVendorProducts(driver))
	e.GET("/cpes/:vendor/:product", getCpesByVendorProduct(driver))
	e.POST("/cpes/fuzzysearch", getSimilarCpesByTitle(driver))

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

// Handler
func getSimilarCpesByTitle(driver db.DB) echo.HandlerFunc {
	return func(c echo.Context) error {
		var d struct {
			Query     string `json:"query"`
			N         int    `json:"n"`
			Algorithm string `json:"algorithm,omitempty"`
		}
		if err := c.Bind(&d); err != nil {
			return err
		}

		var algo edlib.Algorithm
		switch d.Algorithm {
		case "":
			algo = edlib.Jaro
		case "Levenshtein":
			algo = edlib.Levenshtein
		case "DamerauLevenshtein":
			algo = edlib.DamerauLevenshtein
		case "OSADamerauLevenshtein":
			algo = edlib.OSADamerauLevenshtein
		case "Lcs":
			algo = edlib.Lcs
		case "Jaro":
			algo = edlib.Jaro
		case "JaroWinkler":
			algo = edlib.JaroWinkler
		case "Cosine":
			algo = edlib.Cosine
		case "Jaccard":
			algo = edlib.Jaccard
		case "SorensenDice":
			algo = edlib.SorensenDice
		case "Qgram":
			algo = edlib.Qgram
		default:
			log15.Error("Failed to GetSimilarCpesByTitle", "err", "invalid algorithm parameter", "accepts", []string{"", "Levenshtein", "DamerauLevenshtein", "OSADamerauLevenshtein", "Lcs", "Jaro", "JaroWinkler", "Cosine", "Jaccard", "SorensenDice", "Qgram"}, "actual", d.Algorithm)
			return c.JSON(http.StatusInternalServerError, []models.FetchedCPE{})
		}

		rs, err := driver.GetSimilarCpesByTitle(d.Query, d.N, algo)
		if err != nil {
			log15.Error("Failed to GetSimilarCpesByTitle", "err", err)
			return c.JSON(http.StatusInternalServerError, []models.FetchedCPE{})
		}

		return c.JSON(http.StatusOK, rs)
	}
}
