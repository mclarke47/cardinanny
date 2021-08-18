package main

import (
	"context"
	"flag"
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
	Summary            map[string][]string
}

func newCardiNanny(api v1.API, pathToConfigFile, baseURL string, logger *zap.SugaredLogger, labelLimit uint64) *CardiNanny {
	return &CardiNanny{
		Summary: map[string][]string{},
		Logger:  logger,
		CardinalityScanner: pkg.CardinalityScanner{
			Logger:          logger,
			PromAPI:         api,
			LabelCountLimit: labelLimit,
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

func (c *CardiNanny) addToSummary(jobNamesToLabelsToDrop map[string][]string) {
	for k, v := range jobNamesToLabelsToDrop {
		if oldVal, ok := c.Summary[k]; ok {
			c.Summary[k] = append(oldVal, v...)
		} else {
			c.Summary[k] = v
		}
	}
}

func (c *CardiNanny) ScanForHighLabelCardinality(ctx context.Context) {
	c.Logger.Infow("starting cardinality scan", "limit", c.CardinalityScanner.LabelCountLimit)
	jobToLabelToDrop, err := c.CardinalityScanner.Scan(ctx)
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
	c.addToSummary(jobToLabelToDrop)

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

	promFilePath := flag.String("prometheusConfigFile", "./prometheus.yml", "path to the prometheus config file")
	promBaseURL := flag.String("prometheusBaseURL", "http://localhost:9090", "the base URL to use to connect to prometheus")
	labelLimit := flag.Int("cardinalityLabelLimit", 1000000, "the mac number of values a label can have")

	flag.Parse()

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}

	defer logger.Sync() // flushes buffer, if any
	sugar := logger.Sugar()

	client, err := api.NewClient(api.Config{
		Address: *promBaseURL,
	})

	if err != nil {
		sugar.Fatal("", err)
	}

	v1api := v1.NewAPI(client)

	cardinanny := newCardiNanny(v1api, *promFilePath, *promBaseURL, sugar, uint64(*labelLimit))

	sugar.Infow("starting Cardinanny with",
		"configPath", promFilePath,
		"prometheusBaseURL", promBaseURL,
		"cardinalityLabelLimit", labelLimit,
	)

	go cardinanny.Start()

	r := gin.Default()
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})
	r.GET("/summary", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"summary": cardinanny.Summary,
		})
	})
	r.Run() // listen and serve on 0.0.0.0:8080

}
