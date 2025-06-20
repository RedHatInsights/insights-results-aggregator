// Copyright 2020, 2021, 2022, 2023 Red Hat, Inc
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
	"testing"
	"time"

	"github.com/IBM/sarama/mocks"
	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/prometheus/client_golang/prometheus"
	prommodels "github.com/prometheus/client_model/go"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/metrics"
	"github.com/RedHatInsights/insights-results-aggregator/producer"
	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const (
	testTopicName     = "ccx.ocp.results"
	testCaseTimeLimit = 60 * time.Second
)

var testOrgAllowlist = mapset.NewSet(testdata.OrgID)

func init() {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
}

func assertCounterValue(tb testing.TB, expected int64, counter prometheus.Counter, initValue int64) {
	assert.Equal(tb, float64(expected+initValue), getCounterValue(counter))
}

func getCounterValue(counter prometheus.Counter) float64 {
	pb := &prommodels.Metric{}
	err := counter.Write(pb)
	if err != nil {
		panic(fmt.Sprintf("Unable to get counter from counter %v", err))
	}

	return pb.GetCounter().GetValue()
}

// TestConsumedOCPMessagesMetric tests that consumed messages metric works
func TestConsumedOCPMessagesMetric(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		mockConsumer, closer := ira_helpers.MustGetMockOCPRulesConsumerWithExpectedMessages(
			t, testTopicName, testOrgAllowlist, []string{testdata.ConsumerMessage, testdata.ConsumerMessage},
		)
		defer closer()

		assert.Equal(t, 0.0, getCounterValue(metrics.ConsumedMessages))

		go mockConsumer.Serve()

		ira_helpers.WaitForMockConsumerToHaveNConsumedMessages(mockConsumer, 2)

		assert.Equal(t, 2.0, getCounterValue(metrics.ConsumedMessages))
	}, testCaseTimeLimit)
}

func TestProducedMessagesMetric(t *testing.T) {
	brokerCfg := broker.Configuration{
		Addresses:           "localhost:1234",
		Topic:               "consumer-topic",
		PayloadTrackerTopic: "payload-tracker-topic",
		Group:               "test-group",
	}

	// other tests may run at the same process
	initValue := int64(getCounterValue(metrics.ProducedMessages))

	mockProducer := mocks.NewSyncProducer(t, nil)
	mockProducer.ExpectSendMessageAndSucceed()

	payloadTrackerProducer := producer.PayloadTrackerProducer{
		KafkaProducer: producer.KafkaProducer{Producer: mockProducer},
		Configuration: brokerCfg,
	}
	defer func() {
		helpers.FailOnError(t, payloadTrackerProducer.Close())
	}()

	orgID := types.OrgID(1)
	account := types.Account(1)
	err := payloadTrackerProducer.TrackPayload(
		testdata.TestRequestID,
		testdata.LastCheckedAt,
		&orgID,
		&account,
		producer.StatusReceived,
	)
	helpers.FailOnError(t, err)

	assertCounterValue(t, 1, metrics.ProducedMessages, initValue)
}

func TestWrittenReportsMetric(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	// other tests may run at the same process
	initValue := int64(getCounterValue(metrics.WrittenReports))

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.Report3Rules,
		testdata.Report3RulesParsed,
		testdata.LastCheckedAt,
		time.Now(),
		time.Now(),
		testdata.RequestID1,
	)
	helpers.FailOnError(t, err)

	assertCounterValue(t, 1, metrics.WrittenReports, initValue)

	for i := 0; i < 99; i++ {
		err := mockStorage.WriteReportForCluster(
			testdata.OrgID,
			testdata.ClusterName,
			testdata.Report3Rules,
			testdata.Report3RulesParsed,
			testdata.LastCheckedAt.Add(time.Duration(i+1)*time.Second),
			time.Now(),
			time.Now(),
			testdata.RequestID1,
		)
		helpers.FailOnError(t, err)
	}

	assertCounterValue(t, 100, metrics.WrittenReports, initValue)
}

// unregisterAllMetrics manually unregisters metrics before use
func unregisterAllMetrics() {
	prometheus.Unregister(metrics.ConsumedMessages)
	prometheus.Unregister(metrics.ConsumingErrors)
	prometheus.Unregister(metrics.SkippedEmptyReports)
	prometheus.Unregister(metrics.SuccessfulMessagesProcessingTime)
	prometheus.Unregister(metrics.FailedMessagesProcessingTime)
	prometheus.Unregister(metrics.LastCheckedTimestampLagMinutes)
	prometheus.Unregister(metrics.ProducedMessages)
	prometheus.Unregister(metrics.WrittenReports)
	prometheus.Unregister(metrics.FeedbackOnRules)
	prometheus.Unregister(metrics.SQLQueriesCounter)
}

// TestCounterMetrics checks whether incrementing metrics
// works as expected
func TestCounterMetrics(t *testing.T) {
	type testCase struct {
		name    string
		counter prometheus.Counter
	}
	testCases := []testCase{
		{
			name:    "ConsumedMessagesMetric",
			counter: metrics.ConsumedMessages,
		},
		{
			name:    "ConsumingErrorsMetric",
			counter: metrics.ConsumingErrors,
		},
		{
			name:    "SkippedEmptyReportsMetric",
			counter: metrics.SkippedEmptyReports,
		},
		{
			name:    "ProducedMessagesMetric",
			counter: metrics.ProducedMessages,
		},
		{
			name:    "WrittenReportsMetric",
			counter: metrics.WrittenReports,
		},
		{
			name:    "FeedbackOnRulesMetric",
			counter: metrics.FeedbackOnRules,
		},
		{
			name:    "SQLQueriesCounterMetric",
			counter: metrics.SQLQueriesCounter,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			unregisterAllMetrics()

			metrics.AddMetricsWithNamespace(tc.name)

			initValue := int64(getCounterValue(tc.counter))
			assertCounterValue(t, 0, tc.counter, initValue)

			tc.counter.Inc()
			tc.counter.Inc()

			expectedValue := int64(2)
			assertCounterValue(t, expectedValue, tc.counter, initValue)
		})
	}
}

// TestHistogramMetrics checks whether incrementing time metrics
// works as expected
func TestHistogramMetrics(t *testing.T) {
	type testCase struct {
		name      string
		histogram prometheus.Histogram
	}
	testCases := []testCase{
		{
			name:      "SuccessfulMessagesProcessingTimeMetric",
			histogram: metrics.SuccessfulMessagesProcessingTime,
		},
		{
			name:      "FailedMessagesProcessingTimeMetric",
			histogram: metrics.FailedMessagesProcessingTime,
		},
		{
			name:      "LastCheckedTimestampLagMinutesMetric",
			histogram: metrics.LastCheckedTimestampLagMinutes,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			unregisterAllMetrics()

			metrics.AddMetricsWithNamespace(tc.name)

			tc.histogram.Observe(1.5)
			tc.histogram.Observe(2.0)

			metric := &prommodels.Metric{}
			err := tc.histogram.Write(metric)
			helpers.FailOnError(t, err)

			count := metric.GetHistogram().GetSampleCount()
			expectedCount := uint64(2)
			assert.Equal(t, expectedCount, count)

			sum := metric.GetHistogram().GetSampleSum()
			assert.InDelta(t, 3.5, sum, 0.01)
		})
	}
}

// TODO: write tests for sql queries metrics
// - SQLQueriesCounter
// - SQLQueriesDurations
