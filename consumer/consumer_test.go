/*
Copyright © 2020, 2021, 2022, 2023 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package consumer_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/RedHatInsights/insights-results-aggregator/producer"

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/IBM/sarama"
	mapset "github.com/deckarep/golang-set"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/consumer"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const (
	testTopicName = "topic"
	// time limit for *some* tests which can stuck in forever loop
	testCaseTimeLimit = 60 * time.Second
	saramaLogPrefix   = "[sarama]"

	// message to be checked
	organizationIDNotInAllowList = "organization ID is not in allow list"

	testReport  = `{"fingerprints": [], "info": [], "skips": [], "system": {}, "analysis_metadata":{"metadata":"some metadata"},"reports":[{"rule_id":"rule_4|RULE_4","component":"ccx_rules_ocp.external.rules.rule_1.report","type":"rule","key":"RULE_4","details":"some details"},{"rule_id":"rule_4|RULE_4","component":"ccx_rules_ocp.external.rules.rule_2.report","type":"rule","key":"RULE_2","details":"some details"},{"rule_id":"rule_5|RULE_5","component":"ccx_rules_ocp.external.rules.rule_5.report","type":"rule","key":"RULE_3","details":"some details"}]}`
	testMetrics = `{"system":{"metadata":{},"hostname":null},"fingerprints":[],"version":1,"analysis_metadata":{},"workload_recommendations":[{"response_id":"an_issue|DVO_AN_ISSUE","component":"ccx_rules_ocp.external.dvo.an_issue_pod.recommendation","key":"DVO_AN_ISSUE","details":{"check_name":"","check_url":"","samples":[{"namespace_uid":"NAMESPACE-UID-A","kind":"DaemonSet","uid":"193a2099-1234-5678-916a-d570c9aac158"}]},"tags":[],"links":{"jira":["https://issues.redhat.com/browse/AN_ISSUE"],"product_documentation":[]},"workloads":[{"namespace":"namespace-name-A","namespace_uid":"NAMESPACE-UID-A","kind":"DaemonSet","name":"test-name-0099","uid":"193a2099-1234-5678-916a-d570c9aac158"}]}]}`
)

var (
	testOrgAllowlist = mapset.NewSetWith(types.OrgID(1))
	wrongBrokerCfg   = broker.Configuration{
		Addresses: "localhost:1234",
		Topic:     "topic",
		Group:     "group",
	}
	messageReportWithRuleHits = `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Report":` + testReport + `,
		"LastChecked": "` + testdata.LastCheckedAt.UTC().Format(time.RFC3339) + `"
	}`
	messageReportWithDVOHits = `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Metrics":` + testMetrics + `,
		"LastChecked": "` + testdata.LastCheckedAt.UTC().Format(time.RFC3339) + `"
	}`

	messageNoReportsNoInfo = `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"LastChecked": "` + testdata.LastCheckedAt.Format(time.RFC3339) + `",
		"Report": {
			"system": {
				"metadata": {},
				"hostname": null
			},
			"fingerprints": [],
			"skips": []
		}
	}`

	messageReportNoDVOMetrics = `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Metrics": {},
		"LastChecked": "` + testdata.LastCheckedAt.UTC().Format(time.RFC3339) + `"
	}`
)

func init() {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
}

func consumerProcessMessage(mockConsumer consumer.Consumer, message string) error {
	saramaMessage := sarama.ConsumerMessage{}
	saramaMessage.Value = []byte(message)
	return mockConsumer.HandleMessage(&saramaMessage)
}

func mustConsumerProcessMessage(t testing.TB, mockConsumer consumer.Consumer, message string) {
	helpers.FailOnError(t, consumerProcessMessage(mockConsumer, message))
}

func createConsumerMessage(report string) string {
	consumerMessage := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + fmt.Sprint(testdata.ClusterName) + `",
		"LastChecked": "` + testdata.LastCheckedAt.UTC().Format(time.RFC3339) + `",
		"Report": ` + report + `
	}
	`
	return consumerMessage
}

func unmarshall(s string) *json.RawMessage {
	var res json.RawMessage
	err := json.Unmarshal([]byte(s), &res)
	if err != nil {
		return nil
	}
	return &res
}

func TestConsumerConstructorNoKafka(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, false)
	defer closer()

	mockConsumer, err := consumer.NewKafkaConsumer(wrongBrokerCfg, mockStorage, nil)
	assert.Error(t, err)
	assert.Contains(
		t, err.Error(), "kafka: client has run out of available brokers to talk to",
	)
	assert.Equal(
		t,
		(*consumer.KafkaConsumer)(nil),
		mockConsumer,
		"consumer.New should return nil instead of Consumer implementation",
	)
}

func TestKafkaConsumer_New(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		sarama.Logger = log.New(os.Stdout, saramaLogPrefix, log.LstdFlags)

		mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
		defer closer()

		mockBroker := sarama.NewMockBroker(t, 0)
		defer mockBroker.Close()

		mockBroker.SetHandlerByMap(ira_helpers.GetHandlersMapForMockConsumer(t, mockBroker, testTopicName))

		mockConsumer, err := consumer.NewKafkaConsumer(broker.Configuration{
			Addresses: mockBroker.Addr(),
			Topic:     testTopicName,
			Enabled:   true,
		}, mockStorage, nil)
		helpers.FailOnError(t, err)

		err = mockConsumer.Close()
		helpers.FailOnError(t, err)
	}, testCaseTimeLimit)
}

func TestKafkaConsumer_SetupCleanup(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, false)
	defer closer()

	mockBroker := sarama.NewMockBroker(t, 0)
	defer mockBroker.Close()

	mockBroker.SetHandlerByMap(ira_helpers.GetHandlersMapForMockConsumer(t, mockBroker, testTopicName))

	mockConsumer, err := consumer.NewKafkaConsumer(broker.Configuration{
		Addresses: mockBroker.Addr(),
		Topic:     testTopicName,
		Enabled:   true,
	}, mockStorage, nil)
	helpers.FailOnError(t, err)

	defer func() {
		helpers.FailOnError(t, mockConsumer.Close())
	}()

	// The functions don't really use their arguments at all,
	// so it's possible to just pass nil into them.
	helpers.FailOnError(t, mockConsumer.Setup(nil))
	helpers.FailOnError(t, mockConsumer.Cleanup(nil))
}

func TestKafkaConsumer_NewDeadLetterProducer_Error(t *testing.T) {
	// Backup original functions
	originalNewDeadLetterProducer := producer.NewDeadLetterProducer
	originalNewPayloadTrackerProducer := producer.NewPayloadTrackerProducer
	defer func() {
		producer.NewDeadLetterProducer = originalNewDeadLetterProducer
		producer.NewPayloadTrackerProducer = originalNewPayloadTrackerProducer
	}()

	// Override functions for testing
	producer.NewDeadLetterProducer = func(_ broker.Configuration) (*producer.DeadLetterProducer, error) {
		return nil, errors.New("error happened")
	}

	producer.NewPayloadTrackerProducer = func(_ broker.Configuration) (*producer.PayloadTrackerProducer, error) {
		return nil, nil
	}

	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	mockBroker := sarama.NewMockBroker(t, 0)
	defer mockBroker.Close()

	mockBroker.SetHandlerByMap(ira_helpers.GetHandlersMapForMockConsumer(t, mockBroker, testTopicName))

	_, err := consumer.NewKafkaConsumer(broker.Configuration{
		Addresses: mockBroker.Addr(),
		Topic:     testTopicName,
		Enabled:   true,
	}, mockStorage, nil)

	assert.EqualError(t, err, "error happened")
}

func TestKafkaConsumer_NewPayloadTrackerProducer_Error(t *testing.T) {
	// Backup original functions
	originalNewDeadLetterProducer := producer.NewDeadLetterProducer
	originalNewPayloadTrackerProducer := producer.NewPayloadTrackerProducer
	defer func() {
		producer.NewDeadLetterProducer = originalNewDeadLetterProducer
		producer.NewPayloadTrackerProducer = originalNewPayloadTrackerProducer
	}()

	// Override functions for testing
	producer.NewDeadLetterProducer = func(_ broker.Configuration) (*producer.DeadLetterProducer, error) {
		return nil, nil
	}

	producer.NewPayloadTrackerProducer = func(_ broker.Configuration) (*producer.PayloadTrackerProducer, error) {
		return nil, errors.New("error happened")
	}

	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	mockBroker := sarama.NewMockBroker(t, 0)
	defer mockBroker.Close()

	mockBroker.SetHandlerByMap(ira_helpers.GetHandlersMapForMockConsumer(t, mockBroker, testTopicName))

	_, err := consumer.NewKafkaConsumer(broker.Configuration{
		Addresses: mockBroker.Addr(),
		Topic:     testTopicName,
		Enabled:   true,
	}, mockStorage, nil)

	assert.EqualError(t, err, "error happened")
}
