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

// Package producer contains functions that can be used to produce (that is
// send) messages to properly configured Kafka broker.
package producer

import (
	"github.com/Shopify/sarama"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/metrics"
)

const (
	// StatusReceived is reported when a new payload is received.
	StatusReceived = "received"
	// StatusMessageProcessed is reported when the message of a payload has been processed.
	StatusMessageProcessed = "processed"
	// StatusSuccess is reported upon a successful handling of a payload.
	StatusSuccess = "success"
	// StatusError is reported when the handling of a payload fails for any reason.
	StatusError = "error"
)

// Producer represents any producer
type Producer interface {
	Close() error
}

// KafkaProducer is an implementation of Producer interface
type KafkaProducer struct {
	Producer sarama.SyncProducer
}

// New constructs new implementation of Producer interface
func New(brokerCfg broker.Configuration) (*KafkaProducer, error) {
	saramaConfig, err := broker.SaramaConfigFromBrokerConfig(brokerCfg)
	if err != nil {
		log.Error().Err(err).Msg("unable to create sarama configuration from current broker configuration")
		return nil, err
	}
	// needed producer parameter
	saramaConfig.Producer.Return.Successes = true

	producer, err := sarama.NewSyncProducer([]string{brokerCfg.Address}, saramaConfig)
	if err != nil {
		log.Error().Err(err).Msg("unable to create a new Kafka producer")
		return nil, err
	}

	return &KafkaProducer{
		Producer: producer,
	}, nil
}

// produceMessage produces message to selected topic. That function returns
// partition ID and offset of new message or an error value in case of any
// problem on broker side.
func (producer *KafkaProducer) produceMessage(jsonBytes []byte, topic string) (int32, int64, error) {
	producerMsg := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(jsonBytes),
	}

	partition, offset, err := producer.Producer.SendMessage(producerMsg)
	if err != nil {
		log.Error().Err(err).Msg("failed to produce message to Kafka")
	} else {
		log.Info().Msgf("message sent to partition %d at offset %d", partition, offset)
		metrics.ProducedMessages.Inc()
	}
	return partition, offset, err
}

// Close allow the Sarama producer to be gracefully closed
func (producer *KafkaProducer) Close() error {
	if err := producer.Producer.Close(); err != nil {
		log.Error().Err(err).Msg("unable to close Kafka producer")
		return err
	}

	return nil
}
