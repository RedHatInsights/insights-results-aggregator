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
