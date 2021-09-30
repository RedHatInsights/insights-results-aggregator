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
	"testing"
	"time"

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/Shopify/sarama/mocks"
	mapset "github.com/deckarep/golang-set"
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

var testOrgAllowlist = mapset.NewSetWith(testdata.OrgID)

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

func getCounterVecValue(counterVec *prometheus.CounterVec, labels map[string]string) float64 {
	counter, err := counterVec.GetMetricWith(labels)
	if err != nil {
		panic(fmt.Sprintf("Unable to get counter from counterVec %v", err))
	}

	return getCounterValue(counter)
}

//TestConsumedMessagesMetric tests that consumed messages metric works
func TestConsumedMessagesMetric(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		mockConsumer, closer := ira_helpers.MustGetMockKafkaConsumerWithExpectedMessages(
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
		Address:             "localhost:1234",
		Topic:               "consumer-topic",
		PayloadTrackerTopic: "payload-tracker-topic",
		Group:               "test-group",
	}

	// other tests may run at the same process
	initValue := int64(getCounterValue(metrics.ProducedMessages))

	mockProducer := mocks.NewSyncProducer(t, nil)
	mockProducer.ExpectSendMessageAndSucceed()

	kafkaProducer := producer.KafkaProducer{
		Configuration: brokerCfg,
		Producer:      mockProducer,
	}
	defer func() {
		helpers.FailOnError(t, kafkaProducer.Close())
	}()

	err := kafkaProducer.TrackPayload(testdata.TestRequestID, testdata.LastCheckedAt, producer.StatusReceived)
	helpers.FailOnError(t, err)

	assertCounterValue(t, 1, metrics.ProducedMessages, initValue)
}

func TestWrittenReportsMetric(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	// other tests may run at the same process
	initValue := int64(getCounterValue(metrics.WrittenReports))

	err := mockStorage.WriteReportForCluster(testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.Report3RulesParsed, testdata.LastCheckedAt, 0)
	helpers.FailOnError(t, err)

	assertCounterValue(t, 1, metrics.WrittenReports, initValue)

	for i := 0; i < 99; i++ {
		err := mockStorage.WriteReportForCluster(
			testdata.OrgID,
			testdata.ClusterName,
			testdata.Report3Rules,
			testdata.Report3RulesParsed,
			testdata.LastCheckedAt.Add(time.Duration(i+1)*time.Second),
			types.KafkaOffset(i+1),
		)
		helpers.FailOnError(t, err)
	}

	assertCounterValue(t, 100, metrics.WrittenReports, initValue)
}

// TODO: write tests for sql queries metrics
// - SQLQueriesCounter
// - SQLQueriesDurations
