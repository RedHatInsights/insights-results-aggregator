package consumer_test

import (
	"testing"

	"github.com/rs/zerolog"

	"github.com/RedHatInsights/insights-results-aggregator/consumer"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/tests/testdata"
)

func BenchmarkKafkaConsumer_ProcessMessage_NoStorage(b *testing.B) {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

	mockConsumer := &consumer.KafkaConsumer{
		Storage: &storage.NoopStorage{},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mustConsumerProcessMessage(b, mockConsumer, testdata.ConsumerMessage)
	}
}

func BenchmarkKafkaConsumer_ProcessMessage_SameMessageSQLiteStorage(b *testing.B) {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

	mockStorage := helpers.MustGetMockStorage(b, true)
	defer helpers.MustCloseStorage(b, mockStorage)

	mockConsumer := &consumer.KafkaConsumer{
		Storage: mockStorage,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mustConsumerProcessMessage(b, mockConsumer, testdata.ConsumerMessage)
	}
}

func BenchmarkKafkaConsumer_ProcessMessage_DifferentMessagesSQLiteStorage(b *testing.B) {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

	mockStorage := helpers.MustGetMockStorage(b, true)
	defer helpers.MustCloseStorage(b, mockStorage)

	mockConsumer := &consumer.KafkaConsumer{
		Storage: mockStorage,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mustConsumerProcessMessage(b, mockConsumer, testdata.GetRandomConsumerMessage())
	}
}
