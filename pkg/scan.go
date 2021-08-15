package pkg

import (
	"context"
	"fmt"
	"log"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

type CardinalityScanner struct {
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

	for _, lv := range result.LabelValueCountByLabelName {

		if lv.Value > labelCountLimit {
			log.Printf("High cardinality label found key=%s value=%d", lv.Name, lv.Value)

			r, _, err := c.PromAPI.Query(ctx, queryByJob(lv.Name), time.Now())
			if err != nil {
				return nil, fmt.Errorf("error querying the promtheus API, %w", err)
			}

			if r.Type() == model.ValVector {
				vec := r.(model.Vector)
				for _, v := range vec {
					if job, ok := v.Metric["job"]; ok {
						log.Printf("Found bad label %s in job %s\n", lv.Name, job)

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
