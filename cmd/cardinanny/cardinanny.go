package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/mclarke47/cardinanny/pkg"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

type PromContext struct {
	PathToConfigFile string
}

type CardiNanny struct {
	Logger             *zap.SugaredLogger
	CardinalityScanner pkg.CardinalityScanner
	PromConfigRewriter pkg.PromConfigRewriter
	PromContext        PromContext
	PromCleaner        pkg.PromCleaner
}

func newCardiNanny(api v1.API, pathToConfigFile, baseURL string, logger *zap.SugaredLogger) *CardiNanny {
	return &CardiNanny{
		Logger: logger,
		CardinalityScanner: pkg.CardinalityScanner{
			Logger:  logger,
			PromAPI: api,
		},
		PromConfigRewriter: pkg.PromConfigRewriter{
			Logger:     logger,
			PromAPI:    api,
			HTTPClient: &http.Client{},
			BaseURL:    baseURL,
		},
		PromContext: PromContext{
			PathToConfigFile: pathToConfigFile,
		},
		PromCleaner: pkg.PromCleaner{
			Logger:  logger,
			PromAPI: api,
		},
	}
}

func (c *CardiNanny) Start() {
	ticker := time.NewTicker(2 * time.Minute)
	ctx := context.TODO()
	c.ScanForHighLabelCardinality(ctx)

	for {
		select {
		case <-ticker.C:
			ctx := context.TODO()
			c.ScanForHighLabelCardinality(ctx)
		}
	}
}

func (c *CardiNanny) ScanForHighLabelCardinality(ctx context.Context) {
	c.Logger.Infow("starting cardinality scan", "limit", 100)
	jobToLabelToDrop, err := c.CardinalityScanner.Scan(ctx, 100)
	if err != nil {
		c.Logger.Error("Error when scanning", err)
		return
	}

	if len(jobToLabelToDrop) == 0 {
		c.Logger.Infow("starting cardinality scan done, no config changed required")
		return
	}

	c.Logger.Infow("high cardinality labels found", "labels", jobToLabelToDrop)

	err = c.PromConfigRewriter.DropLabelsInJobs(ctx, jobToLabelToDrop, c.PromContext.PathToConfigFile)
	if err != nil {
		c.Logger.Error("Error when updating prometheus config", err)
		return
	}

	// TODO pass job to delete series to ensure we are dropping the right data
	var labelsToDrop []string

	for _, v := range jobToLabelToDrop {
		labelsToDrop = append(labelsToDrop, v...)
	}

	err = c.PromCleaner.Clean(ctx, labelsToDrop)
	if err != nil {
		c.Logger.Error("Error when cleaning high cardinality data", err)
	}
	c.Logger.Info("Cardinality averted")

}

func main() {

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}

	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	baseURL := "http://localhost:9090"

	client, err := api.NewClient(api.Config{
		Address: baseURL,
	})

	if err != nil {
		sugar.Fatal("", err)
	}

	v1api := v1.NewAPI(client)

	configPath := "./prometheus.yml"

	cardinanny := newCardiNanny(v1api, configPath, baseURL, sugar)

	sugar.Infow("starting Cardinanny with",
		"configPath", configPath,
		"prometheusBaseURL", baseURL,
	)

	cardinanny.Start()

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	r.Run() // listen and serve on 0.0.0.0:8080

}
