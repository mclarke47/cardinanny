package pkg

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"go.uber.org/zap"
)

type CardinalityScanner struct {
	Logger  *zap.SugaredLogger
	PromAPI v1.API
}

func queryByJob(labelName string) string {
	return fmt.Sprintf("sum({%s=~\".+\"}) by (job)", labelName)
}

func (c *CardinalityScanner) Scan(ctx context.Context, labelCountLimit uint64) (map[string][]string, error) {

	result, err := c.PromAPI.TSDB(ctx)
	if err != nil {
		return nil, fmt.Errorf("error retrieving TSDB stats from the promtheus API, %w", err)
	}

	jobToLabelToDrop := map[string][]string{}

	c.Logger.Debugw("tsdb result", "tsdb.LabelValueCountByLabelName", result.LabelValueCountByLabelName)

	for _, lv := range result.LabelValueCountByLabelName {

		if lv.Value > labelCountLimit {

			r, _, err := c.PromAPI.Query(ctx, queryByJob(lv.Name), time.Now())
			if err != nil {
				return nil, fmt.Errorf("error querying the promtheus API, %w", err)
			}

			if r.Type() == model.ValVector {
				vec := r.(model.Vector)

				c.Logger.Debugw("vector found", "vec", vec)

				for _, v := range vec {
					if job, ok := v.Metric["job"]; ok {
						j := string(job)

						if labels, ok := jobToLabelToDrop[j]; ok {
							jobToLabelToDrop[j] = append(labels, lv.Name)
						} else {
							jobToLabelToDrop[j] = []string{lv.Name}
						}

					}
				}
			}
		}
	}

	return jobToLabelToDrop, nil
}
