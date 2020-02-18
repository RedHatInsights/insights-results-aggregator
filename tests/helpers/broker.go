package helpers

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
)

// GetTestBrokerCfg returns config for a broker with address set in env var TEST_KAFKA_ADDRESS
func GetTestBrokerCfg(t *testing.T, topic string) broker.Configuration {
	brokerCfg := broker.Configuration{
		Address: os.Getenv("TEST_KAFKA_ADDRESS"),
		Topic:   topic,
		Group:   "",
	}

	assert.NotEmpty(
		t,
		brokerCfg.Address,
		`Please, set up TEST_KAFKA_ADDRESS env variable. For example "localhost:9092"`,
	)

	return brokerCfg
}
