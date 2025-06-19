// Copyright 2022, 2023 Red Hat, Inc
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

package broker_test

import (
	"testing"
	"time"

	"github.com/IBM/sarama"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/stretchr/testify/assert"
)

func TestSaramaConfigFromBrokerConfig(t *testing.T) {
	cfg := broker.Configuration{}
	saramaConfig, err := broker.SaramaConfigFromBrokerConfig(cfg)
	helpers.FailOnError(t, err)
	assert.Equal(t, sarama.V3_8_0_0, saramaConfig.Version)

	cfg = broker.Configuration{
		Timeout: time.Second,
	}
	saramaConfig, err = broker.SaramaConfigFromBrokerConfig(cfg)
	helpers.FailOnError(t, err)
	assert.Equal(t, sarama.V3_8_0_0, saramaConfig.Version)
	assert.Equal(t, time.Second, saramaConfig.Net.DialTimeout)
	assert.Equal(t, time.Second, saramaConfig.Net.ReadTimeout)
	assert.Equal(t, time.Second, saramaConfig.Net.WriteTimeout)
	assert.Equal(t, "sarama", saramaConfig.ClientID) // default value

	cfg = broker.Configuration{
		SecurityProtocol: "SSL",
	}

	saramaConfig, err = broker.SaramaConfigFromBrokerConfig(cfg)
	helpers.FailOnError(t, err)
	assert.Equal(t, sarama.V3_8_0_0, saramaConfig.Version)
	assert.True(t, saramaConfig.Net.TLS.Enable)

	cfg = broker.Configuration{
		SecurityProtocol: "SASL_SSL",
		SaslMechanism:    "PLAIN",
		SaslUsername:     "username",
		SaslPassword:     "password",
		ClientID:         "foobarbaz",
	}
	saramaConfig, err = broker.SaramaConfigFromBrokerConfig(cfg)
	helpers.FailOnError(t, err)
	assert.Equal(t, sarama.V3_8_0_0, saramaConfig.Version)
	assert.True(t, saramaConfig.Net.TLS.Enable)
	assert.True(t, saramaConfig.Net.SASL.Enable)
	assert.Equal(t, sarama.SASLMechanism("PLAIN"), saramaConfig.Net.SASL.Mechanism)
	assert.Equal(t, "username", saramaConfig.Net.SASL.User)
	assert.Equal(t, "password", saramaConfig.Net.SASL.Password)
	assert.Equal(t, "foobarbaz", saramaConfig.ClientID)

	cfg.SaslMechanism = "SCRAM-SHA-512"
	saramaConfig, err = broker.SaramaConfigFromBrokerConfig(cfg)
	helpers.FailOnError(t, err)
	assert.Equal(t, sarama.V3_8_0_0, saramaConfig.Version)
	assert.True(t, saramaConfig.Net.TLS.Enable)
	assert.True(t, saramaConfig.Net.SASL.Enable)
	assert.Equal(t, sarama.SASLMechanism(sarama.SASLTypeSCRAMSHA512), saramaConfig.Net.SASL.Mechanism)
	assert.Equal(t, "username", saramaConfig.Net.SASL.User)
	assert.Equal(t, "password", saramaConfig.Net.SASL.Password)
	assert.Equal(t, "foobarbaz", saramaConfig.ClientID)
}

func TestBadConfiguration(t *testing.T) {
	cfg := broker.Configuration{
		SecurityProtocol: "SSL",
		CertPath:         "missing_path",
	}

	saramaCfg, err := broker.SaramaConfigFromBrokerConfig(cfg)
	assert.Error(t, err)
	assert.Nil(t, saramaCfg)
}
