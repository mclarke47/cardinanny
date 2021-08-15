package pkg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/mclarke47/cardinanny/mock_v1"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/stretchr/testify/assert"
)

func yamlFixture(t *testing.T, file string) string {
	yaml, err := ioutil.ReadFile(file)

	assert.Nil(t, err)
	return string(yaml)
}

func TestConfigWriter_emptymap(t *testing.T) {

	ctrl := gomock.NewController(t)

	m := mock_v1.NewMockAPI(ctrl)

	writer := PromConfigRewriter{
		PromAPI: m,
	}

	emptyMap := map[string][]string{}

	err := writer.DropLabelsInJobs(context.Background(), emptyMap, "")

	assert.Nil(t, err)
}

func TestConfigWriter_promAPIReturnsError(t *testing.T) {

	ctrl := gomock.NewController(t)

	m := mock_v1.NewMockAPI(ctrl)
	m.
		EXPECT().
		Config(gomock.Any()).
		Return(v1.ConfigResult{}, errors.New("some-error")).
		MaxTimes(1)

	writer := PromConfigRewriter{
		PromAPI: m,
	}

	oneValue := map[string][]string{
		"some-job": {"somevalue"},
	}

	err := writer.DropLabelsInJobs(context.Background(), oneValue, "../some/path")

	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "error retrieving the latest config from the promtheus API, some-error")
}

func TestConfigWriter_cantWriteOutNewConfigFile(t *testing.T) {

	yaml := yamlFixture(t, "./fixtures/2-scrape-jobs.yaml")

	ctrl := gomock.NewController(t)

	m := mock_v1.NewMockAPI(ctrl)
	m.
		EXPECT().
		Config(gomock.Any()).
		Return(v1.ConfigResult{
			YAML: string(yaml),
		}, nil).
		MaxTimes(1)

	writer := PromConfigRewriter{
		PromAPI: m,
	}

	oneValue := map[string][]string{
		"some-job": {"somevalue"},
	}

	err := writer.DropLabelsInJobs(context.Background(), oneValue, "/some/path/that/doesnt/exist")

	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "open /some/path/that/doesnt/exist: no such file or directory")
}

func TestConfigWriter_promConfigParseReturnsError(t *testing.T) {

	ctrl := gomock.NewController(t)

	m := mock_v1.NewMockAPI(ctrl)
	m.
		EXPECT().
		Config(gomock.Any()).
		Return(v1.ConfigResult{
			YAML: "something:\n  invalid",
		}, nil).
		MaxTimes(1)

	writer := PromConfigRewriter{
		PromAPI: m,
	}

	oneValue := map[string][]string{
		"some-job": {"somevalue"},
	}

	err := writer.DropLabelsInJobs(context.Background(), oneValue, "../some/path")

	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "yaml: unmarshal errors:\n  line 1: field something not found in type config.plain")
}

func TestConfigWriter_ValuesToDropButNoConfigFile(t *testing.T) {

	ctrl := gomock.NewController(t)

	m := mock_v1.NewMockAPI(ctrl)
	m.
		EXPECT().
		Config(gomock.Any()).
		Return(v1.ConfigResult{}, nil).
		MaxTimes(1)

	writer := PromConfigRewriter{
		PromAPI: m,
	}

	oneValue := map[string][]string{
		"some-job": {"somevalue"},
	}

	err := writer.DropLabelsInJobs(context.Background(), oneValue, "../some/path")

	assert.NotNil(t, err)
	assert.Equal(t, err.Error(), "had labels to drop map[some-job:[somevalue]], but no scrapeConfigs in config file at ../some/path")
}

func TestConfigWriter_oneLabelToDrop(t *testing.T) {
	testLabelDropping(
		t,
		"./fixtures/2-scrape-jobs.yaml",
		"./fixtures/2-scrape-jobs-expected-1-label.yaml",
		map[string][]string{
			"some-job": {"somevalue"},
		},
	)
}

func TestConfigWriter_twoLabelsInOneJobToDrop(t *testing.T) {
	testLabelDropping(
		t,
		"./fixtures/2-scrape-jobs.yaml",
		"./fixtures/2-scrape-jobs-expected-2-label.yaml",
		map[string][]string{
			"some-job": {"somevalue", "anotherBadLabel"},
		},
	)
}

func TestConfigWriter_twoLabelsInTwoJobsToDrop(t *testing.T) {
	testLabelDropping(
		t,
		"./fixtures/2-scrape-jobs.yaml",
		"./fixtures/2-scrape-jobs-expected-1-label-each.yaml",
		map[string][]string{
			"some-job":       {"anotherBadLabel"},
			"some-other-job": {"somevalue"},
		},
	)
}

func testLabelDropping(t *testing.T, inputYamlFixturePath string, expectedYamlFixturePath string, jobsToLabels map[string][]string) {

	tempFile, err := ioutil.TempFile("", fmt.Sprintf("%s.yaml", t.Name()))
	assert.Nil(t, err)

	yaml := yamlFixture(t, inputYamlFixturePath)

	ctrl := gomock.NewController(t)

	m := mock_v1.NewMockAPI(ctrl)
	m.
		EXPECT().
		Config(gomock.Any()).
		Return(v1.ConfigResult{
			YAML: string(yaml),
		}, nil).
		MaxTimes(1)

	writer := PromConfigRewriter{
		PromAPI: m,
	}

	err = writer.DropLabelsInJobs(context.Background(), jobsToLabels, tempFile.Name())

	assert.Nil(t, err)

	assertConfigFilesAreEqual(t, expectedYamlFixturePath, tempFile)
}

func assertConfigFilesAreEqual(t *testing.T, expectedFilePath string, actualFile io.Reader) {

	expected := yamlFixture(t, expectedFilePath)

	actual, err := ioutil.ReadAll(actualFile)

	assert.Equal(t, expected, string(actual))

	assert.Nil(t, err)
}
