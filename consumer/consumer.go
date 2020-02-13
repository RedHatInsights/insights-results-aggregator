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

// Package consumer contains interface for any consume that is able to
// process messages. It also contains implementation of Kafka consumer.
package consumer

import (
	"encoding/json"
	"errors"
	"log"

	"github.com/Shopify/sarama"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/metrics"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// Consumer represents any consumer of insights-rules messages
type Consumer interface {
	Start() error
	Close() error
	ProcessMessage(msg *sarama.ConsumerMessage) error
}

// KafkaConsumer in an implementation of Consumer interface
type KafkaConsumer struct {
	Configuration     broker.Configuration
	Consumer          sarama.Consumer
	PartitionConsumer sarama.PartitionConsumer
	Storage           storage.Storage
}

type incomingMessage struct {
	Organization *types.OrgID       `json:"OrgID"`
	ClusterName  *types.ClusterName `json:"ClusterName"`
	Report       *interface{}       `json:"Report"`
}

// New constructs new implementation of Consumer interface
func New(brokerCfg broker.Configuration, storage storage.Storage) (Consumer, error) {
	c, err := sarama.NewConsumer([]string{brokerCfg.Address}, nil)
	if err != nil {
		return nil, err
	}

	partitions, err := c.Partitions(brokerCfg.Topic)
	if err != nil {
		return nil, err
	}

	partitionConsumer, err := c.ConsumePartition(brokerCfg.Topic, partitions[0], sarama.OffsetNewest)
	if err != nil {
		return nil, err
	}

	consumer := KafkaConsumer{
		Configuration:     brokerCfg,
		Consumer:          c,
		PartitionConsumer: partitionConsumer,
		Storage:           storage,
	}
	return consumer, nil
}

func parseMessage(messageValue []byte) (types.OrgID, types.ClusterName, interface{}, error) {
	var deserialized incomingMessage

	err := json.Unmarshal(messageValue, &deserialized)
	if err != nil {
		return 0, "", "", err
	}

	if deserialized.Organization == nil {
		return 0, "", "", errors.New("Missing required attribute 'OrgID'")
	}
	if deserialized.ClusterName == nil {
		return 0, "", "", errors.New("Missing required attribute 'ClusterName'")
	}
	if deserialized.Report == nil {
		return 0, "", "", errors.New("Missing required attribute 'Report'")
	}
	return *deserialized.Organization, *deserialized.ClusterName, *deserialized.Report, nil
}

// Start starts consumer
func (consumer KafkaConsumer) Start() error {
	log.Printf("Consumer has been started, waiting for messages send to topic %s\n", consumer.Configuration.Topic)
	consumed := 0
	for {
		msg := <-consumer.PartitionConsumer.Messages()
		err := consumer.ProcessMessage(msg)
		if err != nil {
			log.Println("Error processing message consumed from Kafka:", err)
		}
		consumed++
	}
}

// ProcessMessage processes an incoming message
func (consumer KafkaConsumer) ProcessMessage(msg *sarama.ConsumerMessage) error {
	log.Printf("Consumed message offset %d\n", msg.Offset)
	orgID, clusterName, report, err := parseMessage(msg.Value)
	if err != nil {
		log.Println("Error parsing message from Kafka:", err)
		return err
	}
	metrics.ConsumedMessages.Inc()
	log.Printf("Results for organization %d and cluster %s", orgID, clusterName)

	reportAsStr, err := json.Marshal(report)
	if err != nil {
		log.Println("Error marshalling report:", err)
		return err
	}

	err = consumer.Storage.WriteReportForCluster(orgID, clusterName, types.ClusterReport(reportAsStr))
	if err != nil {
		log.Println("Error writing report to database:", err)
		return err
	}
	// message has been parsed and stored into storage
	return nil
}

// Close method closes all resources used by consumer
func (consumer KafkaConsumer) Close() error {
	err := consumer.PartitionConsumer.Close()
	if err != nil {
		return err
	}
	err = consumer.Consumer.Close()
	return err
}
