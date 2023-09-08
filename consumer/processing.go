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

	"github.com/Shopify/sarama"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/metrics"
	"github.com/RedHatInsights/insights-results-aggregator/producer"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const (
	improperIncomeMessageError = "Deserialized report read from message with improper structure "

	reportAttributeSystem        = "system"
	reportAttributeReports       = "reports"
	reportAttributeInfo          = "info"
	reportAttributeFingerprints  = "fingerprints"
	numberOfExpectedKeysInReport = 4
)

var (
	expectedKeysInReport = [numberOfExpectedKeysInReport]string{
		reportAttributeFingerprints, reportAttributeInfo, reportAttributeReports, reportAttributeSystem,
	}
)

// Report represents report send in a message consumed from any broker
type Report map[string]*json.RawMessage

type system struct {
	Hostname string `json:"hostname"`
}

// incomingMessage is representation of message consumed from any broker
type incomingMessage struct {
	Organization *types.OrgID       `json:"OrgID"`
	Account      *types.Account     `json:"AccountNumber"`
	ClusterName  *types.ClusterName `json:"ClusterName"`
	Report       *Report            `json:"Report"`
	// LastChecked is a date in format "2020-01-23T16:15:59.478901889Z"
	LastChecked string              `json:"LastChecked"`
	Version     types.SchemaVersion `json:"Version"`
	RequestID   types.RequestID     `json:"RequestId"`
	Metadata    types.Metadata      `json:"Metadata"`
	ParsedHits  []types.ReportItem
	ParsedInfo  []types.InfoItem
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
	requestID, message, err := consumer.processMessage(msg)
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

		if err := consumer.Storage.WriteConsumerError(msg, err); err != nil {
			log.Error().Err(err).Msg("Unable to write consumer error to storage")
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
		logMessageWarning(consumer, msg, *message, warning)
	}
}

// checkMessageOrgInAllowList - checks up incoming data's OrganizationID against allowed orgs list
func checkMessageOrgInAllowList(consumer *KafkaConsumer, message *incomingMessage, msg *sarama.ConsumerMessage) (bool, string) {
	if consumer.Configuration.OrgAllowlistEnabled {
		logMessageInfo(consumer, msg, *message, "Checking organization ID against allow list")

		if ok := organizationAllowed(consumer, *message.Organization); !ok {
			const cause = "organization ID is not in allow list"
			return false, cause
		}

		logMessageDebug(consumer, msg, *message, "Organization is in allow list")
	} else {
		logMessageDebug(consumer, msg, *message, "Organization allow listing disabled")
	}
	return true, ""
}

func (consumer *KafkaConsumer) writeRecommendations(
	msg *sarama.ConsumerMessage, message incomingMessage, reportAsBytes []byte,
) (time.Time, error) {
	err := consumer.Storage.WriteRecommendationsForCluster(
		*message.Organization,
		*message.ClusterName,
		types.ClusterReport(reportAsBytes),
		types.Timestamp(time.Now().UTC().Format(time.RFC3339)),
	)
	if err != nil {
		logMessageError(consumer, msg, message, "Error writing recommendations to database", err)
		return time.Time{}, err
	}
	tStored := time.Now()
	logMessageDebug(consumer, msg, message, "Stored recommendations")
	logClusterInfo(&message)
	return tStored, nil
}

func (consumer *KafkaConsumer) writeInfoReport(
	msg *sarama.ConsumerMessage, message incomingMessage, infoStoredAtTime time.Time,
) error {
	// it is expected that message.ParsedInfo contains at least one item:
	// result from special INFO rule containing cluster version that is
	// used just in external data pipeline
	err := consumer.Storage.WriteReportInfoForCluster(
		*message.Organization,
		*message.ClusterName,
		message.ParsedInfo,
		infoStoredAtTime,
	)
	if err == types.ErrOldReport {
		logMessageInfo(consumer, msg, message, "Skipping because a more recent info report already exists for this cluster")
		return nil
	} else if err != nil {
		logMessageError(consumer, msg, message, "Error writing info report to database", err)
		return err
	}
	logMessageInfo(consumer, msg, message, "Stored info report")
	return nil
}

func (consumer *KafkaConsumer) logReportStructureError(err error, msg *sarama.ConsumerMessage) {
	if consumer.Configuration.DisplayMessageWithWrongStructure {
		log.Err(err).Msgf(improperIncomeMessageError+"%v", string(msg.Value))
	} else {
		log.Err(err).Msg(improperIncomeMessageError)
	}
}

func (consumer *KafkaConsumer) shouldProcess(consumed *sarama.ConsumerMessage, parsed *incomingMessage) bool {
	keepProcessing, err := checkReportStructure(*parsed.Report)
	if err != nil {
		consumer.logReportStructureError(err, consumed)
		return false
	}

	if !keepProcessing {
		logMessageInfo(consumer, consumed, *parsed, "This message will not be processed further")
		metrics.SkippedEmptyReports.Inc()
		return false
	}
	return true
}

// processMessage processes an incoming message
func (consumer *KafkaConsumer) processMessage(msg *sarama.ConsumerMessage) (types.RequestID, incomingMessage, error) {
	tStart := time.Now()

	log.Info().Int(offsetKey, int(msg.Offset)).Str(topicKey, consumer.Configuration.Topic).Str(groupKey, consumer.Configuration.Group).Msg("Consumed")

	message, process := consumer.parseMessage(msg, tStart)
	if !process {
		return message.RequestID, message, nil
	}
	logMessageInfo(consumer, msg, message, "Read")
	tRead := time.Now()

	checkMessageVersion(consumer, &message, msg)

	if ok, cause := checkMessageOrgInAllowList(consumer, &message, msg); !ok {
		err := errors.New(cause)
		logMessageError(consumer, msg, message, cause, err)
		return message.RequestID, message, err
	}

	tAllowlisted := time.Now()

	reportAsBytes, err := json.Marshal(*message.Report)
	if err != nil {
		logMessageError(consumer, msg, message, "Error marshalling report", err)
		return message.RequestID, message, err
	}

	logMessageDebug(consumer, msg, message, "Marshalled")
	tMarshalled := time.Now()

	lastCheckedTime, err := time.Parse(time.RFC3339Nano, message.LastChecked)
	if err != nil {
		logMessageError(consumer, msg, message, "Error parsing date from message", err)
		return message.RequestID, message, err
	}

	lastCheckedTimestampLagMinutes := time.Since(lastCheckedTime).Minutes()
	if lastCheckedTimestampLagMinutes < 0 {
		logMessageError(consumer, msg, message, "got a message from the future", nil)
	}

	metrics.LastCheckedTimestampLagMinutes.Observe(lastCheckedTimestampLagMinutes)

	logMessageDebug(consumer, msg, message, "Time ok")
	tTimeCheck := time.Now()

	// timestamp when the report is about to be written into database
	storedAtTime := time.Now()

	err = consumer.Storage.WriteReportForCluster(
		*message.Organization,
		*message.ClusterName,
		types.ClusterReport(reportAsBytes),
		message.ParsedHits,
		lastCheckedTime,
		message.Metadata.GatheredAt,
		storedAtTime,
		message.RequestID,
	)
	if err == types.ErrOldReport {
		logMessageInfo(consumer, msg, message, "Skipping because a more recent report already exists for this cluster")
		return message.RequestID, message, nil
	} else if err != nil {
		logMessageError(consumer, msg, message, "Error writing report to database", err)
		return message.RequestID, message, err
	}
	logMessageDebug(consumer, msg, message, "Stored report")
	tStored := time.Now()

	tRecommendationsStored, err := consumer.writeRecommendations(msg, message, reportAsBytes)
	if err != nil {
		return message.RequestID, message, err
	}

	logClusterInfo(&message)

	infoStoredAtTime := time.Now()
	if err := consumer.writeInfoReport(msg, message, infoStoredAtTime); err != nil {
		return message.RequestID, message, err
	}
	infoStored := time.Now()

	// log durations for every message consumption steps
	logDuration(tStart, tRead, msg.Offset, "read")
	logDuration(tRead, tAllowlisted, msg.Offset, "org_filtering")
	logDuration(tAllowlisted, tMarshalled, msg.Offset, "marshalling")
	logDuration(tMarshalled, tTimeCheck, msg.Offset, "time_check")
	logDuration(tTimeCheck, tStored, msg.Offset, "db_store_report")
	logDuration(tStored, tRecommendationsStored, msg.Offset, "db_store_recommendations")
	logDuration(infoStoredAtTime, infoStored, msg.Offset, "db_store_info_report")

	// message has been parsed and stored into storage
	return message.RequestID, message, nil
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

func verifySystemAttributeIsEmpty(r Report) bool {
	if r[reportAttributeSystem] != nil {
		var s system
		if err := json.Unmarshal(*r[reportAttributeSystem], &s); err != nil {
			return false
		}
		if s.Hostname != "" {
			return false
		}
	}
	return true
}

func verifyJSONArrayAttributeIsEmpty(attr string, r Report) bool {
	if val, exist := r[attr]; exist && val != nil {
		var arr []interface{}
		if err := json.Unmarshal(*val, &arr); err != nil {
			return false
		}
		if len(arr) != 0 {
			return false
		}
	}
	return true
}

// isReportWithEmptyAttributes checks if the report is empty, or if the attributes
// expected in the report, minus the analysis_metadata, are empty.
// If this function returns true, this report will not be processed further as it is
// PROBABLY the result of an archive that was not processed by insights-core.
// see https://github.com/RedHatInsights/insights-results-aggregator/issues/1834
func isReportWithEmptyAttributes(r Report, keysToCheck [numberOfExpectedKeysInReport]string) bool {
	for _, key := range keysToCheck {
		switch key {
		case reportAttributeSystem:
			if !verifySystemAttributeIsEmpty(r) {
				return false
			}
		case reportAttributeReports:
			if !verifyJSONArrayAttributeIsEmpty(reportAttributeReports, r) {
				return false
			}
		case reportAttributeFingerprints:
			if !verifyJSONArrayAttributeIsEmpty(reportAttributeFingerprints, r) {
				return false
			}
		case reportAttributeInfo:
			if !verifyJSONArrayAttributeIsEmpty(reportAttributeInfo, r) {
				return false
			}
		}
	}

	return true
}

// checkReportStructure tests if the report has correct structure
func checkReportStructure(r Report) (shouldProcess bool, err error) {
	// the structure is not well defined yet, so all we should do is to check if all keys are there

	// 'skips' key is now optional, we should not expect it anymore:
	// https://github.com/RedHatInsights/insights-results-aggregator/issues/1206
	keysFound := [numberOfExpectedKeysInReport]string{}
	keysNotFound := [numberOfExpectedKeysInReport]string{}

	// check if the structure contains all expected keys
	for _, expectedKey := range expectedKeysInReport {
		_, found := r[expectedKey]
		if found {
			keysFound[len(keysFound)-1] = expectedKey
		} else {
			keysNotFound[len(keysNotFound)-1] = expectedKey
		}
	}

	// empty reports mean that this message should not be processed further
	isEmpty := len(r) == 0 || isReportWithEmptyAttributes(r, keysFound)
	if isEmpty {
		log.Debug().Msg("Empty report or report with only empty attributes. Processing of this message will be skipped.")
		return false, nil
	}

	// report is not empty, and some keys have not been found -> malformed
	if len(keysNotFound) != 0 {
		return false, fmt.Errorf("improper report structure, missing key(s) with name '%v'", keysNotFound)
	}

	return true, nil
}

// deserializeMessage tries to parse incoming message and read all required attributes from it
func deserializeMessage(messageValue []byte) (incomingMessage, error) {
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
	return deserialized, nil
}

// parseReportContent verifies the content of the Report structure and parses it into
// the relevant parts of the incomingMessage structure
func parseReportContent(message *incomingMessage) error {
	err := json.Unmarshal(*((*message.Report)[reportAttributeReports]), &message.ParsedHits)
	if err != nil {
		return err
	}

	// it is expected that message.ParsedInfo contains at least one item:
	// result from special INFO rule containing cluster version that is
	// used just in external data pipeline
	err = json.Unmarshal(*((*message.Report)[reportAttributeInfo]), &message.ParsedInfo)
	if err != nil {
		return err
	}
	return nil
}

func (consumer *KafkaConsumer) parseMessage(msg *sarama.ConsumerMessage, tStart time.Time) (incomingMessage, bool) {
	message, err := deserializeMessage(msg.Value)
	if err != nil {
		logUnparsedMessageError(consumer, msg, "Error parsing message from Kafka", err)
		return message, false
	}

	consumer.updatePayloadTracker(message.RequestID, tStart, message.Organization, message.Account, producer.StatusReceived)

	if !consumer.shouldProcess(msg, &message) {
		return message, false
	}

	err = parseReportContent(&message)
	if err != nil {
		consumer.logReportStructureError(err, msg)
		return message, false
	}

	return message, true
}
