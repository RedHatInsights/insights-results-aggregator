/*
Copyright © 2020, 2021, 2022, 2023 Red Hat, Inc.

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

// Package consumer contains interface for any consumer that is able to
// process messages. It also contains implementation of Kafka consumer.
//
// It is expected that consumed messages are generated by ccx-data-pipeline
// based on OCP rules framework. The report generated by the framework are
// enhanced with more context information taken from different sources, like
// the organization ID, account number, unique cluster name, and the
// LastChecked timestamp (taken from the incoming Kafka record containing the
// URL to the archive).
//
// It is also expected that consumed messages contains one INFO rule hit that
// contains cluster version. That rule hit is produced by special rule used
// only in external data pipeline:
// "version_info|CLUSTER_VERSION_INFO"
package consumer

import (
	"context"

	"github.com/Shopify/sarama"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/producer"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
)

// OCPRulesConsumer is an implementation of Consumer interface
// Example:
//
// ocpRulesConsumer, err := consumer.NewOCPRulesConsumer(brokerCfg, storage)
//
//	if err != nil {
//	    panic(err)
//	}
//
// ocpRulesConsumer.Serve()
//
// err := ocpRulesConsumer.Stop()
//
//	if err != nil {
//	    panic(err)
//	}
type OCPRulesConsumer struct {
	Configuration                        broker.Configuration
	ConsumerGroup                        sarama.ConsumerGroup
	Storage                              storage.Storage
	numberOfSuccessfullyConsumedMessages uint64
	numberOfErrorsConsumingMessages      uint64
	ready                                chan bool
	cancel                               context.CancelFunc
	payloadTrackerProducer               *producer.PayloadTrackerProducer
	deadLetterProducer                   *producer.DeadLetterProducer
}

// DefaultSaramaConfig is a config which will be used by default
// here you can use specific version of a protocol for example
// useful for testing
var DefaultSaramaConfig *sarama.Config

// NewOCPRulesConsumer constructs new implementation of Consumer interface
func NewOCPRulesConsumer(brokerCfg broker.Configuration, storage storage.Storage) (*OCPRulesConsumer, error) {
	return NewOCPRulesConsumerWithSaramaConfig(brokerCfg, storage, DefaultSaramaConfig)
}

// NewOCPRulesConsumerWithSaramaConfig constructs new implementation of Consumer interface with custom sarama config
func NewOCPRulesConsumerWithSaramaConfig(
	brokerCfg broker.Configuration,
	storage storage.Storage,
	saramaConfig *sarama.Config,
) (*OCPRulesConsumer, error) {
	var err error

	if saramaConfig == nil {
		saramaConfig, err = broker.SaramaConfigFromBrokerConfig(brokerCfg)
		if err != nil {
			log.Error().Err(err).Msg("unable to create sarama configuration from current broker configuration")
			return nil, err
		}
	}

	log.Info().
		Str("addr", brokerCfg.Address).
		Str("group", brokerCfg.Group).
		Msg("New consumer group")

	consumerGroup, err := sarama.NewConsumerGroup([]string{brokerCfg.Address}, brokerCfg.Group, saramaConfig)
	if err != nil {
		log.Error().Err(err).Msg("Unable to create consumer group")
		return nil, err
	}
	log.Info().Msg("Consumer group has been created")

	log.Info().Msg("Constructing payload tracker producer")
	payloadTrackerProducer, err := producer.NewPayloadTrackerProducer(brokerCfg)
	if err != nil {
		log.Error().Err(err).Msg("Unable to construct payload tracker producer")
		return nil, err
	}
	if payloadTrackerProducer == nil {
		log.Info().Msg("Payload tracker producer not configured")
	} else {
		log.Info().Msg("Payload tracker producer has been configured")
	}

	log.Info().Msg("Constructing DLQ producer")
	deadLetterProducer, err := producer.NewDeadLetterProducer(brokerCfg)
	if err != nil {
		log.Error().Err(err).Msg("Unable to construct dead letter producer")
		return nil, err
	}
	if deadLetterProducer == nil {
		log.Info().Msg("Dead letter producer not configured")
	} else {
		log.Info().Msg("Dead letter producer has been configured")
	}

	consumer := &OCPRulesConsumer{
		Configuration:                        brokerCfg,
		ConsumerGroup:                        consumerGroup,
		Storage:                              storage,
		numberOfSuccessfullyConsumedMessages: 0,
		numberOfErrorsConsumingMessages:      0,
		ready:                                make(chan bool),
		payloadTrackerProducer:               payloadTrackerProducer,
		deadLetterProducer:                   deadLetterProducer,
	}

	return consumer, nil
}

// Serve starts listening for messages and processing them. It blocks current thread.
func (consumer *OCPRulesConsumer) Serve() {
	ctx, cancel := context.WithCancel(context.Background())
	consumer.cancel = cancel

	go func() {
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if err := consumer.ConsumerGroup.Consume(ctx, []string{consumer.Configuration.Topic}, consumer); err != nil {
				log.Fatal().Err(err).Msg("unable to recreate kafka session")
			}

			// check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				return
			}

			log.Info().Msg("created new kafka session")

			consumer.ready = make(chan bool)
		}
	}()

	// Await till the consumer has been set up
	log.Info().Msg("waiting for consumer to become ready")
	<-consumer.ready
	log.Info().Msg("finished waiting for consumer to become ready")

	// Actual processing is done in goroutine created by sarama (see ConsumeClaim below)
	log.Info().Msg("started serving consumer")
	<-ctx.Done()
	log.Info().Msg("context cancelled, exiting")

	cancel()
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (consumer *OCPRulesConsumer) Setup(sarama.ConsumerGroupSession) error {
	log.Info().Msg("new session has been setup")
	// Mark the consumer as ready
	close(consumer.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (consumer *OCPRulesConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	log.Info().Msg("new session has been finished")
	return nil
}

// ConsumeClaim starts a consumer loop of ConsumerGroupClaim's Messages().
func (consumer *OCPRulesConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	log.Info().
		Int64(offsetKey, claim.InitialOffset()).
		Msg("starting messages loop")

	for message := range claim.Messages() {
		err := consumer.HandleMessage(message)
		if err != nil {
			// already handled in HandleMessage, just log
			log.Error().Err(err).Msg("Problem while handling the message")
		}
		session.MarkMessage(message, "")
	}

	return nil
}

// Close method closes all resources used by consumer
func (consumer *OCPRulesConsumer) Close() error {
	if consumer.cancel != nil {
		consumer.cancel()
	}

	if consumer.ConsumerGroup != nil {
		if err := consumer.ConsumerGroup.Close(); err != nil {
			log.Error().Err(err).Msg("unable to close consumer group")
		}
	}

	if consumer.payloadTrackerProducer != nil {
		if err := consumer.payloadTrackerProducer.Close(); err != nil {
			log.Error().Err(err).Msg("unable to close payload tracker Kafka producer")
		}
	}

	if consumer.deadLetterProducer != nil {
		if err := consumer.deadLetterProducer.Close(); err != nil {
			log.Error().Err(err).Msg("unable to close dead letter Kafka producer")
		}
	}

	return nil
}

// GetNumberOfSuccessfullyConsumedMessages returns number of consumed messages
// since creating OCPRulesConsumer obj
func (consumer *OCPRulesConsumer) GetNumberOfSuccessfullyConsumedMessages() uint64 {
	return consumer.numberOfSuccessfullyConsumedMessages
}

// GetNumberOfErrorsConsumingMessages returns number of errors during consuming messages
// since creating OCPRulesConsumer obj
func (consumer *OCPRulesConsumer) GetNumberOfErrorsConsumingMessages() uint64 {
	return consumer.numberOfErrorsConsumingMessages
}
