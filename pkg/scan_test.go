package pkg

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/mclarke47/cardinanny/mock_v1"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func Test_CardinalityScanner_scanEmptyLabelValueCountByLabelName(t *testing.T) {
	runTest(t, []v1.Stat{}, map[string][]string{})
}

func Test_CardinalityScanner_scanValuesUnderCardinalityLimit(t *testing.T) {

	runTest(t, []v1.Stat{
		{
			Name:  "v1",
			Value: 1,
		},
		{
			Name:  "v2",
			Value: 49,
		},
		{
			Name:  "v3",
			Value: 50,
		},
	}, map[string][]string{})
}

func Test_CardinalityScanner_scanOneValueOverLimit(t *testing.T) {

	runTest(t, []v1.Stat{
		{
			Name:  "v1",
			Value: 1,
		},
		{
			Name:  "v2",
			Value: 49,
		},
		{
			Name:  "v3",
			Value: 51,
		},
	}, map[string][]string{
		"some-job": {"v3"},
	})
}

func Test_CardinalityScanner_scanTwoValueOverLimit(t *testing.T) {

	runTest(t, []v1.Stat{
		{
			Name:  "v1",
			Value: 1,
		},
		{
			Name:  "v2",
			Value: 1000,
		},
		{
			Name:  "v3",
			Value: 51,
		},
	}, map[string][]string{
		"some-job":       {"v3"},
		"some-other-job": {"v2"},
	})
}

func Test_CardinalityScanner_scanTwoValueOverLimitInSameJob(t *testing.T) {

	runTest(t, []v1.Stat{
		{
			Name:  "v1",
			Value: 1,
		},
		{
			Name:  "v2",
			Value: 1000,
		},
		{
			Name:  "v3",
			Value: 51,
		},
	}, map[string][]string{
		"some-job": {"v2", "v3"},
	})
}

func Test_CardinalityScanner_scanOneValueOverLimitNoJobLabel(t *testing.T) {

	ctrl := gomock.NewController(t)

	m := mock_v1.NewMockAPI(ctrl)

	m.
		EXPECT().
		TSDB(gomock.Any()).
		Return(v1.TSDBResult{
			LabelValueCountByLabelName: []v1.Stat{
				{
					Name:  "v1",
					Value: 1,
				},
				{
					Name:  "v2",
					Value: 49,
				},
				{
					Name:  "v3",
					Value: 51,
				},
			},
		}, nil).
		MaxTimes(1)

	m.
		EXPECT().
		Query(gomock.Any(), gomock.Eq("sum({v3=~\".+\"}) by (job)"), gomock.Any()). // TODO fix time expect
		Return(model.Vector{
			{
				Metric: model.Metric{"somethingelse": model.LabelValue("something")},
				Value:  10000000,
			},
		}, nil, nil).
		MaxTimes(1)

	scanner := CardinalityScanner{
		PromAPI:         m,
		Logger:          zap.NewNop().Sugar(),
		LabelCountLimit: 50,
	}

	result, err := scanner.Scan(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, map[string][]string{}, result)
}

func Test_CardinalityScanner_scanHandleTSDBReturnsError(t *testing.T) {

	ctrl := gomock.NewController(t)

	m := mock_v1.NewMockAPI(ctrl)

	m.
		EXPECT().
		TSDB(gomock.Any()).
		Return(v1.TSDBResult{}, errors.New("some-error")).
		MaxTimes(1)

	scanner := CardinalityScanner{
		PromAPI:         m,
		Logger:          zap.NewNop().Sugar(),
		LabelCountLimit: 50,
	}

	result, err := scanner.Scan(context.Background())
	assert.NotNil(t, err)
	assert.Equal(t, "error retrieving TSDB stats from the promtheus API, some-error", err.Error())
	assert.Nil(t, result)
}

func Test_CardinalityScanner_scanHandleQueryReturnsError(t *testing.T) {

	ctrl := gomock.NewController(t)

	m := mock_v1.NewMockAPI(ctrl)

	m.
		EXPECT().
		TSDB(gomock.Any()).
		Return(v1.TSDBResult{
			LabelValueCountByLabelName: []v1.Stat{
				{
					Name:  "v1",
					Value: 1,
				},
				{
					Name:  "v2",
					Value: 49,
				},
				{
					Name:  "v3",
					Value: 51,
				},
			},
		}, nil).
		MaxTimes(1)

	m.
		EXPECT().
		Query(gomock.Any(), gomock.Eq("sum({v3=~\".+\"}) by (job)"), gomock.Any()). // TODO fix time expect
		Return(model.Vector{}, nil, errors.New("some-error")).
		MaxTimes(1)

	scanner := CardinalityScanner{
		PromAPI:         m,
		Logger:          zap.NewNop().Sugar(),
		LabelCountLimit: 50,
	}

	result, err := scanner.Scan(context.Background())
	assert.NotNil(t, err)
	assert.Equal(t, "error querying the promtheus API, some-error", err.Error())
	assert.Nil(t, result)
}

func runTest(t *testing.T, labelValueCountByLabelName []v1.Stat, expectedResult map[string][]string) {
	ctrl := gomock.NewController(t)

	m := mock_v1.NewMockAPI(ctrl)

	m.
		EXPECT().
		TSDB(gomock.Any()).
		Return(v1.TSDBResult{
			LabelValueCountByLabelName: labelValueCountByLabelName,
		}, nil).
		MaxTimes(1)

	for k, labels := range expectedResult {
		for _, v := range labels {
			m.
				EXPECT().
				Query(gomock.Any(), gomock.Eq(fmt.Sprintf("sum({%s=~\".+\"}) by (job)", v)), gomock.Any()). // TODO fix time expect
				Return(model.Vector{
					{
						Metric: model.Metric{"job": model.LabelValue(k)},
						Value:  10000000,
					},
				}, nil, nil).
				MaxTimes(1)
		}
	}

	scanner := CardinalityScanner{
		PromAPI:         m,
		Logger:          zap.NewNop().Sugar(),
		LabelCountLimit: 50,
	}

	result, err := scanner.Scan(context.Background())
	assert.Nil(t, err)
	assert.Equal(t, expectedResult, result)
}
