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
