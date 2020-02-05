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

package consumer

import (
	"github.com/Shopify/sarama"
	"log"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
)

// Consumer represents any consumer of insights-rules messages
type Consumer interface {
	Start() error
	Close() error
}

// Impl in an implementation of Consumer interface
type Impl struct {
	Configuration     broker.Configuration
	Consumer          sarama.Consumer
	PartitionConsumer sarama.PartitionConsumer
}

// New constructs new implementation of Consumer interface
func New(brokerCfg broker.Configuration) (Consumer, error) {
	c, err := sarama.NewConsumer([]string{brokerCfg.Address}, nil)
	if err != nil {
		return nil, err
	}

	partitions, err := c.Partitions(brokerCfg.Topic)

	partitionConsumer, err := c.ConsumePartition(brokerCfg.Topic, partitions[0], sarama.OffsetNewest)
	if err != nil {
		return nil, err
	}

	consumer := Impl{
		Configuration:     brokerCfg,
		Consumer:          c,
		PartitionConsumer: partitionConsumer,
	}
	return consumer, nil
}

// Start starts consumer
func (consumer Impl) Start() error {
	log.Printf("Consumer has been started, waiting for messages send to topic %s\n", consumer.Configuration.Topic)
	consumed := 0
	for {
		msg := <-consumer.PartitionConsumer.Messages()
		log.Printf("Consumed message offset %d\n", msg.Offset)
		consumed++
	}
}

// Close method closes all resources used by consumer
func (consumer Impl) Close() error {
	err := consumer.PartitionConsumer.Close()
	if err != nil {
		return err
	}
	err = consumer.Consumer.Close()
	return err
}
