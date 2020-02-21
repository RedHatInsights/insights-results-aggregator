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
