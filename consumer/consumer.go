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

// Package consumer contains interface for any consumer that is able to
// process messages. It also contains implementation of Kafka consumer.
package consumer

import (
	"context"

	"github.com/Shopify/sarama"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// Consumer represents any consumer of insights-rules messages
type Consumer interface {
	Serve()
	Close() error
	ProcessMessage(msg *sarama.ConsumerMessage) error
}

// KafkaConsumer in an implementation of Consumer interface
// Example:
//
// kafkaConsumer, err := consumer.New(brokerCfg, storage)
// if err != nil {
//     panic(err)
// }
//
// kafkaConsumer.Serve()
//
// err := kafkaConsumer.Stop()
// if err != nil {
//     panic(err)
// }
type KafkaConsumer struct {
	Configuration                        broker.Configuration
	ConsumerGroup                        sarama.ConsumerGroup
	Storage                              storage.Storage
	numberOfSuccessfullyConsumedMessages uint64
	numberOfErrorsConsumingMessages      uint64
	ready                                chan bool
}

// DefaultSaramaConfig is a config which will be used by default
// here you can use specific version of a protocol for example
// useful for testing
var DefaultSaramaConfig *sarama.Config

// New constructs new implementation of Consumer interface
func New(brokerCfg broker.Configuration, storage storage.Storage) (*KafkaConsumer, error) {
	return NewWithSaramaConfig(brokerCfg, storage, DefaultSaramaConfig)
}

// NewWithSaramaConfig constructs new implementation of Consumer interface with custom sarama config
func NewWithSaramaConfig(
	brokerCfg broker.Configuration,
	storage storage.Storage,
	saramaConfig *sarama.Config,
) (*KafkaConsumer, error) {
	if saramaConfig == nil {
		saramaConfig = sarama.NewConfig()
		saramaConfig.Version = sarama.V0_10_2_0
	}

	consumerGroup, err := sarama.NewConsumerGroup([]string{brokerCfg.Address}, brokerCfg.Group, saramaConfig)
	if err != nil {
		return nil, err
	}

	consumer := &KafkaConsumer{
		Configuration:                        brokerCfg,
		ConsumerGroup:                        consumerGroup,
		Storage:                              storage,
		numberOfSuccessfullyConsumedMessages: 0,
		numberOfErrorsConsumingMessages:      0,
		ready:                                make(chan bool),
	}

	return consumer, nil
}

// Serve starts listening for messages and processing them. It blocks current thread.
func (consumer *KafkaConsumer) Serve() {
	ctx, cancel := context.WithCancel(context.Background())

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
func (consumer *KafkaConsumer) Setup(sarama.ConsumerGroupSession) error {
	log.Info().Msg("new session has been setup")
	// Mark the consumer as ready
	close(consumer.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (consumer *KafkaConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	log.Info().Msg("new session has been finished")
	return nil
}

// ConsumeClaim starts a consumer loop of ConsumerGroupClaim's Messages().
func (consumer *KafkaConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	log.Info().
		Int64(offsetKey, claim.InitialOffset()).
		Msg("starting messages loop")

	latestMessageOffset, err := consumer.Storage.GetLatestKafkaOffset()
	if err != nil {
		log.Error().Msg("unable to get latest offset")
		latestMessageOffset = 0
	}

	for message := range claim.Messages() {
		if types.KafkaOffset(message.Offset) <= latestMessageOffset {
			log.Warn().
				Int64(offsetKey, message.Offset).
				Msg("this offset was already processed by aggregator")
		}

		consumer.HandleMessage(message)

		session.MarkMessage(message, "")
		if types.KafkaOffset(message.Offset) > latestMessageOffset {
			latestMessageOffset = types.KafkaOffset(message.Offset)
		}
	}

	return nil
}

// Close method closes all resources used by consumer
func (consumer *KafkaConsumer) Close() error {
	if consumer.ConsumerGroup != nil {
		if err := consumer.ConsumerGroup.Close(); err != nil {
			log.Error().Err(err).Msg("Unable to close consumer group")
		}
	}

	return nil
}

// GetNumberOfSuccessfullyConsumedMessages returns number of consumed messages
// since creating KafkaConsumer obj
func (consumer *KafkaConsumer) GetNumberOfSuccessfullyConsumedMessages() uint64 {
	return consumer.numberOfSuccessfullyConsumedMessages
}

// GetNumberOfErrorsConsumingMessages returns number of errors during consuming messages
// since creating KafkaConsumer obj
func (consumer *KafkaConsumer) GetNumberOfErrorsConsumingMessages() uint64 {
	return consumer.numberOfErrorsConsumingMessages
}
