// Copyright 2020, 2021, 2022, 2023 Red Hat, Inc
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
	"fmt"
	"time"

	"github.com/RedHatInsights/insights-results-aggregator/storage"

	"github.com/IBM/sarama"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/metrics"
	"github.com/RedHatInsights/insights-results-aggregator/producer"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const (
	improperIncomeMessageError = "Deserialized report read from message with improper structure "
	unexpectedStorageType      = "Unexpected storage type"

	reportAttributeSystem        = "system"
	reportAttributeReports       = "reports"
	reportAttributeInfo          = "info"
	reportAttributeFingerprints  = "fingerprints"
	reportAttributeMetadata      = "analysis_metadata"
	numberOfExpectedKeysInReport = 3 // Number of items in expectedKeysInReport
)

var (
	expectedKeysInReport = []string{
		reportAttributeFingerprints, reportAttributeReports, reportAttributeSystem,
	}
)

// MessageProcessor offers the interface for processing a received message
type MessageProcessor interface {
	deserializeMessage(messageValue []byte) (incomingMessage, error)
	parseMessage(consumer *KafkaConsumer, msg *sarama.ConsumerMessage) (incomingMessage, error)
	processMessage(consumer *KafkaConsumer, msg *sarama.ConsumerMessage) (types.RequestID, incomingMessage, error)
	shouldProcess(consumer *KafkaConsumer, consumed *sarama.ConsumerMessage, parsed *incomingMessage) error
}

// Report represents report sent in a message consumed from any broker
type Report map[string]*json.RawMessage

// DvoMetrics represents DVO workload recommendations received as part
// of the incoming message
type DvoMetrics map[string]*json.RawMessage

type system struct {
	Hostname string `json:"hostname"`
}

// incomingMessage is representation of message consumed from any broker
type incomingMessage struct {
	Organization *types.OrgID       `json:"OrgID"`
	Account      *types.Account     `json:"AccountNumber"`
	ClusterName  *types.ClusterName `json:"ClusterName"`
	Report       *Report            `json:"Report"`
	DvoMetrics   *DvoMetrics        `json:"Metrics"`
	// LastChecked is a date in format "2020-01-23T16:15:59.478901889Z"
	LastChecked     string              `json:"LastChecked"`
	Version         types.SchemaVersion `json:"Version"`
	RequestID       types.RequestID     `json:"RequestId"`
	Metadata        types.Metadata      `json:"Metadata"`
	ParsedHits      []types.ReportItem
	ParsedInfo      []types.InfoItem
	ParsedWorkloads []types.WorkloadRecommendation
}

var currentSchemaVersion = types.AllowedVersions{
	types.SchemaVersion(1): struct{}{},
	types.SchemaVersion(2): struct{}{},
}

// HandleMessage handles the message and does all logging, metrics, etc.
//
// Log message is written for every step made during processing, but in order to
// reduce amount of messages sent to ElasticSearch, most messages are produced
// only when log level is set to DEBUG.
//
// A typical example which log messages are produced w/o DEBUG log level during
// processing:
//
// 1:26PM INF started processing message message_timestamp=2023-07-26T13:26:54+02:00 offset=7 partition=0 topic=ccx.ocp.results
// 1:26PM INF Consumed group=aggregator offset=7 topic=ccx.ocp.results
// 1:26PM INF Read cluster=5d5892d3-1f74-4ccf-91af-548dfc9767aa offset=7 organization=11789772 partition=0 request ID=missing topic=ccx.ocp.results version=2
// 1:26PM WRN Received data with unexpected version. cluster=5d5892d3-1f74-4ccf-91af-548dfc9767aa offset=7 organization=11789772 partition=0 topic=ccx.ocp.results version=2
// 1:26PM INF Stored info report cluster=5d5892d3-1f74-4ccf-91af-548dfc9767aa offset=7 organization=11789772 partition=0 request ID=missing topic=ccx.ocp.results version=2
// 1:26PM WRN request ID is missing, null or empty Operation=TrackPayload
// 1:26PM INF Message consumed duration=3 offset=7
//
// When log level is set to DEBUG, many log messages useful for debugging are
// generated as well:
//
// 2:53PM INF started processing message message_timestamp=2023-07-26T14:53:32+02:00 offset=8 partition=0 topic=ccx.ocp.results
// 2:53PM INF Consumed group=aggregator offset=8 topic=ccx.ocp.results
// 2:53PM INF Read cluster=5d5892d3-1f74-4ccf-91af-548dfc9767aa offset=8 organization=11789772 partition=0 request ID=missing topic=ccx.ocp.results version=2
// 2:53PM WRN Received data with unexpected version. cluster=5d5892d3-1f74-4ccf-91af-548dfc9767aa offset=8 organization=11789772 partition=0 topic=ccx.ocp.results version=2
// 2:53PM DBG Organization allow listing disabled cluster=5d5892d3-1f74-4ccf-91af-548dfc9767aa offset=8 organization=11789772 partition=0 request ID=missing topic=ccx.ocp.results version=2
// 2:53PM DBG Marshalled cluster=5d5892d3-1f74-4ccf-91af-548dfc9767aa offset=8 organization=11789772 partition=0 request ID=missing topic=ccx.ocp.results version=2
// 2:53PM DBG Time ok cluster=5d5892d3-1f74-4ccf-91af-548dfc9767aa offset=8 organization=11789772 partition=0 request ID=missing topic=ccx.ocp.results version=2
// 2:53PM DBG Stored report cluster=5d5892d3-1f74-4ccf-91af-548dfc9767aa offset=8 organization=11789772 partition=0 request ID=missing topic=ccx.ocp.results version=2
// 2:53PM DBG Stored recommendations cluster=5d5892d3-1f74-4ccf-91af-548dfc9767aa offset=8 organization=11789772 partition=0 request ID=missing topic=ccx.ocp.results version=2
// 2:53PM DBG rule hits for 11789772.5d5892d3-1f74-4ccf-91af-548dfc9767aa (request ID missing):
//
//	rule: ccx_rules_ocp.external.rules.nodes_requirements_check.report; error key: NODES_MINIMUM_REQUIREMENTS_NOT_MET
//	rule: ccx_rules_ocp.external.bug_rules.bug_1766907.report; error key: BUGZILLA_BUG_1766907
//	rule: ccx_rules_ocp.external.rules.nodes_kubelet_version_check.report; error key: NODE_KUBELET_VERSION
//	rule: ccx_rules_ocp.external.rules.samples_op_failed_image_import_check.report; error key: SAMPLES_FAILED_IMAGE_IMPORT_ERR
//	rule: ccx_rules_ocp.external.rules.cluster_wide_proxy_auth_check.report; error key: AUTH_OPERATOR_PROXY_ERROR
//
// 2:53PM DBG rule hits for 11789772.5d5892d3-1f74-4ccf-91af-548dfc9767aa (request ID missing):
//
//	rule: ccx_rules_ocp.external.rules.nodes_requirements_check.report; error key: NODES_MINIMUM_REQUIREMENTS_NOT_MET
//	rule: ccx_rules_ocp.external.bug_rules.bug_1766907.report; error key: BUGZILLA_BUG_1766907
//	rule: ccx_rules_ocp.external.rules.nodes_kubelet_version_check.report; error key: NODE_KUBELET_VERSION
//	rule: ccx_rules_ocp.external.rules.samples_op_failed_image_import_check.report; error key: SAMPLES_FAILED_IMAGE_IMPORT_ERR
//	rule: ccx_rules_ocp.external.rules.cluster_wide_proxy_auth_check.report; error key: AUTH_OPERATOR_PROXY_ERROR
//
// 2:53PM INF Stored info report cluster=5d5892d3-1f74-4ccf-91af-548dfc9767aa offset=8 organization=11789772 partition=0 request ID=missing topic=ccx.ocp.results version=2
// 2:53PM DBG read duration=2287 offset=8
// 2:53PM DBG org_filtering duration=440 offset=8
// 2:53PM DBG marshalling duration=2023 offset=8
// 2:53PM DBG time_check duration=120 offset=8
// 2:53PM DBG db_store_report duration=119 offset=8
// 2:53PM DBG db_store_recommendations duration=11 offset=8
// 2:53PM DBG db_store_info_report duration=102 offset=8
// 2:53PM WRN request ID is missing, null or empty Operation=TrackPayload
// 2:53PM WRN request ID is missing, null or empty Operation=TrackPayload
// 2:53PM DBG processing of message took '0.005895183' seconds offset=8 partition=0 topic=ccx.ocp.results
// 2:53PM WRN request ID is missing, null or empty Operation=TrackPayload
// 2:53PM INF Message consumed duration=6 offset=8
func (consumer *KafkaConsumer) HandleMessage(msg *sarama.ConsumerMessage) error {
	log.Info().
		Int64(offsetKey, msg.Offset).
		Int32(partitionKey, msg.Partition).
		Str(topicKey, msg.Topic).
		Time("message_timestamp", msg.Timestamp).
		Msgf("started processing message")

	metrics.ConsumedMessages.Inc()

	startTime := time.Now()
	requestID, message, err := consumer.MessageProcessor.processMessage(consumer, msg)
	timeAfterProcessingMessage := time.Now()
	messageProcessingDuration := timeAfterProcessingMessage.Sub(startTime).Seconds()

	consumer.updatePayloadTracker(requestID, timeAfterProcessingMessage, message.Organization, message.Account, producer.StatusMessageProcessed)

	log.Debug().
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
		if ocpStorage, ok := consumer.Storage.(storage.OCPRecommendationsStorage); ok {
			if err := ocpStorage.WriteConsumerError(msg, err); err != nil {
				log.Error().Err(err).Msg("Unable to write consumer error to storage")
			}
		}
		consumer.sendDeadLetter(msg)

		consumer.updatePayloadTracker(requestID, time.Now(), message.Organization, message.Account, producer.StatusError)
	} else {
		// The message was processed successfully.
		metrics.SuccessfulMessagesProcessingTime.Observe(messageProcessingDuration)
		consumer.numberOfSuccessfullyConsumedMessages++

		consumer.updatePayloadTracker(requestID, time.Now(), message.Organization, message.Account, producer.StatusSuccess)
	}

	totalMessageDuration := time.Since(startTime)
	log.Info().Int64(durationKey, totalMessageDuration.Milliseconds()).Int64(offsetKey, msg.Offset).Msg("Message consumed")
	return err
}

// updatePayloadTracker
func (consumer KafkaConsumer) updatePayloadTracker(
	requestID types.RequestID,
	timestamp time.Time,
	orgID *types.OrgID,
	account *types.Account,
	status string,
) {
	if consumer.payloadTrackerProducer != nil {
		err := consumer.payloadTrackerProducer.TrackPayload(requestID, timestamp, orgID, account, status)
		if err != nil {
			log.Warn().Msgf(`Unable to send "%s" update to Payload Tracker service`, status)
		}
	}
}

// sendDeadLetter - sends unprocessed message to dead letter queue
func (consumer KafkaConsumer) sendDeadLetter(msg *sarama.ConsumerMessage) {
	if consumer.deadLetterProducer != nil {
		if err := consumer.deadLetterProducer.SendDeadLetter(msg); err != nil {
			log.Error().Err(err).Msg("Failed to load message to dead letter queue")
		}
	}
}

// checkMessageVersion - verifies incoming data's version is expected
func checkMessageVersion(consumer *KafkaConsumer, message *incomingMessage, msg *sarama.ConsumerMessage) {
	if _, ok := currentSchemaVersion[message.Version]; !ok {
		warning := fmt.Sprintf("Received data with unexpected version %d.", message.Version)
		logMessageWarning(consumer, msg, message, warning)
	}
}

// checkMessageOrgInAllowList - checks up incoming data's OrganizationID against allowed orgs list
func checkMessageOrgInAllowList(consumer *KafkaConsumer, message *incomingMessage, msg *sarama.ConsumerMessage) (bool, string) {
	if consumer.Configuration.OrgAllowlistEnabled {
		logMessageInfo(consumer, msg, message, "Checking organization ID against allow list")

		if ok := organizationAllowed(consumer, *message.Organization); !ok {
			const cause = "organization ID is not in allow list"
			return false, cause
		}

		logMessageDebug(consumer, msg, message, "Organization is in allow list")
	} else {
		logMessageDebug(consumer, msg, message, "Organization allow listing disabled")
	}
	return true, ""
}

func (consumer *KafkaConsumer) logMsgForFurtherAnalysis(msg *sarama.ConsumerMessage) {
	if consumer.Configuration.DisplayMessageWithWrongStructure {
		log.Info().Str("unparsed message", string(msg.Value)).Msg("Message for further analysis")
	}
}

func (consumer *KafkaConsumer) logReportStructureError(err error, msg *sarama.ConsumerMessage) {
	if consumer.Configuration.DisplayMessageWithWrongStructure {
		log.Err(err).Str("unparsed message", string(msg.Value)).Msg(improperIncomeMessageError)
	} else {
		log.Err(err).Msg(improperIncomeMessageError)
	}
}

func (consumer *KafkaConsumer) retrieveLastCheckedTime(msg *sarama.ConsumerMessage, parsedMsg *incomingMessage) (time.Time, error) {
	lastCheckedTime, err := time.Parse(time.RFC3339Nano, parsedMsg.LastChecked)
	if err != nil {
		logMessageError(consumer, msg, parsedMsg, "Error parsing date from message", err)
		return time.Time{}, err
	}

	lastCheckedTimestampLagMinutes := time.Since(lastCheckedTime).Minutes()
	if lastCheckedTimestampLagMinutes < 0 {
		logMessageError(consumer, msg, parsedMsg, "got a message from the future", nil)
	}

	metrics.LastCheckedTimestampLagMinutes.Observe(lastCheckedTimestampLagMinutes)

	logMessageDebug(consumer, msg, parsedMsg, "Time ok")
	return lastCheckedTime, nil
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

type storeInDBFunction func(
	consumer *KafkaConsumer,
	msg *sarama.ConsumerMessage,
	message incomingMessage) (types.RequestID, incomingMessage, error)

// commonProcessMessage is used by both DVO and OCP message processors as they share
// some steps in common. However, storing the DVO and OCP reports in the DB is done
// differently, so it was needed to introduce a storeInDBFunction type
func commonProcessMessage(
	consumer *KafkaConsumer,
	msg *sarama.ConsumerMessage,
	storeInDBFunction storeInDBFunction,
) (types.RequestID, incomingMessage, error) {
	tStart := time.Now()
	log.Info().Int(offsetKey, int(msg.Offset)).Str(topicKey, consumer.Configuration.Topic).Str(groupKey, consumer.Configuration.Group).Msg("Consumed")
	message, err := consumer.MessageProcessor.parseMessage(consumer, msg)
	if err != nil {
		if errors.Is(err, types.ErrEmptyReport) {
			logMessageInfo(consumer, msg, &message, "This message has an empty report and will not be processed further")
			metrics.SkippedEmptyReports.Inc()
			return message.RequestID, message, nil
		}
		return message.RequestID, message, err
	}
	logMessageInfo(consumer, msg, &message, "Read")
	tRead := time.Now()
	logDuration(tStart, tRead, msg.Offset, "read")

	checkMessageVersion(consumer, &message, msg)

	if ok, cause := checkMessageOrgInAllowList(consumer, &message, msg); !ok {
		err := errors.New(cause)
		logMessageError(consumer, msg, &message, cause, err)
		return message.RequestID, message, err
	}

	tAllowlisted := time.Now()
	logDuration(tRead, tAllowlisted, msg.Offset, "org_filtering")

	logMessageDebug(consumer, msg, &message, "Marshalled")
	tMarshalled := time.Now()
	logDuration(tAllowlisted, tMarshalled, msg.Offset, "marshalling")

	return storeInDBFunction(consumer, msg, message)
}
