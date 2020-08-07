// Copyright 2020 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package consumer

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/Shopify/sarama"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/metrics"
	"github.com/RedHatInsights/insights-results-aggregator/producer"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// Report represents report send in a message consumed from any broker
type Report map[string]*json.RawMessage

// incomingMessage is representation of message consumed from any broker
type incomingMessage struct {
	Organization *types.OrgID       `json:"OrgID"`
	ClusterName  *types.ClusterName `json:"ClusterName"`
	Report       *Report            `json:"Report"`
	// LastChecked is a date in format "2020-01-23T16:15:59.478901889Z"
	LastChecked string          `json:"LastChecked"`
	RequestID   types.RequestID `json:"RequestId"`
}

// HandleMessage handles the message and does all logging, metrics, etc
func (consumer *KafkaConsumer) HandleMessage(msg *sarama.ConsumerMessage) {
	log.Info().
		Int64(offsetKey, msg.Offset).
		Int32(partitionKey, msg.Partition).
		Str(topicKey, msg.Topic).
		Time("message_timestamp", msg.Timestamp).
		Msgf("started processing message")

	metrics.ConsumedMessages.Inc()

	startTime := time.Now()
	requestID, err := consumer.ProcessMessage(msg)
	timeAfterProcessingMessage := time.Now()
	messageProcessingDuration := timeAfterProcessingMessage.Sub(startTime).Seconds()

	consumer.updatePayloadTracker(requestID, startTime, producer.StatusReceived)
	consumer.updatePayloadTracker(requestID, timeAfterProcessingMessage, producer.StatusMessageProcessed)

	log.Info().
		Int64(offsetKey, msg.Offset).
		Int32(partitionKey, msg.Partition).
		Str(topicKey, msg.Topic).
		Msgf("processing of message took '%v' seconds", messageProcessingDuration)

	// Something went wrong while processing the message.
	if err != nil {
		metrics.FailedMessagesProcessingTime.Observe(messageProcessingDuration)
		metrics.ConsumingErrors.Inc()

		log.Error().Err(err).Msg("Error processing message consumed from Kafka")
		consumer.numberOfErrorsConsumingMessages++

		if err := consumer.Storage.WriteConsumerError(msg, err); err != nil {
			log.Error().Err(err).Msg("Unable to write consumer error to storage")
		}

		consumer.updatePayloadTracker(requestID, time.Now(), producer.StatusError)
	} else {
		// The message was processed successfully.
		metrics.SuccessfulMessagesProcessingTime.Observe(messageProcessingDuration)
		consumer.numberOfSuccessfullyConsumedMessages++

		consumer.updatePayloadTracker(requestID, time.Now(), producer.StatusSuccess)
	}

	totalMessageDuration := time.Since(startTime)
	log.Info().Int64(durationKey, totalMessageDuration.Milliseconds()).Int64(offsetKey, msg.Offset).Msg("Message consumed")
}

// updatePayloadTracker
func (consumer KafkaConsumer) updatePayloadTracker(requestID types.RequestID, timestamp time.Time, status string) {
	err := consumer.payloadTrackerProducer.TrackPayload(requestID, timestamp, status)
	if err != nil {
		log.Warn().Msgf(`Unable to send "%s" update to Payload Tracker service`, status)
	}
}

// ProcessMessage processes an incoming message
func (consumer *KafkaConsumer) ProcessMessage(msg *sarama.ConsumerMessage) (types.RequestID, error) {
	tStart := time.Now()

	log.Info().Int(offsetKey, int(msg.Offset)).Str(topicKey, consumer.Configuration.Topic).Str(groupKey, consumer.Configuration.Group).Msg("Consumed")
	message, err := parseMessage(msg.Value)
	if err != nil {
		logUnparsedMessageError(consumer, msg, "Error parsing message from Kafka", err)
		return message.RequestID, err
	}

	logMessageInfo(consumer, msg, message, "Read")
	tRead := time.Now()

	if consumer.Configuration.OrgAllowlistEnabled {
		logMessageInfo(consumer, msg, message, "Checking organization ID against allow list")

		if ok := organizationAllowed(consumer, *message.Organization); !ok {
			const cause = "organization ID is not in allow list"
			// now we have all required information about the incoming message,
			// the right time to record structured log entry
			logMessageError(consumer, msg, message, cause, err)
			return message.RequestID, errors.New(cause)
		}

		logMessageInfo(consumer, msg, message, "Organization is in allow list")
	} else {
		logMessageInfo(consumer, msg, message, "Organization allow listing disabled")
	}
	tAllowlisted := time.Now()

	reportAsStr, err := json.Marshal(*message.Report)
	if err != nil {
		logMessageError(consumer, msg, message, "Error marshalling report", err)
		return message.RequestID, err
	}

	logMessageInfo(consumer, msg, message, "Marshalled")
	tMarshalled := time.Now()

	lastCheckedTime, err := time.Parse(time.RFC3339Nano, message.LastChecked)
	if err != nil {
		logMessageError(consumer, msg, message, "Error parsing date from message", err)
		return message.RequestID, err
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
		types.KafkaOffset(msg.Offset),
	)
	if err != nil {
		if err == types.ErrOldReport {
			logMessageInfo(consumer, msg, message, "Skipping because a more recent report already exists for this cluster")
			return message.RequestID, nil
		}

		logMessageError(consumer, msg, message, "Error writing report to database", err)
		return message.RequestID, err
	}
	logMessageInfo(consumer, msg, message, "Stored")
	tStored := time.Now()

	// log durations for every message consumption steps
	logDuration(tStart, tRead, msg.Offset, "read")
	logDuration(tRead, tAllowlisted, msg.Offset, "org_filtering")
	logDuration(tAllowlisted, tMarshalled, msg.Offset, "marshalling")
	logDuration(tMarshalled, tTimeCheck, msg.Offset, "time_check")
	logDuration(tTimeCheck, tStored, msg.Offset, "db_store")

	// message has been parsed and stored into storage
	return message.RequestID, nil
}

// organizationAllowed checks whether the given organization is on allow list or not
func organizationAllowed(consumer *KafkaConsumer, orgID types.OrgID) bool {
	allowList := consumer.Configuration.OrgAllowlist
	if allowList == nil {
		return false
	}

	orgAllowed := allowList.Contains(orgID)

	return orgAllowed
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
