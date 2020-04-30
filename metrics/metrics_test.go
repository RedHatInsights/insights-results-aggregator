// Copyright 2020 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package metrics_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/prometheus/client_golang/prometheus"
	prom_models "github.com/prometheus/client_model/go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/metrics"
	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/tests/testdata"
)

const (
	testTopicName     = "ccx.ocp.results"
	testCaseTimeLimit = 30 * time.Second
)

var (
	testOrgWhiteList = mapset.NewSetWith(testdata.OrgID)
)

func init() {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
}

func getCounterValue(counter prometheus.Counter) float64 {
	pb := &prom_models.Metric{}
	err := counter.Write(pb)
	if err != nil {
		panic(fmt.Sprintf("Unable to get counter from counter %v", err))
	}

	return pb.GetCounter().GetValue()
}

func getCounterVecValue(counterVec *prometheus.CounterVec, labels map[string]string) float64 {
	counter, err := counterVec.GetMetricWith(labels)
	if err != nil {
		panic(fmt.Sprintf("Unable to get counter from counterVec %v", err))
	}

	return getCounterValue(counter)
}

// TestConsumedMessagesMetric tests that consumed messages metric works
func TestConsumedMessagesMetric(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		mockConsumer, closer := helpers.MustGetMockKafkaConsumerWithExpectedMessages(
			t, testTopicName, testOrgWhiteList, []string{testdata.ConsumerMessage, testdata.ConsumerMessage},
		)
		defer closer()

		assert.Equal(t, 0.0, getCounterValue(metrics.ConsumedMessages))

		go mockConsumer.Serve()

		helpers.WaitForMockConsumerToHaveNConsumedMessages(mockConsumer, 2)

		assert.Equal(t, 2.0, getCounterValue(metrics.ConsumedMessages))
	}, testCaseTimeLimit)
}

func TestAPIRequestsMetrics(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		// resetting since go runs tests in 1 process
		metrics.APIRequests.Reset()

		endpoint := helpers.DefaultServerConfig.APIPrefix + server.ReportEndpoint

		assert.Equal(t, 0.0, getCounterVecValue(metrics.APIRequests, map[string]string{
			"endpoint": endpoint,
		}))

		helpers.AssertAPIRequest(t, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, testdata.BadClusterName},
		}, &helpers.APIResponse{
			StatusCode: http.StatusBadRequest,
			Body: `{
				"status": "Error during parsing param 'cluster' with value 'aaaa'. Error: 'invalid UUID length: 4'"
			}`,
		})

		assert.Equal(t, 1.0, getCounterVecValue(metrics.APIRequests, map[string]string{
			"endpoint": endpoint,
		}))
	}, testCaseTimeLimit)
}

// TODO: metrics.APIResponsesTime
// TODO: metrics.ProducedMessages
// TODO: metrics.WrittenReports

func TestApiResponseStatusCodesMetric_StatusOK(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		metrics.APIResponseStatusCodes.Reset()

		assert.Equal(t, 0.0, getCounterVecValue(metrics.APIResponseStatusCodes, map[string]string{
			"status_code": fmt.Sprint(http.StatusOK),
		}))

		for i := 0; i < 15; i++ {
			helpers.AssertAPIRequest(t, nil, nil, &helpers.APIRequest{
				Method:   http.MethodGet,
				Endpoint: server.MainEndpoint,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       `{"status": "ok"}`,
			})
		}

		assert.Equal(t, 15.0, getCounterVecValue(metrics.APIResponseStatusCodes, map[string]string{
			"status_code": fmt.Sprint(http.StatusOK),
		}))
	}, testCaseTimeLimit)
}

func TestApiResponseStatusCodesMetric_StatusBadRequest(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		metrics.APIResponseStatusCodes.Reset()

		assert.Equal(t, 0.0, getCounterVecValue(metrics.APIResponseStatusCodes, map[string]string{
			"status_code": fmt.Sprint(http.StatusBadRequest),
		}))

		helpers.AssertAPIRequest(t, nil, nil, &helpers.APIRequest{
			Method:       http.MethodGet,
			Endpoint:     server.ReportEndpoint,
			EndpointArgs: []interface{}{testdata.OrgID, testdata.BadClusterName},
		}, &helpers.APIResponse{
			StatusCode: http.StatusBadRequest,
			Body: `{
				"status": "Error during parsing param 'cluster' with value 'aaaa'. Error: 'invalid UUID length: 4'"
			}`,
		})

		assert.Equal(t, 1.0, getCounterVecValue(metrics.APIResponseStatusCodes, map[string]string{
			"status_code": fmt.Sprint(http.StatusBadRequest),
		}))
	}, testCaseTimeLimit)
}
