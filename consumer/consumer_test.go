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
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/Shopify/sarama"
	mapset "github.com/deckarep/golang-set"
	"github.com/rs/zerolog"
	zerolog_log "github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/consumer"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const (
	testTopicName = "topic"
	// time limit for *some* tests which can stuck in forever loop
	testCaseTimeLimit = 60 * time.Second
	saramaLogPrefix   = "[sarama]"
)

var (
	testOrgAllowlist = mapset.NewSetWith(types.OrgID(1))
	wrongBrokerCfg   = broker.Configuration{
		Address: "localhost:1234",
		Topic:   "topic",
		Group:   "group",
	}
)

func init() {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
}

func consumerProcessMessage(mockConsumer consumer.Consumer, message string) error {
	saramaMessage := sarama.ConsumerMessage{}
	saramaMessage.Value = []byte(message)
	_, err := mockConsumer.ProcessMessage(&saramaMessage)
	return err
}

func mustConsumerProcessMessage(t testing.TB, mockConsumer consumer.Consumer, message string) {
	helpers.FailOnError(t, consumerProcessMessage(mockConsumer, message))
}

func TestConsumerConstructorNoKafka(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, false)
	defer closer()

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

func TestParseMessageWithImproperReport(t *testing.T) {
	message := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"LastChecked": "` + testdata.LastCheckedAt.Format(time.RFC3339) + `",
		"Report": {
			"system": {
				"metadata": {},
				"hostname": null
			},
			"reports": "blablablabla",
			"fingerprints": [],
			"skips": [],
			"info": []
	}
}`
	_, err := consumer.ParseMessage([]byte(message))
	assert.EqualError(
		t,
		err,
		"json: cannot unmarshal string into Go value of type []types.ReportItem",
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
	assert.EqualValues(t, []types.ReportItem{}, message.ParsedHits)
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

func dummyConsumer(s storage.Storage, allowlist bool) consumer.Consumer {
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
	return &consumer.KafkaConsumer{
		Configuration: brokerCfg,
		Storage:       s,
	}
}

func TestProcessEmptyMessage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	c := dummyConsumer(mockStorage, true)

	message := sarama.ConsumerMessage{}
	// message is empty -> nothing should be written into storage
	_, err := c.ProcessMessage(&message)
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
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	c := dummyConsumer(mockStorage, true)

	message := sarama.ConsumerMessage{}
	message.Value = []byte(testdata.ConsumerMessage)
	// message is empty -> nothing should be written into storage
	_, err := c.ProcessMessage(&message)
	helpers.FailOnError(t, err)

	count, err := mockStorage.ReportsCount()
	helpers.FailOnError(t, err)

	assert.Equal(t, 1, count)
}

func TestProcessingMessageWithClosedStorage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)

	mockConsumer := dummyConsumer(mockStorage, true)
	closer()

	err := consumerProcessMessage(mockConsumer, testdata.ConsumerMessage)
	assert.EqualError(t, err, "sql: database is closed")
}

func TestProcessingMessageWithWrongDateFormat(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

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
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		mockConsumer, closer := ira_helpers.MustGetMockKafkaConsumerWithExpectedMessages(
			t,
			testTopicName,
			testOrgAllowlist,
			[]string{testdata.ConsumerMessage},
		)

		go mockConsumer.Serve()

		// wait for message processing
		ira_helpers.WaitForMockConsumerToHaveNConsumedMessages(mockConsumer, 1)

		closer()

		assert.Equal(t, uint64(1), mockConsumer.KafkaConsumer.GetNumberOfSuccessfullyConsumedMessages())
		assert.Equal(t, uint64(0), mockConsumer.KafkaConsumer.GetNumberOfErrorsConsumingMessages())
	}, testCaseTimeLimit)
}

func TestKafkaConsumerMockBadMessage(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		mockConsumer, closer := ira_helpers.MustGetMockKafkaConsumerWithExpectedMessages(
			t,
			testTopicName,
			testOrgAllowlist,
			[]string{"bad message"},
		)

		go mockConsumer.Serve()

		// wait for message processing
		ira_helpers.WaitForMockConsumerToHaveNConsumedMessages(mockConsumer, 1)

		closer()

		assert.Equal(t, uint64(0), mockConsumer.KafkaConsumer.GetNumberOfSuccessfullyConsumedMessages())
		assert.Equal(t, uint64(1), mockConsumer.KafkaConsumer.GetNumberOfErrorsConsumingMessages())
	}, testCaseTimeLimit)
}

func TestKafkaConsumerMockWritingToClosedStorage(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		mockConsumer, closer := ira_helpers.MustGetMockKafkaConsumerWithExpectedMessages(
			t, testTopicName, testOrgAllowlist, []string{testdata.ConsumerMessage},
		)

		err := mockConsumer.KafkaConsumer.Storage.Close()
		helpers.FailOnError(t, err)

		go mockConsumer.Serve()

		ira_helpers.WaitForMockConsumerToHaveNConsumedMessages(mockConsumer, 1)

		closer()

		assert.Equal(t, uint64(0), mockConsumer.KafkaConsumer.GetNumberOfSuccessfullyConsumedMessages())
		assert.Equal(t, uint64(1), mockConsumer.KafkaConsumer.GetNumberOfErrorsConsumingMessages())
	}, testCaseTimeLimit)
}

func TestKafkaConsumer_New(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		sarama.Logger = log.New(os.Stdout, saramaLogPrefix, log.LstdFlags)

		mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
		defer closer()

		mockBroker := sarama.NewMockBroker(t, 0)
		defer mockBroker.Close()

		mockBroker.SetHandlerByMap(ira_helpers.GetHandlersMapForMockConsumer(t, mockBroker, testTopicName))

		mockConsumer, err := consumer.New(broker.Configuration{
			Address: mockBroker.Addr(),
			Topic:   testTopicName,
			Enabled: true,
		}, mockStorage)
		helpers.FailOnError(t, err)

		err = mockConsumer.Close()
		helpers.FailOnError(t, err)
	}, testCaseTimeLimit)
}

// TODO: fix with new groups consumer
//func TestKafkaConsumer_New_FindCoordinatorRequestError(t *testing.T) {
//	helpers.RunTestWithTimeout(t, func(t *testing.T) {
//		sarama.Logger = log.New(os.Stdout, saramaLogPrefix, log.LstdFlags)
//
//		mockBroker := sarama.NewMockBroker(t, 0)
//		defer mockBroker.Close()
//
//		handlersMap := helpers.GetHandlersMapForMockConsumer(t, mockBroker, testTopicName)
//		handlersMap["FindCoordinatorRequest"] = sarama.NewMockFindCoordinatorResponse(t).
//			SetError(sarama.CoordinatorGroup, "", sarama.ErrUnknown)
//
//		mockBroker.SetHandlerByMap(handlersMap)
//
//		_, err := consumer.New(broker.Configuration{
//			Address: mockBroker.Addr(),
//			Topic:   testTopicName,
//			Enabled: true,
//		}, nil)
//		assert.EqualError(t, err, "kafka server: Unexpected (unknown?) server error.")
//	}, testCaseTimeLimit)
//}

func TestKafkaConsumer_ProcessMessage_OrganizationAllowlistDisabled(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	mockConsumer := dummyConsumer(mockStorage, false)

	err := consumerProcessMessage(mockConsumer, testdata.ConsumerMessage)
	helpers.FailOnError(t, err)
}

func TestKafkaConsumer_ProcessMessage_OrganizationIsNotAllowed(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	brokerCfg := broker.Configuration{
		Address:             "localhost:1234",
		Topic:               "topic",
		Group:               "group",
		OrgAllowlist:        mapset.NewSetWith(types.OrgID(123)), // in testdata, OrgID = 1
		OrgAllowlistEnabled: true,
	}
	mockConsumer := &consumer.KafkaConsumer{
		Configuration: brokerCfg,
		Storage:       mockStorage,
	}

	err := consumerProcessMessage(mockConsumer, testdata.ConsumerMessage)
	assert.EqualError(t, err, "organization ID is not in allow list")
}

func TestKafkaConsumer_ProcessMessage_OrganizationBadConfigIsNotAllowed(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	brokerCfg := broker.Configuration{
		Address:             "localhost:1234",
		Topic:               "topic",
		Group:               "group",
		OrgAllowlist:        nil,
		OrgAllowlistEnabled: true,
	}
	mockConsumer := &consumer.KafkaConsumer{
		Configuration: brokerCfg,
		Storage:       mockStorage,
	}

	err := consumerProcessMessage(mockConsumer, testdata.ConsumerMessage)
	assert.EqualError(t, err, "organization ID is not in allow list")
}

func TestKafkaConsumer_ProcessMessage_MessageFromTheFuture(t *testing.T) {
	buf := new(bytes.Buffer)
	zerolog_log.Logger = zerolog.New(buf)

	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	mockConsumer := &consumer.KafkaConsumer{
		Configuration: wrongBrokerCfg,
		Storage:       mockStorage,
	}

	message := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Report":` + testdata.ConsumerReport + `,
		"LastChecked": "` + time.Now().Add(24*time.Hour).Format(time.RFC3339) + `"
	}`

	err := consumerProcessMessage(mockConsumer, message)
	helpers.FailOnError(t, err)
	assert.Contains(t, buf.String(), "got a message from the future")
}

func TestKafkaConsumer_ProcessMessage_MoreRecentReportAlreadyExists(t *testing.T) {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	buf := new(bytes.Buffer)
	zerolog_log.Logger = zerolog.New(buf)

	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	mockConsumer := &consumer.KafkaConsumer{
		Configuration: wrongBrokerCfg,
		Storage:       mockStorage,
	}

	message := `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Report":` + testdata.ConsumerReport + `,
		"LastChecked": "` + time.Now().Format(time.RFC3339) + `"
	}`

	err := consumerProcessMessage(mockConsumer, message)
	helpers.FailOnError(t, err)

	message = `{
		"OrgID": ` + fmt.Sprint(testdata.OrgID) + `,
		"ClusterName": "` + string(testdata.ClusterName) + `",
		"Report":` + testdata.ConsumerReport + `,
		"LastChecked": "` + time.Now().Add(-24*time.Hour).Format(time.RFC3339) + `"
	}`

	err = consumerProcessMessage(mockConsumer, message)
	helpers.FailOnError(t, err)

	assert.Contains(t, buf.String(), "Skipping because a more recent report already exists for this cluster")
}

func TestKafkaConsumer_ConsumeClaim(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	kafkaConsumer := consumer.KafkaConsumer{
		Storage: mockStorage,
	}

	mockConsumerGroupSession := &ira_helpers.MockConsumerGroupSession{}
	mockConsumerGroupClaim := ira_helpers.NewMockConsumerGroupClaim(nil)

	err := kafkaConsumer.ConsumeClaim(mockConsumerGroupSession, mockConsumerGroupClaim)
	helpers.FailOnError(t, err)
}

func TestKafkaConsumer_ConsumeClaim_DBError(t *testing.T) {
	buf := new(bytes.Buffer)
	zerolog_log.Logger = zerolog.New(buf)

	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	closer()

	kafkaConsumer := consumer.KafkaConsumer{
		Storage: mockStorage,
	}

	mockConsumerGroupSession := &ira_helpers.MockConsumerGroupSession{}
	mockConsumerGroupClaim := ira_helpers.NewMockConsumerGroupClaim(nil)

	err := kafkaConsumer.ConsumeClaim(mockConsumerGroupSession, mockConsumerGroupClaim)
	helpers.FailOnError(t, err)

	assert.Contains(t, buf.String(), "unable to get latest offset")
}

func TestKafkaConsumer_ConsumeClaim_OKMessage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	kafkaConsumer := consumer.KafkaConsumer{
		Storage: mockStorage,
	}

	mockConsumerGroupSession := &ira_helpers.MockConsumerGroupSession{}
	mockConsumerGroupClaim := ira_helpers.NewMockConsumerGroupClaim([]*sarama.ConsumerMessage{
		ira_helpers.StringToSaramaConsumerMessage(testdata.ConsumerMessage),
	})

	err := kafkaConsumer.ConsumeClaim(mockConsumerGroupSession, mockConsumerGroupClaim)
	helpers.FailOnError(t, err)
}

func TestKafkaConsumer_SetupCleanup(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, false)
	defer closer()

	mockBroker := sarama.NewMockBroker(t, 0)
	defer mockBroker.Close()

	mockBroker.SetHandlerByMap(ira_helpers.GetHandlersMapForMockConsumer(t, mockBroker, testTopicName))

	mockConsumer, err := consumer.New(broker.Configuration{
		Address: mockBroker.Addr(),
		Topic:   testTopicName,
		Enabled: true,
	}, mockStorage)
	helpers.FailOnError(t, err)

	defer func() {
		helpers.FailOnError(t, mockConsumer.Close())
	}()

	// The functions don't really use their arguments at all,
	// so it's possible to just pass nil into them.
	helpers.FailOnError(t, mockConsumer.Setup(nil))
	helpers.FailOnError(t, mockConsumer.Cleanup(nil))
}
