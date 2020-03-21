package helpers

import (
	"testing"

	"github.com/RedHatInsights/insights-results-aggregator/consumer"
)

func MustCloseConsumer(t *testing.T, mockConsumer consumer.Consumer) {
	FailOnError(t, mockConsumer.Close())
}
