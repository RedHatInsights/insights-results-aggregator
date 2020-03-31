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

package helpers

import (
	"testing"
	"time"

	"github.com/Shopify/sarama"
	sarama_mocks "github.com/Shopify/sarama/mocks"
	mapset "github.com/deckarep/golang-set"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/consumer"
)

// MustGetMockKafkaConsumerWithExpectedMessages creates mocked kafka consumer
// which produces list of messages automatically
// calls t.Fatal on error
func MustGetMockKafkaConsumerWithExpectedMessages(
	t *testing.T,
	topic string,
	orgWhiteList mapset.Set,
	messages []string,
) *consumer.KafkaConsumer {
	mockConsumer, err := GetMockKafkaConsumerWithExpectedMessages(t, topic, orgWhiteList, messages)
	if err != nil {
		t.Fatal(err)
	}

	return mockConsumer
}

// GetMockKafkaConsumerWithExpectedMessages creates mocked kafka consumer
// which produces list of messages automatically
func GetMockKafkaConsumerWithExpectedMessages(
	t *testing.T, topic string, orgWhiteList mapset.Set, messages []string,
) (*consumer.KafkaConsumer, error) {
	mockConsumer := sarama_mocks.NewConsumer(t, nil)

	for _, message := range messages {
		mockConsumer.
			ExpectConsumePartition(topic, 0, sarama.OffsetOldest).
			YieldMessage(&sarama.ConsumerMessage{Value: []byte(message)})
	}

	mockPartitionConsumer, err := mockConsumer.ConsumePartition(
		topic, 0, sarama.OffsetOldest,
	)
	if err != nil {
		return nil, err
	}

	mockStorage := MustGetMockStorage(t, true)

	return &consumer.KafkaConsumer{
		Configuration: broker.Configuration{
			Address:      "",
			Topic:        "",
			Group:        "",
			Enabled:      true,
			OrgWhitelist: orgWhiteList,
		},
		Consumer:          mockConsumer,
		PartitionConsumer: mockPartitionConsumer,
		Storage:           mockStorage,
	}, nil
}

// WaitForMockConsumerToHaveNConsumedMessages waits until mockConsumer has at least N
// consumed(either successfully or not) messages
func WaitForMockConsumerToHaveNConsumedMessages(mockConsumer *consumer.KafkaConsumer, nMessages uint64) {
	for {
		n := mockConsumer.GetNumberOfSuccessfullyConsumedMessages() +
			mockConsumer.GetNumberOfErrorsConsumingMessages()
		if n >= nMessages {
			break
		}

		time.Sleep(500 * time.Millisecond)
	}
}

// GetHandlersMapForMockConsumer returns handlers for mock broker to successfully create a new consumer
func GetHandlersMapForMockConsumer(t *testing.T, mockBroker *sarama.MockBroker, topicName string) map[string]sarama.MockResponse {
	return map[string]sarama.MockResponse{
		"MetadataRequest": sarama.NewMockMetadataResponse(t).
			SetBroker(mockBroker.Addr(), mockBroker.BrokerID()).
			SetLeader(topicName, 0, mockBroker.BrokerID()),
		"OffsetRequest": sarama.NewMockOffsetResponse(t).
			SetOffset(topicName, 0, -1, 0).
			SetOffset(topicName, 0, -2, 0),
		"FetchRequest": sarama.NewMockFetchResponse(t, 1),
		"FindCoordinatorRequest": sarama.NewMockFindCoordinatorResponse(t).
			SetCoordinator(sarama.CoordinatorGroup, "", mockBroker),
		"OffsetFetchRequest": sarama.NewMockOffsetFetchResponse(t).
			SetOffset("", topicName, 0, 0, "", sarama.ErrNoError),
	}
}
