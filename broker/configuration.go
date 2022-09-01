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

// Package broker contains data types, interfaces, and methods related to
// brokers that can be used to consume input messages by aggegator.
package broker

import (
	"strings"
	"time"

	tlsutils "github.com/RedHatInsights/insights-operator-utils/tls"
	"github.com/Shopify/sarama"
	mapset "github.com/deckarep/golang-set"
	"github.com/rs/zerolog/log"
)

// Configuration represents configuration of Kafka broker
type Configuration struct {
	Address              string        `mapstructure:"address" toml:"address"`
	SecurityProtocol     string        `mapstructure:"security_protocol" toml:"security_protocol"`
	CertPath             string        `mapstructure:"cert_path" toml:"cert_path"`
	SaslMechanism        string        `mapstructure:"sasl_mechanism" toml:"sasl_mechanism"`
	SaslUsername         string        `mapstructure:"sasl_username" toml:"sasl_username"`
	SaslPassword         string        `mapstructure:"sasl_password" toml:"sasl_password"`
	Topic                string        `mapstructure:"topic" toml:"topic"`
	Timeout              time.Duration `mapstructure:"timeout" toml:"timeout"`
	PayloadTrackerTopic  string        `mapstructure:"payload_tracker_topic" toml:"payload_tracker_topic"`
	DeadLetterQueueTopic string        `mapstructure:"dead_letter_queue_topic" toml:"dead_letter_queue_topic"`
	ServiceName          string        `mapstructure:"service_name" toml:"service_name"`
	Group                string        `mapstructure:"group" toml:"group"`
	Enabled              bool          `mapstructure:"enabled" toml:"enabled"`
	OrgAllowlist         mapset.Set    `mapstructure:"org_allowlist_file" toml:"org_allowlist_file"`
	OrgAllowlistEnabled  bool          `mapstructure:"enable_org_allowlist" toml:"enable_org_allowlist"`
}

// SaramaConfigFromBrokerConfig returns a Config struct from broker.Configuration parameters
func SaramaConfigFromBrokerConfig(cfg Configuration) (*sarama.Config, error) {
	saramaConfig := sarama.NewConfig()
	saramaConfig.Version = sarama.V0_10_2_0

	if cfg.Timeout > 0 {
		saramaConfig.Net.DialTimeout = cfg.Timeout
		saramaConfig.Net.ReadTimeout = cfg.Timeout
		saramaConfig.Net.WriteTimeout = cfg.Timeout
	}

	if strings.Contains(cfg.SecurityProtocol, "SSL") {
		saramaConfig.Net.TLS.Enable = true
	}
	if cfg.CertPath != "" || cfg.CertPath != nil {
		tlsConfig, err := tlsutils.NewTLSConfig(cfg.CertPath)
		if err != nil {
			log.Error().Msgf("Unable to load TLS config for %s cert", cfg.CertPath)
			return nil, err
		}
		saramaConfig.Net.TLS.Config = tlsConfig
	}
	if strings.HasPrefix(cfg.SecurityProtocol, "SASL_") {
		log.Info().Msg("Configuring SASL authentication")
		saramaConfig.Net.SASL.Enable = true
		saramaConfig.Net.SASL.User = cfg.SaslUsername
		saramaConfig.Net.SASL.Password = cfg.SaslPassword
		saramaConfig.Net.SASL.Mechanism = sarama.SASLMechanism(cfg.SaslMechanism)
	}
	return saramaConfig, nil
}
