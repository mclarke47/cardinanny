package pkg

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	plog "github.com/go-kit/log"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/pkg/relabel"
	"go.uber.org/zap"
)

type PromConfigRewriter struct {
	Logger     *zap.SugaredLogger
	PromAPI    v1.API
	HTTPClient *http.Client
	BaseURL    string
}

func toRegexMap(jobNamesToLabelsToDrop map[string][]string) map[string]string {
	result := map[string]string{}

	for k, labels := range jobNamesToLabelsToDrop {

		var r string
		for _, v := range labels {
			if r == "" {
				r += v
			} else {
				r += fmt.Sprintf("|%s", v)
			}
		}

		result[k] = r
	}

	return result
}

func (p *PromConfigRewriter) DropLabelsInJobs(ctx context.Context, jobNamesToLabelsToDrop map[string][]string, configPath string) error {

	if len(jobNamesToLabelsToDrop) == 0 {
		return nil
	}

	c, err := p.PromAPI.Config(ctx)
	if err != nil {
		return fmt.Errorf("error retrieving the latest config from the promtheus API, %w", err)
	}

	var cfgFile *config.Config
	if cfgFile, err = config.Load(c.YAML, false, plog.NewNopLogger()); err != nil {
		return err
	}

	if len(cfgFile.ScrapeConfigs) == 0 {
		return fmt.Errorf("had labels to drop %v, but no scrapeConfigs in config file at %s", jobNamesToLabelsToDrop, configPath)
	}

	jobNamesToLabelDropRegex := toRegexMap(jobNamesToLabelsToDrop)

	for _, sc := range cfgFile.ScrapeConfigs {

		if v, ok := jobNamesToLabelDropRegex[sc.JobName]; ok {

			sc.MetricRelabelConfigs = append(sc.MetricRelabelConfigs, &relabel.Config{
				Action: relabel.LabelDrop,
				Regex:  relabel.MustNewRegexp(v),
			})
		}
	}
	err = ioutil.WriteFile(configPath, []byte(cfgFile.String()), 0644)
	if err != nil {
		return err
	}

	p.Logger.Debug("Config file generated")

	resp, err := p.HTTPClient.Post(fmt.Sprintf("%s/-/reload", p.BaseURL), "", nil)

	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		b, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error when reloading prometheus config, expected status code 200 but was %d, body was unreadable", resp.StatusCode)
		}
		return fmt.Errorf("error when reloading prometheus config, expected status code 200 but was %d, body: %s", resp.StatusCode, b)
	}

	p.Logger.Debug("Prom config reloaded")

	return nil
}
