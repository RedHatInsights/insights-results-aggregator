/*
Copyright © 2020, 2023 Red Hat, Inc.

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

// Package broker contains data types, interfaces, and methods related to
// brokers that can be used to consume input messages by aggegator.
package broker

import (
	"crypto/sha512"
	"strings"
	"time"

	"github.com/IBM/sarama"
	tlsutils "github.com/RedHatInsights/insights-operator-utils/tls"
	"github.com/RedHatInsights/insights-results-aggregator/types"
	mapset "github.com/deckarep/golang-set/v2"
	"github.com/rs/zerolog/log"
)

// SaramaVersion is the version of Kafka API used in sarama client
var SaramaVersion = sarama.V3_8_0_0

// Configuration represents configuration of Kafka broker
type Configuration struct {
	Addresses                        string                  `mapstructure:"addresses" toml:"addresses"`
	SecurityProtocol                 string                  `mapstructure:"security_protocol" toml:"security_protocol"`
	CertPath                         string                  `mapstructure:"cert_path" toml:"cert_path"`
	SaslMechanism                    string                  `mapstructure:"sasl_mechanism" toml:"sasl_mechanism"`
	SaslUsername                     string                  `mapstructure:"sasl_username" toml:"sasl_username"`
	SaslPassword                     string                  `mapstructure:"sasl_password" toml:"sasl_password"`
	Topic                            string                  `mapstructure:"topic" toml:"topic"`
	Timeout                          time.Duration           `mapstructure:"timeout" toml:"timeout"`
	PayloadTrackerTopic              string                  `mapstructure:"payload_tracker_topic" toml:"payload_tracker_topic"`
	DeadLetterQueueTopic             string                  `mapstructure:"dead_letter_queue_topic" toml:"dead_letter_queue_topic"`
	ServiceName                      string                  `mapstructure:"service_name" toml:"service_name"`
	Group                            string                  `mapstructure:"group" toml:"group"`
	Enabled                          bool                    `mapstructure:"enabled" toml:"enabled"`
	OrgAllowlist                     mapset.Set[types.OrgID] `mapstructure:"org_allowlist_file" toml:"org_allowlist_file"`
	OrgAllowlistEnabled              bool                    `mapstructure:"enable_org_allowlist" toml:"enable_org_allowlist"`
	ClientID                         string                  `mapstructure:"client_id" toml:"client_id"`
	DisplayMessageWithWrongStructure bool                    `mapstructure:"display_message_with_wrong_structure" toml:"display_message_with_wrong_structure"`
}

// SaramaConfigFromBrokerConfig returns a Config struct from broker.Configuration parameters
func SaramaConfigFromBrokerConfig(cfg Configuration) (*sarama.Config, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = SaramaVersion

	if cfg.Timeout > 0 {
		saramaConfig.Net.DialTimeout = cfg.Timeout
		saramaConfig.Net.ReadTimeout = cfg.Timeout
		saramaConfig.Net.WriteTimeout = cfg.Timeout
	}

	if strings.Contains(cfg.SecurityProtocol, "SSL") {
		saramaConfig.Net.TLS.Enable = true
	}

	if strings.EqualFold(cfg.SecurityProtocol, "SSL") && cfg.CertPath != "" {
		tlsConfig, err := tlsutils.NewTLSConfig(cfg.CertPath)
		if err != nil {
			log.Error().Msgf("Unable to load TLS config for %s cert", cfg.CertPath)
			return nil, err
		}
		saramaConfig.Net.TLS.Config = tlsConfig
	} else if strings.HasPrefix(cfg.SecurityProtocol, "SASL_") {
		log.Info().Msg("Configuring SASL authentication")
		saramaConfig.Net.SASL.Enable = true
		saramaConfig.Net.SASL.User = cfg.SaslUsername
		saramaConfig.Net.SASL.Password = cfg.SaslPassword
		saramaConfig.Net.SASL.Mechanism = sarama.SASLMechanism(cfg.SaslMechanism)

		if strings.EqualFold(cfg.SaslMechanism, sarama.SASLTypeSCRAMSHA512) {
			log.Info().Msg("Configuring SCRAM-SHA512")
			saramaConfig.Net.SASL.Handshake = true
			saramaConfig.Net.SASL.SCRAMClientGeneratorFunc = func() sarama.SCRAMClient {
				return &SCRAMClient{HashGeneratorFcn: sha512.New}
			}
		}
	}

	// ClientID is fully optional, but by setting it, we can get rid of some warning messages in logs
	if cfg.ClientID != "" {
		// if not set, the "sarama" will be used instead
		saramaConfig.ClientID = cfg.ClientID
	}

	// now the config structure is filled-in
	return saramaConfig, nil
}
