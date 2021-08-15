package pkg

import (
	"context"
	"fmt"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"go.uber.org/zap"
)

type PromCleaner struct {
	Logger  *zap.SugaredLogger
	PromAPI v1.API
}

func query(labelName string) string {
	return fmt.Sprintf("{%s=~\".+\"}", labelName)
}

func (p *PromCleaner) Clean(ctx context.Context, labelsToDrop []string) error {

	var seriesToDrop []string

	for _, l := range labelsToDrop {
		seriesToDrop = append(seriesToDrop, query(l))
	}

	p.Logger.Debugw("deleting series", "series", seriesToDrop)

	err := p.PromAPI.DeleteSeries(ctx, seriesToDrop, time.Now().Add(-time.Hour), time.Now())
	if err != nil {
		return fmt.Errorf("error while deleting label data %v for query %v, error %v", labelsToDrop, seriesToDrop, err)
	}

	err = p.PromAPI.CleanTombstones(ctx)
	if err != nil {
		return fmt.Errorf("error while cleaning tombstones for label data %v, error %v", labelsToDrop, err)
	}
	return nil
}
