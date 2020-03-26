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

// Package producer contains functions that can be used to produce (i.e. send)
// messages to properly configured Kafka broker.
package producer

import (
	"fmt"

	"github.com/Shopify/sarama"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/metrics"
)

// Producer represents any producer
type Producer interface {
	ProduceMessage(msg string) (int32, int64, error)
	Close() error
}

// KafkaProducer is an implementation of Producer interface
type KafkaProducer struct {
	Configuration broker.Configuration
	Producer      sarama.SyncProducer
}

// New constructs new implementation of Producer interface
func New(brokerCfg broker.Configuration) (*KafkaProducer, error) {
	producer, err := sarama.NewSyncProducer([]string{brokerCfg.Address}, nil)
	if err != nil {
		log.Error().Err(err).Msg("New producer")
		return nil, err
	}

	return &KafkaProducer{
		Configuration: brokerCfg,
		Producer:      producer,
	}, nil
}

// ProduceMessage produces message to selected topic. That function returns
// partition ID and offset of new message or an error value in case of any
// problem on broker side.
func (producer *KafkaProducer) ProduceMessage(message string) (int32, int64, error) {
	brokerCfg := producer.Configuration
	msg := &sarama.ProducerMessage{Topic: brokerCfg.PublishTopic, Value: sarama.StringEncoder(message)}
	partition, offset, errout := producer.Producer.SendMessage(msg)

	if errout != nil {
		log.Error().Err(errout).Msg("FAILED to send message")
	} else {
		log.Info().Msg(fmt.Sprintf("message sent to partition %d at offset %d\n", partition, offset))
		metrics.ProducedMessages.Inc()
	}
	return partition, offset, errout
}

// Close allow the Sarama producer to be gracefully closed
func (producer *KafkaProducer) Close() error {
	if err := producer.Producer.Close(); err != nil {
		log.Error().Err(err).Msg("Producer.close()")
		return err
	}

	return nil
}
