package pkg

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/mclarke47/cardinanny/mock_v1"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func Test_PromCleaner_Clean(t *testing.T) {

	ctrl := gomock.NewController(t)

	m := mock_v1.NewMockAPI(ctrl)

	pc := PromCleaner{
		Logger:  zap.NewNop().Sugar(),
		PromAPI: m,
	}

	m.
		EXPECT().
		DeleteSeries(gomock.Any(), gomock.Eq([]string{"{label1=~\".+\"}", "{otherlabel2=~\".+\"}"}), gomock.Any(), gomock.Any()).
		Return(nil).
		MaxTimes(1)

	m.
		EXPECT().
		CleanTombstones(gomock.Any()).
		Return(nil).
		MaxTimes(1)

	err := pc.Clean(context.Background(), []string{"label1", "otherlabel2"})

	assert.Nil(t, err)
}

func Test_PromCleaner_DeleteReturnsError(t *testing.T) {

	ctrl := gomock.NewController(t)

	m := mock_v1.NewMockAPI(ctrl)

	pc := PromCleaner{
		Logger:  zap.NewNop().Sugar(),
		PromAPI: m,
	}

	m.
		EXPECT().
		DeleteSeries(gomock.Any(), gomock.Eq([]string{"{label1=~\".+\"}", "{otherlabel2=~\".+\"}"}), gomock.Any(), gomock.Any()).
		Return(errors.New("some-error")).
		MaxTimes(1)

	err := pc.Clean(context.Background(), []string{"label1", "otherlabel2"})

	assert.NotNil(t, err)
	assert.Equal(t, "error while deleting label data [label1 otherlabel2] for query [{label1=~\".+\"} {otherlabel2=~\".+\"}], error some-error", err.Error())
}

func Test_PromCleaner_CleanTombstonesReturnsError(t *testing.T) {

	ctrl := gomock.NewController(t)

	m := mock_v1.NewMockAPI(ctrl)

	pc := PromCleaner{
		Logger:  zap.NewNop().Sugar(),
		PromAPI: m,
	}

	m.
		EXPECT().
		DeleteSeries(gomock.Any(), gomock.Eq([]string{"{label1=~\".+\"}", "{otherlabel2=~\".+\"}"}), gomock.Any(), gomock.Any()).
		Return(nil).
		MaxTimes(1)

	m.
		EXPECT().
		CleanTombstones(gomock.Any()).
		Return(errors.New("some-error")).
		MaxTimes(1)

	err := pc.Clean(context.Background(), []string{"label1", "otherlabel2"})

	assert.NotNil(t, err)
	assert.Equal(t, "error while cleaning tombstones for label data [label1 otherlabel2], error some-error", err.Error())
}
