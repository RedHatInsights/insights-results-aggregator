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
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/RedHatInsights/insights-results-aggregator/storage"

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/Shopify/sarama"
	mapset "github.com/deckarep/golang-set"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/consumer"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
}

var (
	dvoConsumer = consumer.KafkaConsumer{
		MessageProcessor: consumer.DVORulesProcessor{},
	}
)

func createDVOConsumer(brokerCfg broker.Configuration, mockStorage storage.DVORecommendationsStorage) *consumer.KafkaConsumer {
	return &consumer.KafkaConsumer{
		Configuration:    brokerCfg,
		Storage:          mockStorage,
		MessageProcessor: consumer.DVORulesProcessor{},
	}
}

func dummyDVOConsumer(s storage.DVORecommendationsStorage, allowlist bool) consumer.Consumer {
	brokerCfg := broker.Configuration{
		Addresses: "localhost:1234",
		Topic:     "topic",
		Group:     "group",
	}
	if allowlist {
		brokerCfg.OrgAllowlist = mapset.NewSetWith(types.OrgID(1))
		brokerCfg.OrgAllowlistEnabled = true
	} else {
		brokerCfg.OrgAllowlistEnabled = false
	}
	return createDVOConsumer(brokerCfg, s)
}

func TestDVORulesConsumer_New(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		sarama.Logger = log.New(os.Stdout, saramaLogPrefix, log.LstdFlags)

		mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)
		defer closer()

		mockBroker := sarama.NewMockBroker(t, 0)
		defer mockBroker.Close()

		mockBroker.SetHandlerByMap(ira_helpers.GetHandlersMapForMockConsumer(t, mockBroker, testTopicName))

		mockConsumer, err := consumer.NewDVORulesConsumer(broker.Configuration{
			Addresses: mockBroker.Addr(),
			Topic:     testTopicName,
			Enabled:   true,
		}, mockStorage)
		helpers.FailOnError(t, err)

		err = mockConsumer.Close()
		helpers.FailOnError(t, err)
	}, testCaseTimeLimit)
}

func TestDeserializeEmptyDVOMessage(t *testing.T) {
	c := consumer.KafkaConsumer{MessageProcessor: consumer.DVORulesProcessor{}}
	_, err := consumer.DeserializeMessage(&c, []byte(""))
	assert.EqualError(t, err, "unexpected end of JSON input")
}

func TestDeserializeDVOMessageWithWrongContent(t *testing.T) {
	const message = `{"this":"is", "not":"expected content"}`
	c := consumer.KafkaConsumer{MessageProcessor: consumer.DVORulesProcessor{}}
	_, err := consumer.DeserializeMessage(&c, []byte(message))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required attribute")
}

func TestDeserializeDVOMessageWithImproperJSON(t *testing.T) {
	const message = `"this_is_not_json_dude"`
	c := consumer.KafkaConsumer{MessageProcessor: consumer.DVORulesProcessor{}}
	_, err := consumer.DeserializeMessage(&c, []byte(message))
	assert.EqualError(
		t,
		err,
		"json: cannot unmarshal string into Go value of type consumer.incomingMessage",
	)
}

func TestDeserializeDVOMessageWithImproperMetrics(t *testing.T) {
	consumerMessage := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"LastChecked": "` + testdata.LastCheckedAt.Format(time.RFC3339) + `",
		"Metrics": "this is not a JSON"
	}`
	c := consumer.KafkaConsumer{MessageProcessor: consumer.DVORulesProcessor{}}
	_, err := consumer.DeserializeMessage(&c, []byte(consumerMessage))
	assert.EqualError(
		t,
		err,
		"json: cannot unmarshal string into Go struct field incomingMessage.Metrics of type consumer.DvoMetrics",
	)
}

func TestDeserializeDVOProperMessage(t *testing.T) {
	consumerMessage := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"LastChecked": "` + testdata.LastCheckedAt.Format(time.RFC3339) + `",
		"Metrics":` + testMetrics + `
	}`
	c := consumer.KafkaConsumer{MessageProcessor: consumer.DVORulesProcessor{}}
	message, err := consumer.DeserializeMessage(&c, []byte(consumerMessage))
	helpers.FailOnError(t, err)
	assert.Equal(t, types.OrgID(1), *message.Organization)
	assert.Equal(t, testdata.ClusterName, *message.ClusterName)
}

func TestDeserializeDVOMessageWrongClusterName(t *testing.T) {
	message := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "this is not a UUID"
	}`
	c := consumer.KafkaConsumer{MessageProcessor: consumer.DVORulesProcessor{}}
	_, err := consumer.DeserializeMessage(&c, []byte(message))
	assert.EqualError(t, err, "cluster name is not a UUID")
}

func TestDeserializeDVOMessageWithoutOrgID(t *testing.T) {
	message := `{
		"ClusterName": "` + string(testdata.ClusterName) + `"
	}`
	c := consumer.KafkaConsumer{MessageProcessor: consumer.DVORulesProcessor{}}
	_, err := consumer.DeserializeMessage(&c, []byte(message))
	assert.EqualError(t, err, "missing required attribute 'OrgID'")
}

func TestDeserializeDVOMessageWithoutClusterName(t *testing.T) {
	message := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `
	}`
	c := consumer.KafkaConsumer{MessageProcessor: consumer.DVORulesProcessor{}}
	_, err := consumer.DeserializeMessage(&c, []byte(message))
	assert.EqualError(t, err, "missing required attribute 'ClusterName'")
}

func TestDeserializeDVOMessageWithoutMetrics(t *testing.T) {
	message := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `"
	}`
	c := consumer.KafkaConsumer{MessageProcessor: consumer.DVORulesProcessor{}}
	_, err := consumer.DeserializeMessage(&c, []byte(message))
	assert.EqualError(t, err, "missing required attribute 'Metrics'")
}

func TestDeserializeDVOMessageWithEmptyReport(t *testing.T) {
	message := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Metrics": {}
	}`
	c := consumer.KafkaConsumer{MessageProcessor: consumer.DVORulesProcessor{}}
	_, err := consumer.DeserializeMessage(&c, []byte(message))
	assert.Nil(t, err, "deserializeMessage should not return error for empty metrics")
}

func TestDeserializeDVOMessageNullMetrics(t *testing.T) {
	message := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Metrics": null
	}`
	c := consumer.KafkaConsumer{MessageProcessor: consumer.DVORulesProcessor{}}
	_, err := consumer.DeserializeMessage(&c, []byte(message))
	assert.EqualError(t, err, "missing required attribute 'Metrics'")
}

func TestDeserializeCompressedDVOMessage(t *testing.T) {
	consumerMessage := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"LastChecked": "` + testdata.LastCheckedAt.Format(time.RFC3339) + `",
		"Metrics": {
			"this_is_not": "a_proper_format",
			"but": "this should be deserialized properly"
		}
	}`
	compressed := compressConsumerMessage([]byte(consumerMessage))
	c := consumer.KafkaConsumer{MessageProcessor: consumer.DVORulesProcessor{}}
	message, err := consumer.DeserializeMessage(&c, compressed)
	helpers.FailOnError(t, err)
	assert.Equal(t, types.OrgID(1), *message.Organization)
	assert.Equal(t, testdata.ClusterName, *message.ClusterName)
}

func TestParseEmptyDVOMessage(t *testing.T) {
	message := sarama.ConsumerMessage{}
	_, err := consumer.ParseMessage(&dvoConsumer, &message)
	assert.EqualError(t, err, "unexpected end of JSON input")
}

func TestParseDVOMessageWithWrongContent(t *testing.T) {
	message := sarama.ConsumerMessage{Value: []byte(`{"this":"is", "not":"expected content"}`)}
	_, err := consumer.ParseMessage(&dvoConsumer, &message)
	assert.EqualError(t, err, "missing required attribute 'OrgID'")
}

func TestParseProperDVOMessageWrongClusterName(t *testing.T) {
	data := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "this is not a UUID",
		"Metrics": {}
	}`
	message := sarama.ConsumerMessage{Value: []byte(data)}
	_, err := consumer.ParseMessage(&dvoConsumer, &message)
	assert.EqualError(t, err, "cluster name is not a UUID")
}

func TestParseDVOMessageWithoutOrgID(t *testing.T) {
	data := `{
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Metrics": {}
	}`
	message := sarama.ConsumerMessage{Value: []byte(data)}
	_, err := consumer.ParseMessage(&dvoConsumer, &message)
	assert.EqualError(t, err, "missing required attribute 'OrgID'")
}

func TestParseDVOMessageWithoutClusterName(t *testing.T) {
	data := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"Metrics": {}
	}`
	message := sarama.ConsumerMessage{Value: []byte(data)}
	_, err := consumer.ParseMessage(&dvoConsumer, &message)
	assert.EqualError(t, err, "missing required attribute 'ClusterName'")
}

func TestParseMessageWithoutMetrics(t *testing.T) {
	data := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `"
	}`
	message := sarama.ConsumerMessage{Value: []byte(data)}
	_, err := consumer.ParseMessage(&dvoConsumer, &message)
	assert.EqualError(t, err, "missing required attribute 'Metrics'")
}

func TestParseDVOMessageEmptyMetrics(t *testing.T) {
	data := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Metrics": {}
	}`
	message := sarama.ConsumerMessage{Value: []byte(data)}
	_, err := consumer.ParseMessage(&dvoConsumer, &message)
	assert.Equal(t, types.ErrEmptyReport, err)
}

func TestParseDVOMessageNullMetrics(t *testing.T) {
	data := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Metrics": null
	}`
	message := sarama.ConsumerMessage{Value: []byte(data)}
	_, err := consumer.ParseMessage(&dvoConsumer, &message)
	assert.EqualError(t, err, "missing required attribute 'Metrics'")
}

func TestParseDVOMessageWithImproperJSON(t *testing.T) {
	message := sarama.ConsumerMessage{Value: []byte(`"this_is_not_json_dude"`)}
	_, err := consumer.ParseMessage(&dvoConsumer, &message)
	assert.EqualError(t, err, "json: cannot unmarshal string into Go value of type consumer.incomingMessage")
}

func TestParseDVOMessageWithImproperMetrics(t *testing.T) {
	data := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Metrics": "this is not a JSON"
	}`
	message := sarama.ConsumerMessage{Value: []byte(data)}
	_, err := consumer.ParseMessage(&dvoConsumer, &message)
	assert.EqualError(t, err, "json: cannot unmarshal string into Go struct field incomingMessage.Metrics of type consumer.DvoMetrics")
}

func TestParseDVOMessageWithProperMetrics(t *testing.T) {
	data := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"LastChecked": "` + testdata.LastCheckedAt.Format(time.RFC3339) + `",
		"Metrics":` + testMetrics + `
	}`
	message := sarama.ConsumerMessage{Value: []byte(data)}
	parsed, err := consumer.ParseMessage(&dvoConsumer, &message)
	helpers.FailOnError(t, err)
	assert.Equal(t, types.OrgID(1), *parsed.Organization)
	assert.Equal(t, testdata.ClusterName, *parsed.ClusterName)

	var expectedDvoMetrics consumer.DvoMetrics
	err = json.Unmarshal([]byte(testMetrics), &expectedDvoMetrics)
	helpers.FailOnError(t, err)
	assert.Equal(t, expectedDvoMetrics, *parsed.DvoMetrics)

	expectedWorkloads := []types.WorkloadRecommendation{
		{
			ResponseID: "an_issue|DVO_AN_ISSUE",
			Component:  "ccx_rules_ocp.external.dvo.an_issue_pod.recommendation",
			Key:        "DVO_AN_ISSUE",
			Links: types.DVOLinks{
				Jira:                 []string{"https://issues.redhat.com/browse/AN_ISSUE"},
				ProductDocumentation: []string{},
			},
			Details: map[string]interface{}{
				"check_name": "",
				"check_url":  "",
				"samples": []map[string]interface{}{
					{"namespace_uid": "193a2099-1234-5678-916a-d570c9aac158", "kind": "Deployment", "uid": "0501e150-1234-5678-907f-ee732c25044a"},
					{"namespace_uid": "337477af-1234-5678-b258-16f19d8a6289", "kind": "Deployment", "uid": "8c534861-1234-5678-9af5-913de71a545b"},
				},
			},
			Tags: []string{},
			Workloads: []types.DVOWorkload{
				{
					Namespace:    "namespace-name-A",
					NamespaceUID: "NAMESPACE-UID-A",
					Kind:         "DaemonSet",
					Name:         "test-name-0099",
					UID:          "UID-0099",
				},
			},
		},
	}
	assert.EqualValues(t, expectedWorkloads, parsed.ParsedWorkloads)
}

func TestProcessEmptyDVOMessage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)
	defer closer()

	c := dummyDVOConsumer(mockStorage, true)

	message := sarama.ConsumerMessage{}
	// message is empty -> nothing should be written into storage
	err := c.HandleMessage(&message)
	assert.EqualError(t, err, "unexpected end of JSON input")

	count, err := mockStorage.ReportsCount()
	helpers.FailOnError(t, err)

	// no record should be written into database
	assert.Equal(
		t,
		0,
		count,
		"process message shouldn't write anything into the DB",
	)
}

func TestProcessCorrectDVOMessage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)

	defer closer()

	c := dummyDVOConsumer(mockStorage, true)

	message := sarama.ConsumerMessage{}
	message.Value = []byte(messageReportWithDVOHits)
	// message is correct -> one record should be written into storage
	err := c.HandleMessage(&message)
	helpers.FailOnError(t, err)

	count, err := mockStorage.ReportsCount()
	helpers.FailOnError(t, err)

	// exactly one record should be written into database
	assert.Equal(t, 1, count, "process message should write one record into DB")
}

func TestProcessingEmptyMetricsMissingAttributesWithClosedStorage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)

	mockConsumer := dummyDVOConsumer(mockStorage, true)
	closer()

	err := consumerProcessMessage(mockConsumer, messageReportNoDVOMetrics)
	helpers.FailOnError(t, err, "empty report should not be considered an error at HandleMessage level")
}

func TestProcessingCorrectDVOMessageWithClosedStorage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)

	mockConsumer := dummyDVOConsumer(mockStorage, true)
	closer()

	err := consumerProcessMessage(mockConsumer, messageReportWithDVOHits)
	assert.EqualError(t, err, "sql: database is closed")
}

func TestProcessingDVOMessageWithWrongDateFormatReportNotEmpty(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)
	defer closer()

	mockConsumer := dummyDVOConsumer(mockStorage, true)

	messageValue := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Metrics":` + testMetrics + `,
		"LastChecked": "2020.01.23 16:15:59"
	}`
	err := consumerProcessMessage(mockConsumer, messageValue)
	if _, ok := err.(*time.ParseError); err == nil || !ok {
		t.Fatal(fmt.Errorf(
			"expected time.ParseError error because date format is wrong. Got %+v", err,
		))
	}
}

func TestDVOKafkaConsumerMockOK(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		mockConsumer, closer := ira_helpers.MustGetMockDVOConsumerWithExpectedMessages(
			t,
			testTopicName,
			testOrgAllowlist,
			[]string{messageReportWithDVOHits},
		)

		go mockConsumer.Serve()

		// wait for message processing
		ira_helpers.WaitForMockConsumerToHaveNConsumedMessages(mockConsumer, 1)

		closer()

		assert.Equal(t, uint64(1), mockConsumer.KafkaConsumer.GetNumberOfSuccessfullyConsumedMessages())
		assert.Equal(t, uint64(0), mockConsumer.KafkaConsumer.GetNumberOfErrorsConsumingMessages())
	}, testCaseTimeLimit)
}
