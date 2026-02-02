package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/hbollon/go-edlib"
	"github.com/inconshreveable/log15"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/spf13/viper"
	"golang.org/x/xerrors"

	"github.com/vulsio/go-cpe-dictionary/db"
	"github.com/vulsio/go-cpe-dictionary/models"
)

// Start starts CVE dictionary HTTP Server.
func Start(logToFile bool, logDir string, driver db.DB) error {
	e := echo.New()
	e.Debug = viper.GetBool("debug")

	// Middleware
	e.Use(middleware.RequestLoggerWithConfig(newRequestLoggerConfig(os.Stderr)))
	e.Use(middleware.Recover())

	// setup access logger
	if logToFile {
		logPath := filepath.Join(logDir, "access.log")
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return xerrors.Errorf("Failed to open a log file: %s", err)
		}
		defer f.Close()
		e.Use(middleware.RequestLoggerWithConfig(newRequestLoggerConfig(f)))
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

func newRequestLoggerConfig(writer io.Writer) middleware.RequestLoggerConfig {
	return middleware.RequestLoggerConfig{
		LogLatency:       true,
		LogRemoteIP:      true,
		LogHost:          true,
		LogMethod:        true,
		LogURI:           true,
		LogRequestID:     true,
		LogUserAgent:     true,
		LogStatus:        true,
		LogError:         true,
		LogContentLength: true,
		LogResponseSize:  true,

		LogValuesFunc: func(_ echo.Context, v middleware.RequestLoggerValues) error {
			errStr := ""
			if v.Error != nil {
				errStr = v.Error.Error()
			}

			type logFormat struct {
				Time         string `json:"time"`
				RemoteIP     string `json:"remote_ip"`
				Host         string `json:"host"`
				Method       string `json:"method"`
				URI          string `json:"uri"`
				ID           string `json:"id"`
				UserAgent    string `json:"user_agent"`
				Status       int    `json:"status"`
				Error        string `json:"error"`
				Latency      int64  `json:"latency"`
				LatencyHuman string `json:"latency_human"`
				BytesIn      int64  `json:"bytes_in"`
				BytesOut     int64  `json:"bytes_out"`
			}

			return json.NewEncoder(writer).Encode(logFormat{
				Time:         v.StartTime.Format(time.RFC3339Nano),
				RemoteIP:     v.RemoteIP,
				Host:         v.Host,
				Method:       v.Method,
				URI:          v.URI,
				ID:           v.RequestID,
				UserAgent:    v.UserAgent,
				Status:       v.Status,
				Error:        errStr,
				Latency:      v.Latency.Nanoseconds(),
				LatencyHuman: v.Latency.String(),
				BytesIn: func() int64 {
					i, _ := strconv.ParseInt(v.ContentLength, 10, 64)
					return i
				}(),
				BytesOut: v.ResponseSize,
			})
		},
	}
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
