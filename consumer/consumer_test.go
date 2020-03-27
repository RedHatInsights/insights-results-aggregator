/*
Copyright © 2020 Red Hat, Inc.

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
	"testing"
	"time"

	"github.com/RedHatInsights/insights-results-aggregator/tests/testdata"

	"bou.ke/monkey"
	"github.com/Shopify/sarama"
	mapset "github.com/deckarep/golang-set"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/consumer"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const (
	testTopicName = "topic"
	// time limit for *some* tests which can stuck in forever loop
	testCaseTimeLimit = 20 * time.Second
)

var (
	testOrgWhiteList = mapset.NewSetWith(types.OrgID(1))
	wrongBrokerCfg   = broker.Configuration{
		Address: "localhost:1234",
		Topic:   "topic",
		Group:   "group",
	}
)

func TestConsumerConstructorNoKafka(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, false)
	defer helpers.MustCloseStorage(t, mockStorage)

	mockConsumer, err := consumer.New(wrongBrokerCfg, mockStorage)
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

func TestParseEmptyMessage(t *testing.T) {
	_, err := consumer.ParseMessage([]byte(""))
	assert.EqualError(t, err, "unexpected end of JSON input")
}

func TestParseMessageWithWrongContent(t *testing.T) {
	const message = `{"this":"is", "not":"expected content"}`
	_, err := consumer.ParseMessage([]byte(message))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing required attribute")
}

func TestParseMessageWithImproperJSON(t *testing.T) {
	const message = `"this_is_not_json_dude"`
	_, err := consumer.ParseMessage([]byte(message))
	assert.EqualError(
		t,
		err,
		"json: cannot unmarshal string into Go value of type consumer.incomingMessage",
	)
}

func TestParseProperMessage(t *testing.T) {
	message, err := consumer.ParseMessage([]byte(testdata.ConsumerMessage))
	helpers.FailOnError(t, err)

	assert.Equal(t, types.OrgID(1), *message.Organization)
	assert.Equal(t, testdata.ClusterName, *message.ClusterName)

	var expectedReport consumer.Report
	err = json.Unmarshal([]byte(testdata.ConsumerReport), &expectedReport)
	helpers.FailOnError(t, err)

	assert.Equal(t, expectedReport, *message.Report)
}

func TestParseProperMessageWrongClusterName(t *testing.T) {
	message := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "this is not a UUID",
		"Report": ` + testdata.ConsumerReport + `
	}`
	_, err := consumer.ParseMessage([]byte(message))
	assert.EqualError(t, err, "cluster name is not a UUID")
}

func TestParseMessageWithoutOrgID(t *testing.T) {
	message := `{
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Report": ` + testdata.ConsumerReport + `
	}`
	_, err := consumer.ParseMessage([]byte(message))
	assert.EqualError(t, err, "missing required attribute 'OrgID'")
}

func TestParseMessageWithoutClusterName(t *testing.T) {
	message := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"Report": ` + testdata.ConsumerReport + `
	}`
	_, err := consumer.ParseMessage([]byte(message))
	assert.EqualError(t, err, "missing required attribute 'ClusterName'")
}

func TestParseMessageWithoutReport(t *testing.T) {
	message := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `"
	}`
	_, err := consumer.ParseMessage([]byte(message))
	assert.EqualError(t, err, "missing required attribute 'Report'")
}

func TestParseMessageEmptyReport(t *testing.T) {
	message := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Report": {}
	}`

	_, err := consumer.ParseMessage([]byte(message))
	assert.EqualError(t, err, "Improper report structure, missing key fingerprints")
}

func TestParseMessageNullReport(t *testing.T) {
	message := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Report": null
	}`

	_, err := consumer.ParseMessage([]byte(message))
	assert.EqualError(t, err, "missing required attribute 'Report'")
}

func dummyConsumer(s storage.Storage, whitelist bool) consumer.Consumer {
	brokerCfg := broker.Configuration{
		Address: "localhost:1234",
		Topic:   "topic",
		Group:   "group",
	}
	if whitelist {
		brokerCfg.OrgWhitelist = mapset.NewSetWith(types.OrgID(1))
	}
	return &consumer.KafkaConsumer{
		Configuration:     brokerCfg,
		Consumer:          nil,
		PartitionConsumer: nil,
		Storage:           s,
	}
}

func TestProcessEmptyMessage(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	c := dummyConsumer(mockStorage, true)

	message := sarama.ConsumerMessage{}
	// message is empty -> nothing should be written into storage
	err := c.ProcessMessage(&message)
	assert.EqualError(t, err, "unexpected end of JSON input")

	count, err := mockStorage.ReportsCount()
	helpers.FailOnError(t, err)

	assert.Equal(
		t,
		0,
		count,
		"process message shouldn't write anything into the DB",
	)
}

func TestProcessCorrectMessage(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	c := dummyConsumer(mockStorage, true)

	message := sarama.ConsumerMessage{}
	message.Value = []byte(testdata.ConsumerMessage)
	// message is empty -> nothing should be written into storage
	err := c.ProcessMessage(&message)
	helpers.FailOnError(t, err)

	count, err := mockStorage.ReportsCount()
	helpers.FailOnError(t, err)

	assert.Equal(t, 1, count)
}

func consumerProcessMessage(mockConsumer consumer.Consumer, message string) error {
	saramaMessage := sarama.ConsumerMessage{}
	saramaMessage.Value = []byte(message)
	return mockConsumer.ProcessMessage(&saramaMessage)
}

func TestProcessingMessageWithClosedStorage(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)

	mockConsumer := dummyConsumer(mockStorage, true)
	helpers.MustCloseStorage(t, mockStorage)

	err := consumerProcessMessage(mockConsumer, testdata.ConsumerMessage)
	assert.EqualError(t, err, "sql: database is closed")
}

func TestProcessingMessageWithWrongDateFormat(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	mockConsumer := dummyConsumer(mockStorage, true)

	messageValue := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Report": ` + testdata.ConsumerReport + `,
		"LastChecked": "2020.01.23 16:15:59"
	}`

	err := consumerProcessMessage(mockConsumer, messageValue)
	if _, ok := err.(*time.ParseError); err == nil || !ok {
		t.Fatal(fmt.Errorf(
			"expected time.ParseError error because date format is wrong. Got %+v", err,
		))
	}
}

func TestKafkaConsumerMockOK(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		mockConsumer := helpers.MustGetMockKafkaConsumerWithExpectedMessages(
			t,
			testTopicName,
			testOrgWhiteList,
			[]string{testdata.ConsumerMessage},
		)

		go mockConsumer.Serve()

		// wait for message processing
		helpers.WaitForMockConsumerToHaveNConsumedMessages(mockConsumer, 1)

		err := mockConsumer.Close()
		helpers.FailOnError(t, err)

		assert.Equal(t, uint64(1), mockConsumer.GetNumberOfSuccessfullyConsumedMessages())
		assert.Equal(t, uint64(0), mockConsumer.GetNumberOfErrorsConsumingMessages())
	}, testCaseTimeLimit)
}

func TestKafkaConsumerMockBadMessage(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		mockConsumer := helpers.MustGetMockKafkaConsumerWithExpectedMessages(
			t,
			testTopicName,
			testOrgWhiteList,
			[]string{"bad message"},
		)

		go mockConsumer.Serve()

		// wait for message processing
		helpers.WaitForMockConsumerToHaveNConsumedMessages(mockConsumer, 1)

		err := mockConsumer.Close()
		helpers.FailOnError(t, err)

		assert.Equal(t, uint64(0), mockConsumer.GetNumberOfSuccessfullyConsumedMessages())
		assert.Equal(t, uint64(1), mockConsumer.GetNumberOfErrorsConsumingMessages())
	}, testCaseTimeLimit)
}

func TestKafkaConsumerMockWritingToClosedStorage(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		mockConsumer := helpers.MustGetMockKafkaConsumerWithExpectedMessages(
			t, testTopicName, testOrgWhiteList, []string{testdata.ConsumerMessage},
		)

		err := mockConsumer.Storage.Close()
		helpers.FailOnError(t, err)

		go mockConsumer.Serve()

		helpers.WaitForMockConsumerToHaveNConsumedMessages(mockConsumer, 1)

		err = mockConsumer.Close()
		helpers.FailOnError(t, err)

		assert.Equal(t, uint64(0), mockConsumer.GetNumberOfSuccessfullyConsumedMessages())
		assert.Equal(t, uint64(1), mockConsumer.GetNumberOfErrorsConsumingMessages())
	}, testCaseTimeLimit)
}

func TestKafkaConsumer_ProcessMessage_OrganizationIsNotAllowed(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	mockConsumer := dummyConsumer(mockStorage, false)

	err := consumerProcessMessage(mockConsumer, testdata.ConsumerMessage)
	assert.EqualError(t, err, "organization ID is not whitelisted")
}

// newConsumerWithMockBroker creates new mock consumer with mock broker
// don't forget to wrap a calling test to helpers.RunTestWithTimeout,
// because it can wait for mock broker creation forever
func newConsumerWithMockBroker(t *testing.T) (storage.Storage, *helpers.MockBroker, consumer.Consumer) {
	mockStorage := helpers.MustGetMockStorage(t, false)
	mockBroker := helpers.MustNewMockBroker(t)

	saramaConfig := sarama.NewConfig()
	saramaConfig.ChannelBufferSize = 1
	saramaConfig.Version = sarama.V0_10_0_1
	saramaConfig.Producer.Return.Successes = true

	var (
		mockConsumer consumer.Consumer
		err          error
	)

	for {
		mockConsumer, err = consumer.NewWithSaramaConfig(broker.Configuration{
			Address:      mockBroker.Address,
			Topic:        mockBroker.TopicName,
			Group:        mockBroker.Group,
			Enabled:      true,
			OrgWhitelist: nil,
		}, mockStorage, saramaConfig, false)
		// wait for topic to be created
		if kErr, ok := err.(sarama.KError); ok && kErr == sarama.ErrUnknownTopicOrPartition {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		helpers.FailOnError(t, err)
		break
	}

	assert.NotNil(t, mockConsumer)

	return mockStorage, mockBroker, mockConsumer
}

func TestKafkaConsumer_New_MockBroker(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		mockStorage, mockBroker, mockConsumer := newConsumerWithMockBroker(t)
		defer helpers.MustCloseStorage(t, mockStorage)
		defer mockBroker.MustClose(t)
		defer helpers.MustCloseConsumer(t, mockConsumer)
	}, testCaseTimeLimit)
}

func TestMonkeyPatch(t *testing.T) {
	guard := monkey.Patch(sarama.NewClient, helpers.NewMockClient)
	defer guard.Unpatch()

	consumer, err := consumer.NewWithSaramaConfig(broker.Configuration{}, nil, nil, false)

	if consumer != nil || err == nil {
		t.Fatal("WTF!")
	}
}

func TestKafkaConsumer_New_MockBrokerNoTopicError(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		mockStorage := helpers.MustGetMockStorage(t, false)
		defer helpers.MustCloseStorage(t, mockStorage)

		// pass empty string instead of topic name
		mockBroker, err := helpers.NewMockBroker(t, "")
		helpers.FailOnError(t, err)
		defer mockBroker.MustClose(t)

		saramaConfig := sarama.NewConfig()
		saramaConfig.ChannelBufferSize = 1
		saramaConfig.Version = sarama.V0_10_0_1
		saramaConfig.Producer.Return.Successes = true

		mockConsumer, err := consumer.NewWithSaramaConfig(broker.Configuration{
			Address:      mockBroker.Address,
			Topic:        mockBroker.TopicName,
			Group:        mockBroker.Group,
			Enabled:      true,
			OrgWhitelist: nil,
		}, mockStorage, saramaConfig, false)
		assert.EqualError(
			t,
			err,
			"kafka server: The request attempted to perform an operation on an invalid topic.",
		)
		assert.Nil(t, mockConsumer)
	}, testCaseTimeLimit)
}

func TestKafkaConsumer_Close_MockBrokerError(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		mockStorage, mockBroker, mockConsumer := newConsumerWithMockBroker(t)
		defer helpers.MustCloseStorage(t, mockStorage)
		defer mockBroker.MustClose(t)
		helpers.MustCloseConsumer(t, mockConsumer)

		// closing it second time should cause an error
		err := mockConsumer.Close()
		assert.EqualError(t, err, "kafka: tried to use a client that was closed")
	}, testCaseTimeLimit)
}
