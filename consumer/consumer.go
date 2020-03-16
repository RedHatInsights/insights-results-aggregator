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
	"encoding/json"
	"errors"
	"time"

	"github.com/Shopify/sarama"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/metrics"
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
type KafkaConsumer struct {
	Configuration                        broker.Configuration
	Consumer                             sarama.Consumer
	PartitionConsumer                    sarama.PartitionConsumer
	Storage                              storage.Storage
	numberOfSuccessfullyConsumedMessages uint64
	numberOfErrorsConsumingMessages      uint64
	offsetManager                        sarama.OffsetManager
	partitionOffsetManager               sarama.PartitionOffsetManager
	client                               sarama.Client
}

// report represents report send in a message consumed from any broker
type report map[string]*json.RawMessage

// incomingMessage is representation of message consumed from any broker
type incomingMessage struct {
	Organization *types.OrgID       `json:"OrgID"`
	ClusterName  *types.ClusterName `json:"ClusterName"`
	Report       *report            `json:"Report"`
	// LastChecked is a date in format "2020-01-23T16:15:59.478901889Z"
	LastChecked string `json:"LastChecked"`
}

// New constructs new implementation of Consumer interface
func New(brokerCfg broker.Configuration, storage storage.Storage) (*KafkaConsumer, error) {
	client, err := sarama.NewClient([]string{brokerCfg.Address}, nil)
	if err != nil {
		return nil, err
	}

	consumer, err := sarama.NewConsumerFromClient(client)
	if err != nil {
		return nil, err
	}

	partitions, err := consumer.Partitions(brokerCfg.Topic)
	if err != nil {
		return nil, err
	}

	offsetManager, err := sarama.NewOffsetManagerFromClient(brokerCfg.Group, client)
	if err != nil {
		return nil, err
	}

	partitionOffsetManager, err := offsetManager.ManagePartition(brokerCfg.Topic, partitions[0])
	if err != nil {
		return nil, err
	}

	nextOffset, _ := partitionOffsetManager.NextOffset()
	if nextOffset < 0 {
		// if next offset wasn't stored yet, initial state of the broker
		nextOffset = sarama.OffsetOldest
	}

	partitionConsumer, err := consumer.ConsumePartition(
		brokerCfg.Topic,
		partitions[0],
		nextOffset,
	)
	if err != nil {
		return nil, err
	}

	return &KafkaConsumer{
		Configuration:          brokerCfg,
		Consumer:               consumer,
		PartitionConsumer:      partitionConsumer,
		Storage:                storage,
		offsetManager:          offsetManager,
		partitionOffsetManager: partitionOffsetManager,
		client:                 client,
	}, nil
}

// checkReportStructure tests if the report has correct structure
func checkReportStructure(r report) error {
	// the structure is not well defined yet, so all we should do is to check if all keys are there
	expectedKeys := []string{"fingerprints", "info", "reports", "skips", "system"}
	for _, expectedKey := range expectedKeys {
		_, found := r[expectedKey]
		if !found {
			return errors.New("Improper report structure, missing key " + expectedKey)
		}
	}
	return nil
}

// parseMessage tries to parse incoming message and read all required attributes from it
func parseMessage(messageValue []byte) (incomingMessage, error) {
	var deserialized incomingMessage

	err := json.Unmarshal(messageValue, &deserialized)
	if err != nil {
		return deserialized, err
	}

	if deserialized.Organization == nil {
		return deserialized, errors.New("missing required attribute 'OrgID'")
	}
	if deserialized.ClusterName == nil {
		return deserialized, errors.New("missing required attribute 'ClusterName'")
	}
	if deserialized.Report == nil {
		return deserialized, errors.New("missing required attribute 'Report'")
	}

	_, err = uuid.Parse(string(*deserialized.ClusterName))

	if err != nil {
		return deserialized, errors.New("cluster name is not a UUID")
	}

	err = checkReportStructure(*deserialized.Report)
	if err != nil {
		log.Print("Deserialied report read from message with improper structure:")
		log.Print(*deserialized.Report)
		return deserialized, err
	}

	return deserialized, nil
}

// organizationAllowed checks whether the given organization is on whitelist or not
func organizationAllowed(consumer *KafkaConsumer, orgID types.OrgID) bool {
	whitelist := consumer.Configuration.OrgWhitelist
	if whitelist == nil {
		return false
	}

	orgWhitelisted := whitelist.Contains(types.OrgID(orgID))

	return orgWhitelisted
}

// Serve starts listening for messages and processing them. It blocks current thread
func (consumer *KafkaConsumer) Serve() {
	log.Printf("Consumer has been started, waiting for messages send to topic %s\n", consumer.Configuration.Topic)

	for msg := range consumer.PartitionConsumer.Messages() {
		err := consumer.ProcessMessage(msg)
		if err != nil {
			log.Error().Err(err).Msg("Error processing message consumed from Kafka")
			consumer.numberOfErrorsConsumingMessages++
		} else {
			consumer.numberOfSuccessfullyConsumedMessages++
		}
	}
}

// ProcessMessage processes an incoming message
func (consumer *KafkaConsumer) ProcessMessage(msg *sarama.ConsumerMessage) error {
	log.Printf("Consumed message offset %d\n", msg.Offset)
	message, err := parseMessage(msg.Value)
	if err != nil {
		log.Error().Err(err).Msg("Error parsing message from Kafka")
		return err
	}
	metrics.ConsumedMessages.Inc()

	if ok := organizationAllowed(consumer, *message.Organization); !ok {
		return errors.New("Organization ID is not whitelisted")
	}

	log.Printf("Results for organization %d and cluster %s", *message.Organization, *message.ClusterName)

	reportAsStr, err := json.Marshal(*message.Report)
	if err != nil {
		log.Error().Err(err).Msg("Error marshalling report")
		return err
	}

	lastCheckedTime, err := time.Parse(time.RFC3339Nano, message.LastChecked)
	if err != nil {
		log.Error().Err(err).Msg("Error parsing date from message")
		return err
	}

	err = consumer.Storage.WriteReportForCluster(
		*message.Organization,
		*message.ClusterName,
		types.ClusterReport(reportAsStr),
		lastCheckedTime,
	)
	if err != nil {
		log.Error().Err(err).Msg("Error writing report to database")
		return err
	}
	// message has been parsed and stored into storage

	// remember offset
	if consumer.partitionOffsetManager != nil {
		consumer.partitionOffsetManager.MarkOffset(msg.Offset+1, "")
	}

	return nil
}

// Close method closes all resources used by consumer
func (consumer *KafkaConsumer) Close() error {
	err := consumer.PartitionConsumer.Close()
	if err != nil {
		return err
	}

	err = consumer.Consumer.Close()
	if err != nil {
		return err
	}

	if consumer.partitionOffsetManager != nil {
		err = consumer.partitionOffsetManager.Close()
		if err != nil {
			return err
		}
	}

	if consumer.partitionOffsetManager != nil {
		err = consumer.offsetManager.Close()
		if err != nil {
			return err
		}
	}

	if consumer.client != nil {
		err = consumer.client.Close()
		if err != nil {
			return err
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
