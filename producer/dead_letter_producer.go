/*
Copyright © 2022 Red Hat, Inc.

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
	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/Shopify/sarama"
	"github.com/rs/zerolog/log"
)

// DeadLetterProducer is a producer for dead letter queue
type DeadLetterProducer struct {
	KafkaProducer KafkaProducer
	Configuration broker.Configuration
}

// NewDeadLetterProducer constructs producer for payload tracker topic.
// It is implemented as variable in order to allow monkey patching in unit tests.
var NewDeadLetterProducer = func(brokerCfg broker.Configuration) (*DeadLetterProducer, error) {
	if brokerCfg.DeadLetterQueueTopic == "" {
		return nil, nil
	}
	p, err := New(brokerCfg)
	if err != nil {
		log.Error().Err(err).Msg("unable to create a new dead letter producer")
		return nil, err
	}
	return &DeadLetterProducer{
		KafkaProducer: *p,
		Configuration: brokerCfg,
	}, nil
}

// SendDeadLetter loads the unprocessed message to the dedicated Kafka topic for further analysis
func (producer *DeadLetterProducer) SendDeadLetter(msg *sarama.ConsumerMessage) error {
	if msg == nil {
		log.Warn().Msg("message to be produced in dead letter is empty, skipping")
		return nil
	}

	partitionID, offset, err := producer.KafkaProducer.produceMessage(msg.Value, producer.Configuration.DeadLetterQueueTopic)
	if err != nil {
		log.Error().Err(err).Msg("unable to produce message to dead letter queue")
		return err
	}

	log.Info().Msgf("message has been produced to dead letter queue with partition ID %d and offset %d", partitionID, offset)
	return nil
}

// Close allow the Sarama producer to be gracefully closed
func (producer *DeadLetterProducer) Close() error {
	if err := producer.KafkaProducer.Close(); err != nil {
		log.Error().Err(err).Msg("unable to close dead letter producer")
		return err
	}

	return nil
}
