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

const (
	// key for topic name used in structured log messages
	topicKey = "topic"
	// key for broker group name used in structured log messages
	groupKey = "group"
	// key for message offset used in structured log messages
	offsetKey = "offset"
	// key for message partition used in structured log messages
	partitionKey = "partition"
	// key for organization ID used in structured log messages
	organizationKey = "organization"
	// key for cluster ID used in structured log messages
	clusterKey = "cluster"
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

// Report represents report send in a message consumed from any broker
type Report map[string]*json.RawMessage

// incomingMessage is representation of message consumed from any broker
type incomingMessage struct {
	Organization *types.OrgID       `json:"OrgID"`
	ClusterName  *types.ClusterName `json:"ClusterName"`
	Report       *Report            `json:"Report"`
	// LastChecked is a date in format "2020-01-23T16:15:59.478901889Z"
	LastChecked string `json:"LastChecked"`
}

// DefaultSaramaConfig is a config which will be used by default
// here you can use specific version of a protocol for example
// useful for testing
var DefaultSaramaConfig *sarama.Config

// New constructs new implementation of Consumer interface
func New(brokerCfg broker.Configuration, storage storage.Storage) (*KafkaConsumer, error) {
	return NewWithSaramaConfig(brokerCfg, storage, DefaultSaramaConfig, brokerCfg.SaveOffset)
}

// NewWithSaramaConfig constructs new implementation of Consumer interface with custom sarama config
func NewWithSaramaConfig(
	brokerCfg broker.Configuration,
	storage storage.Storage,
	saramaConfig *sarama.Config,
	saveOffset bool,
) (*KafkaConsumer, error) {
	client, err := sarama.NewClient([]string{brokerCfg.Address}, saramaConfig)
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

	var (
		offsetManager          sarama.OffsetManager
		partitionOffsetManager sarama.PartitionOffsetManager
	)
	nextOffset := sarama.OffsetNewest

	if saveOffset {
		offsetManager, partitionOffsetManager, nextOffset, err = getOffsetManagers(brokerCfg, client, partitions)
		if err != nil {
			return nil, err
		}
	}

	partitionConsumer, err := consumer.ConsumePartition(
		brokerCfg.Topic,
		partitions[0],
		nextOffset,
	)
	if kErr, ok := err.(sarama.KError); ok && kErr == sarama.ErrOffsetOutOfRange {
		// try again with offset from the beginning
		log.Error().Err(err).Msg("consuming from the beginning")

		nextOffset = sarama.OffsetOldest
		partitionConsumer, err = consumer.ConsumePartition(
			brokerCfg.Topic,
			partitions[0],
			nextOffset,
		)
	}
	if err != nil {
		return nil, err
	}

	log.Info().Msgf("created consumer with starting offset %+v", nextOffset)

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

func getOffsetManagers(
	brokerCfg broker.Configuration, client sarama.Client, partitions []int32,
) (sarama.OffsetManager, sarama.PartitionOffsetManager, int64, error) {
	offsetManager, err := sarama.NewOffsetManagerFromClient(brokerCfg.Group, client)
	if err != nil {
		return nil, nil, 0, err
	}

	partitionOffsetManager, err := offsetManager.ManagePartition(brokerCfg.Topic, partitions[0])
	if err != nil {
		return nil, nil, 0, err
	}

	nextOffset, _ := partitionOffsetManager.NextOffset()
	if nextOffset < 0 {
		// if next offset wasn't stored yet, initial state of the broker
		log.Info().Msg("saved offset was not found, consuming from the beginning")
		nextOffset = sarama.OffsetOldest
	}

	return offsetManager, partitionOffsetManager, nextOffset, nil
}

// checkReportStructure tests if the report has correct structure
func checkReportStructure(r Report) error {
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
		log.Err(err).Msgf("Deserialized report read from message with improper structure: %v", *deserialized.Report)
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

	orgWhitelisted := whitelist.Contains(orgID)

	return orgWhitelisted
}

// Serve starts listening for messages and processing them. It blocks current thread
func (consumer *KafkaConsumer) Serve() {
	log.Info().Msgf("Consumer has been started, waiting for messages send to topic '%s'", consumer.Configuration.Topic)

	for msg := range consumer.PartitionConsumer.Messages() {
		metrics.ConsumedMessages.Inc()

		startTime := time.Now()
		err := consumer.ProcessMessage(msg)
		messageProcessingDuration := time.Since(startTime)

		log.Info().
			Int(offsetKey, int(msg.Offset)).
			Int(partitionKey, int(msg.Partition)).
			Str(topicKey, msg.Topic).
			Msgf("processing of message took '%v' seconds", messageProcessingDuration.Seconds())

		if err != nil {
			metrics.FailedMessagesProcessingTime.Observe(messageProcessingDuration.Seconds())
			metrics.ConsumingErrors.Inc()

			log.Error().Err(err).Msg("Error processing message consumed from Kafka")
			consumer.numberOfErrorsConsumingMessages++

			if err := consumer.Storage.WriteConsumerError(msg, err); err != nil {
				log.Error().Err(err).Msg("Unable to write consumer error to storage")
			} else {
				// if error is written, we don't want to deal with this message again
				consumer.saveLastMessageOffset(msg.Offset)
			}
		} else {
			metrics.SuccessfulMessagesProcessingTime.Observe(messageProcessingDuration.Seconds())
			consumer.numberOfSuccessfullyConsumedMessages++
			consumer.saveLastMessageOffset(msg.Offset)
		}
		endTime := time.Now()
		duration := endTime.Sub(startTime)
		log.Info().Int64("duration", duration.Milliseconds()).Int64(offsetKey, msg.Offset).Msg("Message consumed")
	}
}

func (consumer *KafkaConsumer) saveLastMessageOffset(lastMessageOffset int64) {
	// remember offset
	if consumer.partitionOffsetManager != nil {
		consumer.partitionOffsetManager.MarkOffset(lastMessageOffset+1, "")
	} else {
		log.Warn().Msgf(`not saving offset "%+v" because it's disabled'`, lastMessageOffset)
	}
}

func logMessageInfo(consumer *KafkaConsumer, originalMessage *sarama.ConsumerMessage, parsedMessage incomingMessage, event string) {
	log.Info().
		Int(offsetKey, int(originalMessage.Offset)).
		Int(partitionKey, int(originalMessage.Partition)).
		Str(topicKey, consumer.Configuration.Topic).
		Int(organizationKey, int(*parsedMessage.Organization)).
		Str(clusterKey, string(*parsedMessage.ClusterName)).
		Msg(event)
}

func logUnparsedMessageError(consumer *KafkaConsumer, originalMessage *sarama.ConsumerMessage, event string, err error) {
	log.Error().
		Int(offsetKey, int(originalMessage.Offset)).
		Str(topicKey, consumer.Configuration.Topic).
		Err(err).
		Msg(event)
}

func logMessageError(consumer *KafkaConsumer, originalMessage *sarama.ConsumerMessage, parsedMessage incomingMessage, event string, err error) {
	log.Error().
		Int(offsetKey, int(originalMessage.Offset)).
		Str(topicKey, consumer.Configuration.Topic).
		Int(organizationKey, int(*parsedMessage.Organization)).
		Str(clusterKey, string(*parsedMessage.ClusterName)).
		Err(err).
		Msg(event)
}

// ProcessMessage processes an incoming message
func (consumer *KafkaConsumer) ProcessMessage(msg *sarama.ConsumerMessage) error {
	tStart := time.Now()

	log.Info().Int(offsetKey, int(msg.Offset)).Str(topicKey, consumer.Configuration.Topic).Str(groupKey, consumer.Configuration.Group).Msg("Consumed")
	message, err := parseMessage(msg.Value)
	if err != nil {
		logUnparsedMessageError(consumer, msg, "Error parsing message from Kafka", err)
		return err
	}

	logMessageInfo(consumer, msg, message, "Read")
	tRead := time.Now()

	if consumer.Configuration.OrgWhitelistEnabled {
		logMessageInfo(consumer, msg, message, "Checking organization ID against whitelist")

		if ok := organizationAllowed(consumer, *message.Organization); !ok {
			const cause = "organization ID is not whitelisted"
			// now we have all required information about the incoming message,
			// the right time to record structured log entry
			logMessageError(consumer, msg, message, cause, err)
			return errors.New(cause)
		}

		logMessageInfo(consumer, msg, message, "Organization whitelisted")
	} else {
		logMessageInfo(consumer, msg, message, "Organization whitelisting disabled")
	}
	tWhitelisted := time.Now()

	reportAsStr, err := json.Marshal(*message.Report)
	if err != nil {
		logMessageError(consumer, msg, message, "Error marshalling report", err)
		return err
	}

	logMessageInfo(consumer, msg, message, "Marshalled")
	tMarshalled := time.Now()

	lastCheckedTime, err := time.Parse(time.RFC3339Nano, message.LastChecked)
	if err != nil {
		logMessageError(consumer, msg, message, "Error parsing date from message", err)
		return err
	}

	lastCheckedTimestampLagMinutes := time.Now().Sub(lastCheckedTime).Minutes()
	if lastCheckedTimestampLagMinutes < 0 {
		logMessageError(consumer, msg, message, "got a message from the future", nil)
	}

	metrics.LastCheckedTimestampLagMinutes.Observe(lastCheckedTimestampLagMinutes)

	logMessageInfo(consumer, msg, message, "Time ok")
	tTimeCheck := time.Now()

	err = consumer.Storage.WriteReportForCluster(
		*message.Organization,
		*message.ClusterName,
		types.ClusterReport(reportAsStr),
		lastCheckedTime,
	)
	if err != nil {
		logMessageError(consumer, msg, message, "Error writing report to database", err)
		return err
	}
	logMessageInfo(consumer, msg, message, "Stored")
	tStored := time.Now()

	duration := tRead.Sub(tStart)
	log.Info().Int64("duration", duration.Microseconds()).Int64(offsetKey, msg.Offset).Msg("read")
	duration = tWhitelisted.Sub(tRead)
	log.Info().Int64("duration", duration.Microseconds()).Int64(offsetKey, msg.Offset).Msg("whitelisting")
	duration = tMarshalled.Sub(tWhitelisted)
	log.Info().Int64("duration", duration.Microseconds()).Int64(offsetKey, msg.Offset).Msg("marshalling")
	duration = tTimeCheck.Sub(tMarshalled)
	log.Info().Int64("duration", duration.Microseconds()).Int64(offsetKey, msg.Offset).Msg("time_check")
	duration = tStored.Sub(tTimeCheck)
	log.Info().Int64("duration", duration.Microseconds()).Int64(offsetKey, msg.Offset).Msg("db_store")
	// message has been parsed and stored into storage
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
