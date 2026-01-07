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

	"github.com/IBM/sarama"
	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	mapset "github.com/deckarep/golang-set/v2"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/consumer"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// MockKafkaConsumer is mock consumer
type MockKafkaConsumer struct {
	KafkaConsumer consumer.KafkaConsumer
	topic         string
	messages      []string
}

// Serve simulates sending messages
func (mockKafkaConsumer *MockKafkaConsumer) Serve() {
	for i, message := range mockKafkaConsumer.messages {
		_ = mockKafkaConsumer.KafkaConsumer.HandleMessage(&sarama.ConsumerMessage{
			Timestamp:      time.Now(),
			BlockTimestamp: time.Now(),
			Value:          []byte(message),
			Topic:          mockKafkaConsumer.topic,
			Partition:      0,
			Offset:         int64(i),
		})
	}
}

// Close closes mock consumer
func (mockKafkaConsumer *MockKafkaConsumer) Close(t testing.TB) {
	err := mockKafkaConsumer.KafkaConsumer.Close()
	helpers.FailOnError(t, err)
}

// MustGetMockOCPRulesConsumerWithExpectedMessages creates mocked OCP rules
// consumer which produces list of messages automatically
// calls t.Fatal on error
func MustGetMockOCPRulesConsumerWithExpectedMessages(
	t testing.TB,
	topic string,
	orgAllowlist mapset.Set[types.OrgID],
	messages []string,
) (*MockKafkaConsumer, func()) {
	mockConsumer, closer, err := GetMockOCPRulesConsumerWithExpectedMessages(t, topic, orgAllowlist, messages)
	if err != nil {
		t.Fatal(err)
	}

	return mockConsumer, closer
}

// MustGetMockDVOConsumerWithExpectedMessages same as MustGetMockOCPRulesConsumerWithExpectedMessages
// but for DVO
func MustGetMockDVOConsumerWithExpectedMessages(
	t testing.TB,
	topic string,
	orgAllowlist mapset.Set[types.OrgID],
	messages []string,
) (*MockKafkaConsumer, func()) {
	mockConsumer, closer, err := GetMockDVOConsumerWithExpectedMessages(t, topic, orgAllowlist, messages)
	if err != nil {
		t.Fatal(err)
	}

	return mockConsumer, closer
}

// GetMockOCPRulesConsumerWithExpectedMessages creates mocked OCP rules
// consumer which produces list of messages automatically
func GetMockOCPRulesConsumerWithExpectedMessages(
	t testing.TB, topic string, orgAllowlist mapset.Set[types.OrgID], messages []string,
) (*MockKafkaConsumer, func(), error) {
	mockStorage, storageCloser := MustGetPostgresStorage(t, true)

	mockConsumer := &MockKafkaConsumer{
		KafkaConsumer: consumer.KafkaConsumer{
			Configuration: broker.Configuration{
				Addresses:    "",
				Topic:        topic,
				Group:        "",
				Enabled:      true,
				OrgAllowlist: orgAllowlist,
			},
			Storage:          mockStorage,
			MessageProcessor: consumer.OCPRulesProcessor{},
		},
		topic:    topic,
		messages: messages,
	}

	return mockConsumer, func() {
		storageCloser()
		mockConsumer.Close(t)
	}, nil
}

// GetMockDVOConsumerWithExpectedMessages same as GetMockOCPRulesConsumerWithExpectedMessages
// but for DVO
func GetMockDVOConsumerWithExpectedMessages(
	t testing.TB, topic string, orgAllowlist mapset.Set[types.OrgID], messages []string,
) (*MockKafkaConsumer, func(), error) {
	mockStorage, storageCloser := MustGetPostgresStorageDVO(t, true)

	mockConsumer := &MockKafkaConsumer{
		KafkaConsumer: consumer.KafkaConsumer{
			Configuration: broker.Configuration{
				Addresses:    "",
				Topic:        topic,
				Group:        "",
				Enabled:      true,
				OrgAllowlist: orgAllowlist,
			},
			Storage:          mockStorage,
			MessageProcessor: consumer.DVORulesProcessor{},
		},
		topic:    topic,
		messages: messages,
	}

	return mockConsumer, func() {
		storageCloser()
		mockConsumer.Close(t)
	}, nil
}

// WaitForMockConsumerToHaveNConsumedMessages waits until mockConsumer has at least N
// consumed(either successfully or not) messages
func WaitForMockConsumerToHaveNConsumedMessages(mockConsumer *MockKafkaConsumer, nMessages uint64) {
	for {
		n := mockConsumer.KafkaConsumer.GetNumberOfSuccessfullyConsumedMessages() +
			mockConsumer.KafkaConsumer.GetNumberOfErrorsConsumingMessages()
		if n >= nMessages {
			break
		}

		time.Sleep(500 * time.Millisecond)
	}
}

// GetHandlersMapForMockConsumer returns handlers for mock broker to successfully create a new consumer
func GetHandlersMapForMockConsumer(t testing.TB, mockBroker *sarama.MockBroker, topicName string) map[string]sarama.MockResponse {
	return map[string]sarama.MockResponse{
		"ApiVersionsRequest": sarama.NewMockApiVersionsResponse(t),
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
