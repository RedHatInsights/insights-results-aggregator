package consumer_test

import (
	"log"
	"os"
	"testing"

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/consumer"
	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/Shopify/sarama"
)

type NoopProcessor struct{}

func (p *NoopProcessor) HandleMessage(msg *sarama.ConsumerMessage) error {
	return nil
}

func TestHeartbeatsConsumer_New(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		sarama.Logger = log.New(os.Stdout, saramaLogPrefix, log.LstdFlags)

		mockBroker := sarama.NewMockBroker(t, 0)
		defer mockBroker.Close()

		mockBroker.SetHandlerByMap(ira_helpers.GetHandlersMapForMockConsumer(t, mockBroker, testTopicName))

		mockConsumer, err := consumer.NewKfkConsumer(broker.Configuration{
			Addresses: mockBroker.Addr(),
			Topic:     testTopicName,
			Enabled:   true,
		}, &NoopProcessor{})
		helpers.FailOnError(t, err)

		err = mockConsumer.Close()
		helpers.FailOnError(t, err)
	}, testCaseTimeLimit)
}
