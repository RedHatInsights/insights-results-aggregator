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

	"github.com/Shopify/sarama/mocks"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/producer"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
}

// Test Producer creation with a non accesible Kafka broker
func TestNewProducerBadBroker(t *testing.T) {
	const expectedErr = "kafka: client has run out of available brokers to talk to (Is your cluster reachable?)"

	brokerCfg := broker.Configuration{
		Address:      "localhost:1234",
		PublishTopic: "topic",
		Group:        "Group",
	}

	_, err := producer.New(brokerCfg)
	assert.EqualError(t, err, expectedErr)
}

// Test ProduceMessage using a Sarama Mock producer. Assume sending success
func TestProducerProduceMessage(t *testing.T) {
	brokerCfg := broker.Configuration{
		Address:      "localhost:1234",
		PublishTopic: "topic",
		Group:        "Group",
	}

	mockProducer := mocks.NewSyncProducer(t, nil)
	mockProducer.ExpectSendMessageAndSucceed()
	kafkaProducer := producer.KafkaProducer{
		Configuration: brokerCfg,
		Producer:      mockProducer,
	}

	_, _, err := kafkaProducer.ProduceMessage(producer.PayloadTrackerMessage{})
	helpers.FailOnError(t, err)
}

// Test ProduceMessage using a Sarama Mock producer. Assume sending fails
func TestProducerProduceMessageFails(t *testing.T) {
	expectedErr := errors.New("unable to send the message")

	brokerCfg := broker.Configuration{
		Address:      "localhost:1234",
		PublishTopic: "topic",
		Group:        "Group",
	}

	mockProducer := mocks.NewSyncProducer(t, nil)
	mockProducer.ExpectSendMessageAndFail(expectedErr)

	kafkaProducer := producer.KafkaProducer{
		Configuration: brokerCfg,
		Producer:      mockProducer,
	}

	_, _, err := kafkaProducer.ProduceMessage(producer.PayloadTrackerMessage{})
	assert.EqualError(t, err, expectedErr.Error())
}

// Test Close on success
func TestProducerCloseSuccess(t *testing.T) {
	brokerCfg := broker.Configuration{
		Address:      "localhost:1234",
		PublishTopic: "topic",
		Group:        "Group",
	}

	mockProducer := mocks.NewSyncProducer(t, nil)
	prod := producer.KafkaProducer{
		Configuration: brokerCfg,
		Producer:      mockProducer,
	}

	err := prod.Close()
	helpers.FailOnError(t, err)
}
