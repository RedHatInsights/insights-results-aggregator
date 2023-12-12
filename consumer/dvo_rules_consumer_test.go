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
		Address: "localhost:1234",
		Topic:   "topic",
		Group:   "group",
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

		mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
		defer closer()

		mockBroker := sarama.NewMockBroker(t, 0)
		defer mockBroker.Close()

		mockBroker.SetHandlerByMap(ira_helpers.GetHandlersMapForMockConsumer(t, mockBroker, testTopicName))

		mockConsumer, err := consumer.NewDVORulesConsumer(broker.Configuration{
			Address: mockBroker.Addr(),
			Topic:   testTopicName,
			Enabled: true,
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
		"Metrics": {
			"we_dont_know_yet": "what_format_it_will_have",
            "but": "this should be deserialized properly"
		}
	}`
	c := consumer.KafkaConsumer{MessageProcessor: consumer.DVORulesProcessor{}}
	message, err := consumer.DeserializeMessage(&c, []byte(consumerMessage))
	helpers.FailOnError(t, err)
	assert.Equal(t, types.OrgID(1), *message.Organization)
	assert.Equal(t, testdata.ClusterName, *message.ClusterName)
}

//TODO: We don't know yet what a proper message is
//func TestDeserializeDVOProperMessage(t *testing.T) {}

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

//TODO: We don't know yet how we will handle "empty" metrics situations
//func TestParseDVOMessageEmptyMetrics(t *testing.T) {}

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

//TODO: parseMessage only deserializes the message for now.
//func TestParseMessageWithImproperMetrics(t *testing.T) {}

//func TestProcessEmptyMessage(t *testing.T) {
//	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
//	defer closer()
//
//	c := dummyDVOConsumer(mockStorage, true)
//
//	message := sarama.ConsumerMessage{}
//	// message is empty -> nothing should be written into storage
//	err := c.HandleMessage(&message)
//	assert.EqualError(t, err, "unexpected end of JSON input")
//
//	count, err := mockStorage.ReportsCount()
//	helpers.FailOnError(t, err)
//
//	// no record should be written into database
//	assert.Equal(
//		t,
//		0,
//		count,
//		"process message shouldn't write anything into the DB",
//	)
//}
//
//func TestProcessCorrectMessage(t *testing.T) {
//	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
//
//	defer closer()
//
//	c := dummyDVOConsumer(mockStorage, true)
//
//	message := sarama.ConsumerMessage{}
//	message.Value = []byte(messageReportWithRuleHits)
//	// message is correct -> one record should be written into storage
//	err := c.HandleMessage(&message)
//	helpers.FailOnError(t, err)
//
//	count, err := mockStorage.ReportsCount()
//	helpers.FailOnError(t, err)
//
//	// exactly one record should be written into database
//	assert.Equal(t, 1, count, "process message should write one record into DB")
//}
//
//func TestProcessingEmptyReportMissingAttributesWithClosedStorage(t *testing.T) {
//	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
//
//	mockConsumer := dummyDVOConsumer(mockStorage, true)
//	closer()
//
//	err := consumerProcessMessage(mockConsumer, messageNoReportsNoInfo)
//	helpers.FailOnError(t, err, "empty report should not be considered an error at HandleMessage level")
//}
//
//func TestProcessingValidMessageEmptyReportWithRequiredAttributesWithClosedStorage(t *testing.T) {
//	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
//
//	mockConsumer := dummyDVOConsumer(mockStorage, true)
//	closer()
//
//	err := consumerProcessMessage(mockConsumer, testdata.ConsumerMessage)
//	assert.EqualError(t, err, "sql: database is closed")
//}
//
//func TestProcessingCorrectMessageWithClosedStorage(t *testing.T) {
//	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
//
//	mockConsumer := dummyDVOConsumer(mockStorage, true)
//	closer()
//
//	err := consumerProcessMessage(mockConsumer, messageReportWithRuleHits)
//	assert.EqualError(t, err, "sql: database is closed")
//}
//
//func TestProcessingMessageWithWrongDateFormatAndEmptyReport(t *testing.T) {
//	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
//	defer closer()
//
//	mockConsumer := dummyDVOConsumer(mockStorage, true)
//
//	err := consumerProcessMessage(mockConsumer, messageNoReportsNoInfo)
//	assert.Nil(t, err, "Message with empty report should not be processed")
//}
//
//func TestProcessingMessageWithWrongDateFormatReportNotEmpty(t *testing.T) {
//	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
//	defer closer()
//
//	mockConsumer := dummyDVOConsumer(mockStorage, true)
//
//	messageValue := `{
//		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
//		"ClusterName": "` + string(testdata.ClusterName) + `",
//		"Report":` + testReport + `,
//		"LastChecked": "2020.01.23 16:15:59"
//	}`
//	err := consumerProcessMessage(mockConsumer, messageValue)
//	if _, ok := err.(*time.ParseError); err == nil || !ok {
//		t.Fatal(fmt.Errorf(
//			"expected time.ParseError error because date format is wrong. Got %+v", err,
//		))
//	}
//}
//
//func TestKafkaConsumerMockOK(t *testing.T) {
//	helpers.RunTestWithTimeout(t, func(t testing.TB) {
//		mockConsumer, closer := ira_helpers.MustGetMockDVORulesConsumerWithExpectedMessages(
//			t,
//			testTopicName,
//			testOrgAllowlist,
//			[]string{messageReportWithRuleHits},
//		)
//
//		go mockConsumer.Serve()
//
//		// wait for message processing
//		ira_helpers.WaitForMockConsumerToHaveNConsumedMessages(mockConsumer, 1)
//
//		closer()
//
//		assert.Equal(t, uint64(1), mockConsumer.KafkaConsumer.GetNumberOfSuccessfullyConsumedMessages())
//		assert.Equal(t, uint64(0), mockConsumer.KafkaConsumer.GetNumberOfErrorsConsumingMessages())
//	}, testCaseTimeLimit)
//}
//
//func TestKafkaConsumerMockBadMessage(t *testing.T) {
//	helpers.RunTestWithTimeout(t, func(t testing.TB) {
//		mockConsumer, closer := ira_helpers.MustGetMockDVORulesConsumerWithExpectedMessages(
//			t,
//			testTopicName,
//			testOrgAllowlist,
//			[]string{"bad message"},
//		)
//
//		go mockConsumer.Serve()
//
//		// wait for message processing
//		ira_helpers.WaitForMockConsumerToHaveNConsumedMessages(mockConsumer, 1)
//
//		closer()
//
//		assert.Equal(t, uint64(0), mockConsumer.KafkaConsumer.GetNumberOfSuccessfullyConsumedMessages())
//		assert.Equal(t, uint64(1), mockConsumer.KafkaConsumer.GetNumberOfErrorsConsumingMessages())
//	}, testCaseTimeLimit)
//}
//
//func TestKafkaConsumerMockWritingMsgWithEmptyReportToClosedStorage(t *testing.T) {
//	helpers.RunTestWithTimeout(t, func(t testing.TB) {
//		mockConsumer, closer := ira_helpers.MustGetMockDVORulesConsumerWithExpectedMessages(
//			t, testTopicName, testOrgAllowlist, []string{messageNoReportsNoInfo},
//		)
//
//		err := mockConsumer.KafkaConsumer.Storage.Close()
//		helpers.FailOnError(t, err)
//
//		go mockConsumer.Serve()
//
//		ira_helpers.WaitForMockConsumerToHaveNConsumedMessages(mockConsumer, 1)
//
//		closer()
//
//		// Since the report is present but empty, we stop processing this message without errors
//		assert.Equal(t, uint64(1), mockConsumer.KafkaConsumer.GetNumberOfSuccessfullyConsumedMessages())
//		assert.Equal(t, uint64(0), mockConsumer.KafkaConsumer.GetNumberOfErrorsConsumingMessages())
//	}, testCaseTimeLimit)
//}
//
//func TestKafkaConsumerMockWritingMsgWithReportToClosedStorage(t *testing.T) {
//	helpers.RunTestWithTimeout(t, func(t testing.TB) {
//		mockConsumer, closer := ira_helpers.MustGetMockDVORulesConsumerWithExpectedMessages(
//			t, testTopicName, testOrgAllowlist, []string{messageReportWithRuleHits},
//		)
//
//		err := mockConsumer.KafkaConsumer.Storage.Close()
//		helpers.FailOnError(t, err)
//
//		go mockConsumer.Serve()
//
//		ira_helpers.WaitForMockConsumerToHaveNConsumedMessages(mockConsumer, 1)
//
//		closer()
//
//		// Since the report is present and not empty, it is processed, and we reach the closed DB error
//		assert.Equal(t, uint64(0), mockConsumer.KafkaConsumer.GetNumberOfSuccessfullyConsumedMessages())
//		assert.Equal(t, uint64(1), mockConsumer.KafkaConsumer.GetNumberOfErrorsConsumingMessages())
//	}, testCaseTimeLimit)
//}
//
//func TestKafkaConsumer_ProcessMessage_OrganizationAllowlistDisabled(t *testing.T) {
//	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
//	defer closer()
//
//	mockConsumer := dummyDVOConsumer(mockStorage, false)
//
//	err := consumerProcessMessage(mockConsumer, messageReportWithRuleHits)
//	helpers.FailOnError(t, err)
//}
//
//func TestKafkaConsumer_ProcessMessageWithEmptyReport_OrganizationIsNotAllowed(t *testing.T) {
//	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
//	defer closer()
//
//	brokerCfg := broker.Configuration{
//		Address:             "localhost:1234",
//		Topic:               "topic",
//		Group:               "group",
//		OrgAllowlist:        mapset.NewSetWith(types.OrgID(123)), // in testdata, OrgID = 1
//		OrgAllowlistEnabled: true,
//	}
//	mockConsumer := createDVOConsumer(brokerCfg, mockStorage)
//
//	err := consumerProcessMessage(mockConsumer, messageNoReportsNoInfo)
//	helpers.FailOnError(t, err, "message have empty report and should not be processed")
//}
//
//func TestKafkaConsumer_ProcessMessage_OrganizationIsNotAllowed(t *testing.T) {
//	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
//	defer closer()
//
//	brokerCfg := broker.Configuration{
//		Address:             "localhost:1234",
//		Topic:               "topic",
//		Group:               "group",
//		OrgAllowlist:        mapset.NewSetWith(types.OrgID(123)), // in testdata, OrgID = 1
//		OrgAllowlistEnabled: true,
//	}
//	mockConsumer := createDVOConsumer(brokerCfg, mockStorage)
//
//	err := consumerProcessMessage(mockConsumer, messageReportWithRuleHits)
//	assert.EqualError(t, err, organizationIDNotInAllowList)
//}
//
//func TestKafkaConsumer_ProcessMessageWithEmptyReport_OrganizationBadConfigIsNotAllowed(t *testing.T) {
//	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
//	defer closer()
//
//	brokerCfg := broker.Configuration{
//		Address:             "localhost:1234",
//		Topic:               "topic",
//		Group:               "group",
//		OrgAllowlist:        nil,
//		OrgAllowlistEnabled: true,
//	}
//	mockConsumer := createDVOConsumer(brokerCfg, mockStorage)
//
//	err := consumerProcessMessage(mockConsumer, messageNoReportsNoInfo)
//	helpers.FailOnError(t, err, "message have empty report and should not be processed")
//}
//
//func TestKafkaConsumer_ProcessMessage_OrganizationBadConfigIsNotAllowed(t *testing.T) {
//	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
//	defer closer()
//
//	brokerCfg := broker.Configuration{
//		Address:             "localhost:1234",
//		Topic:               "topic",
//		Group:               "group",
//		OrgAllowlist:        nil,
//		OrgAllowlistEnabled: true,
//	}
//	mockConsumer := createDVOConsumer(brokerCfg, mockStorage)
//
//	err := consumerProcessMessage(mockConsumer, messageReportWithRuleHits)
//	assert.EqualError(t, err, organizationIDNotInAllowList)
//}
//
//func TestKafkaConsumer_ProcessMessage_MessageFromTheFuture(t *testing.T) {
//	buf := new(bytes.Buffer)
//	zerolog_log.Logger = zerolog.New(buf)
//
//	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
//	defer closer()
//
//	mockConsumer := createDVOConsumer(wrongBrokerCfg, mockStorage)
//
//	message := `{
//		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
//		"ClusterName": "` + string(testdata.ClusterName) + `",
//		"Report":` + testReport + `,
//		"LastChecked": "` + time.Now().Add(24*time.Hour).Format(time.RFC3339) + `"
//	}`
//
//	err := consumerProcessMessage(mockConsumer, message)
//	helpers.FailOnError(t, err)
//	assert.Contains(t, buf.String(), "got a message from the future")
//}
//
//func TestKafkaConsumer_ProcessMessage_MoreRecentReportAlreadyExists(t *testing.T) {
//	zerolog.SetGlobalLevel(zerolog.InfoLevel)
//	buf := new(bytes.Buffer)
//	zerolog_log.Logger = zerolog.New(buf)
//
//	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
//	defer closer()
//
//	mockConsumer := createDVOConsumer(wrongBrokerCfg, mockStorage)
//
//	message := `{
//		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
//		"ClusterName": "` + string(testdata.ClusterName) + `",
//		"Report":` + testReport + `,
//		"LastChecked": "` + time.Now().Format(time.RFC3339) + `"
//	}`
//
//	err := consumerProcessMessage(mockConsumer, message)
//	helpers.FailOnError(t, err)
//
//	message = `{
//		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
//		"ClusterName": "` + string(testdata.ClusterName) + `",
//		"Report":` + testReport + `,
//		"LastChecked": "` + time.Now().Add(-24*time.Hour).Format(time.RFC3339) + `"
//	}`
//
//	err = consumerProcessMessage(mockConsumer, message)
//	helpers.FailOnError(t, err)
//
//	assert.Contains(t, buf.String(), "Skipping because a more recent report already exists for this cluster")
//}
//
//func TestKafkaConsumer_ProcessMessage_MessageWithNoSchemaVersion(t *testing.T) {
//	buf := new(bytes.Buffer)
//	zerolog_log.Logger = zerolog.New(buf)
//
//	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
//	defer closer()
//
//	mockConsumer := createDVOConsumer(wrongBrokerCfg, mockStorage)
//
//	err := consumerProcessMessage(mockConsumer, messageReportWithRuleHits)
//	helpers.FailOnError(t, err)
//	assert.Contains(t, buf.String(), "\"level\":\"warn\"")
//	assert.Contains(t, buf.String(), "Received data with unexpected version")
//}
//
//func TestKafkaConsumer_ProcessMessage_MessageWithUnexpectedSchemaVersion(t *testing.T) {
//	buf := new(bytes.Buffer)
//	zerolog_log.Logger = zerolog.New(buf)
//
//	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
//	defer closer()
//
//	mockConsumer := createDVOConsumer(wrongBrokerCfg, mockStorage)
//
//	message := `{
//		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
//		"ClusterName": "` + string(testdata.ClusterName) + `",
//		"Report":` + testReport + `,
//		"LastChecked": "` + time.Now().Add(-24*time.Hour).Format(time.RFC3339) + `",
//		"Version": ` + fmt.Sprintf("%d", types.SchemaVersion(3)) + `
//	}`
//
//	err := consumerProcessMessage(mockConsumer, message)
//	helpers.FailOnError(t, err)
//	assert.Contains(t, buf.String(), "\"level\":\"warn\"")
//	assert.Contains(t, buf.String(), "Received data with unexpected version")
//}
//
//func TestKafkaConsumer_ProcessMessage_MessageWithExpectedSchemaVersion(t *testing.T) {
//	buf := new(bytes.Buffer)
//	zerolog_log.Logger = zerolog.New(buf)
//
//	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
//	defer closer()
//
//	mockConsumer := createDVOConsumer(wrongBrokerCfg, mockStorage)
//
//	message := `{
//		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
//		"ClusterName": "` + string(testdata.ClusterName) + `",
//		"Report":` + testReport + `,
//		"LastChecked": "` + time.Now().Add(-24*time.Hour).Format(time.RFC3339) + `",
//		"Version": ` + fmt.Sprintf("%d", types.SchemaVersion(1)) + `
//	}`
//
//	err := consumerProcessMessage(mockConsumer, message)
//	helpers.FailOnError(t, err)
//
//	message = `{
//		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
//		"ClusterName": "` + string(testdata.ClusterName) + `",
//		"Report":` + testReport + `,
//		"LastChecked": "` + time.Now().Add(-24*time.Hour).Format(time.RFC3339) + `",
//		"Version": ` + fmt.Sprintf("%d", types.SchemaVersion(2)) + `
//	}`
//
//	err = consumerProcessMessage(mockConsumer, message)
//	helpers.FailOnError(t, err)
//
//	assert.NotContains(t, buf.String(), "\"level\":\"warn\"")
//	assert.NotContains(t, buf.String(), "Received data with unexpected version")
//}
//
//func TestKafkaConsumer_ConsumeClaim(t *testing.T) {
//	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
//	defer closer()
//
//	kafkaConsumer := createDVOConsumer(broker.Configuration{}, mockStorage)
//
//	mockConsumerGroupSession := &saramahelpers.MockConsumerGroupSession{}
//	mockConsumerGroupClaim := saramahelpers.NewMockConsumerGroupClaim(nil)
//
//	err := kafkaConsumer.ConsumeClaim(mockConsumerGroupSession, mockConsumerGroupClaim)
//	helpers.FailOnError(t, err)
//}
//
//func TestKafkaConsumer_ConsumeClaim_DBError(t *testing.T) {
//	buf := new(bytes.Buffer)
//	zerolog_log.Logger = zerolog.New(buf)
//
//	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
//	closer()
//
//	kafkaConsumer := createDVOConsumer(broker.Configuration{}, mockStorage)
//
//	mockConsumerGroupSession := &saramahelpers.MockConsumerGroupSession{}
//	mockConsumerGroupClaim := saramahelpers.NewMockConsumerGroupClaim(nil)
//
//	err := kafkaConsumer.ConsumeClaim(mockConsumerGroupSession, mockConsumerGroupClaim)
//	helpers.FailOnError(t, err)
//
//	assert.Contains(t, buf.String(), "starting messages loop")
//}
//
//func TestKafkaConsumer_ConsumeClaim_OKMessage(t *testing.T) {
//	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
//	defer closer()
//
//	kafkaConsumer := createDVOConsumer(broker.Configuration{}, mockStorage)
//
//	mockConsumerGroupSession := &saramahelpers.MockConsumerGroupSession{}
//	mockConsumerGroupClaim := saramahelpers.NewMockConsumerGroupClaim([]*sarama.ConsumerMessage{
//		saramahelpers.StringToSaramaConsumerMessage(testdata.ConsumerMessage),
//	})
//
//	err := kafkaConsumer.ConsumeClaim(mockConsumerGroupSession, mockConsumerGroupClaim)
//	helpers.FailOnError(t, err)
//}
