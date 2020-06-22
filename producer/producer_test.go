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

package producer_test

import (
	"errors"
	"testing"
	"time"

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/Shopify/sarama"
	"github.com/Shopify/sarama/mocks"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/producer"
	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

var (
	brokerCfg = broker.Configuration{
		Address:             "localhost:1234",
		Topic:               "consumer-topic",
		PayloadTrackerTopic: "payload-tracker-topic",
		Group:               "test-group",
	}
	// Base UNIX time plus approximately 50 years (not long before year 2020).
	testTimestamp = time.Unix(50*365*24*60*60, 0)
)

func init() {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
}

// Test Producer creation with a non accessible Kafka broker
func TestNewProducerBadBroker(t *testing.T) {
	const expectedErr = "kafka: client has run out of available brokers to talk to (Is your cluster reachable?)"

	_, err := producer.New(brokerCfg)
	assert.EqualError(t, err, expectedErr)
}

// TestProducerTrackPayload calls the TrackPayload function using a mock Sarama producer.
func TestProducerTrackPayload(t *testing.T) {
	mockProducer := mocks.NewSyncProducer(t, nil)
	mockProducer.ExpectSendMessageAndSucceed()

	kafkaProducer := producer.KafkaProducer{
		Configuration: brokerCfg,
		Producer:      mockProducer,
	}
	defer func() {
		helpers.FailOnError(t, kafkaProducer.Close())
	}()

	err := kafkaProducer.TrackPayload(testdata.TestRequestID, testTimestamp, producer.StatusReceived)
	assert.NoError(t, err, "payload tracking failed")
}

// TestProducerTrackPayloadEmptyRequestID calls the TrackPayload function using a mock Sarama producer.
// The request ID passed to the function is empty and therefore
// a warning should be logged and nothing more should happen.
func TestProducerTrackPayloadEmptyRequestID(t *testing.T) {
	mockProducer := mocks.NewSyncProducer(t, nil)

	kafkaProducer := producer.KafkaProducer{
		Configuration: brokerCfg,
		Producer:      mockProducer,
	}
	defer func() {
		helpers.FailOnError(t, kafkaProducer.Close())
	}()

	err := kafkaProducer.TrackPayload(types.RequestID(""), testTimestamp, producer.StatusReceived)
	assert.NoError(t, err, "payload tracking failed")
}

// TestProducerTrackPayloadWithError checks that errors
// from the underlying producer are correctly returned.
func TestProducerTrackPayloadWithError(t *testing.T) {
	const producerErrorMessage = "unable to send the message"

	mockProducer := mocks.NewSyncProducer(t, nil)
	mockProducer.ExpectSendMessageAndFail(errors.New(producerErrorMessage))

	kafkaProducer := producer.KafkaProducer{
		Configuration: brokerCfg,
		Producer:      mockProducer,
	}
	defer func() {
		helpers.FailOnError(t, kafkaProducer.Close())
	}()

	err := kafkaProducer.TrackPayload(testdata.TestRequestID, testTimestamp, producer.StatusReceived)
	assert.EqualError(t, err, producerErrorMessage)
}

// TestProducerClose makes sure it's possible to close the producer.
func TestProducerClose(t *testing.T) {
	mockProducer := mocks.NewSyncProducer(t, nil)
	prod := producer.KafkaProducer{
		Configuration: brokerCfg,
		Producer:      mockProducer,
	}

	err := prod.Close()
	assert.NoError(t, err, "failed to close Kafka producer")
}

func TestProducerNew(t *testing.T) {
	mockBroker := sarama.NewMockBroker(t, 0)
	defer mockBroker.Close()

	mockBroker.SetHandlerByMap(ira_helpers.GetHandlersMapForMockConsumer(t, mockBroker, brokerCfg.PayloadTrackerTopic))

	prod, err := producer.New(
		broker.Configuration{
			Address:             mockBroker.Addr(),
			Topic:               brokerCfg.Topic,
			PayloadTrackerTopic: brokerCfg.PayloadTrackerTopic,
			Enabled:             brokerCfg.Enabled,
		})
	helpers.FailOnError(t, err)

	helpers.FailOnError(t, prod.Close())
}
