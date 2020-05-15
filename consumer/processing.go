package consumer

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/Shopify/sarama"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/metrics"
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
	LastChecked string `json:"LastChecked"`
}

func (consumer *KafkaConsumer) handleMessage(msg *sarama.ConsumerMessage) {
	log.Info().
		Int64(offsetKey, msg.Offset).
		Int32(partitionKey, msg.Partition).
		Str(topicKey, msg.Topic).
		Time("message_timestamp", msg.Timestamp).
		Msgf("started processing message")

	metrics.ConsumedMessages.Inc()

	startTime := time.Now()
	err := consumer.ProcessMessage(msg)
	messageProcessingDuration := time.Since(startTime)

	log.Info().
		Int64(offsetKey, msg.Offset).
		Int32(partitionKey, msg.Partition).
		Str(topicKey, msg.Topic).
		Msgf("processing of message took '%v' seconds", messageProcessingDuration.Seconds())

	if err != nil {
		metrics.FailedMessagesProcessingTime.Observe(messageProcessingDuration.Seconds())
		metrics.ConsumingErrors.Inc()

		log.Error().Err(err).Msg("Error processing message consumed from Kafka")
		consumer.numberOfErrorsConsumingMessages++

		if err := consumer.Storage.WriteConsumerError(msg, err); err != nil {
			log.Error().Err(err).Msg("Unable to write consumer error to storage")
		}
	} else {
		metrics.SuccessfulMessagesProcessingTime.Observe(messageProcessingDuration.Seconds())
		consumer.numberOfSuccessfullyConsumedMessages++
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime)
	log.Info().Int64(durationKey, duration.Milliseconds()).Int64(offsetKey, msg.Offset).Msg("Message consumed")
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
		types.KafkaOffset(msg.Offset),
	)
	if err != nil {
		if err == types.ErrOldReport {
			logMessageInfo(consumer, msg, message, "Skipping because a more recent report already exists for this cluster")
			return nil
		}

		logMessageError(consumer, msg, message, "Error writing report to database", err)
		return err
	}
	logMessageInfo(consumer, msg, message, "Stored")
	tStored := time.Now()

	// log durations for every message consumption steps
	logDuration(tStart, tRead, msg.Offset, "read")
	logDuration(tRead, tWhitelisted, msg.Offset, "whitelisting")
	logDuration(tWhitelisted, tMarshalled, msg.Offset, "marshalling")
	logDuration(tMarshalled, tTimeCheck, msg.Offset, "time_check")
	logDuration(tTimeCheck, tStored, msg.Offset, "db_store")

	// message has been parsed and stored into storage
	return nil
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
