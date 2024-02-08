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

	"github.com/RedHatInsights/insights-results-aggregator/producer"
	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/Shopify/sarama"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// OCPRulesProcessor satisfies MessageProcessor interface
type OCPRulesProcessor struct {
}

// deserializeMessage tries to unmarshall the received message
// and read all required attributes from it
func (OCPRulesProcessor) deserializeMessage(messageValue []byte) (incomingMessage, error) {
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

func (consumer *KafkaConsumer) writeOCPReport(
	msg *sarama.ConsumerMessage, message incomingMessage,
	reportAsBytes []byte, lastCheckedTime time.Time,
) error {
	if ocpStorage, ok := consumer.Storage.(storage.OCPRecommendationsStorage); ok {
		// timestamp when the report is about to be written into database
		storedAtTime := time.Now()

		err := ocpStorage.WriteReportForCluster(
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
			logMessageInfo(consumer, msg, &message, "Skipping because a more recent report already exists for this cluster")
			return nil
		} else if err != nil {
			logMessageError(consumer, msg, &message, "Error writing report to database", err)
			return err
		}
		logMessageDebug(consumer, msg, &message, "Stored report")
		return nil
	}
	err := errors.New("report could not be stored")
	logMessageError(consumer, msg, &message, unexpectedStorageType, err)
	return err
}

func (consumer *KafkaConsumer) writeRecommendations(
	msg *sarama.ConsumerMessage, message incomingMessage, reportAsBytes []byte,
) (time.Time, error) {
	if ocpStorage, ok := consumer.Storage.(storage.OCPRecommendationsStorage); ok {
		err := ocpStorage.WriteRecommendationsForCluster(
			*message.Organization,
			*message.ClusterName,
			types.ClusterReport(reportAsBytes),
			types.Timestamp(time.Now().UTC().Format(time.RFC3339)),
		)
		if err != nil {
			logMessageError(consumer, msg, &message, "Error writing recommendations to database", err)
			return time.Time{}, err
		}
		tStored := time.Now()
		logMessageDebug(consumer, msg, &message, "Stored recommendations")
		return tStored, nil
	}
	logMessageError(consumer, msg, &message, unexpectedStorageType, errors.New("recommendation could not be stored"))
	return time.Time{}, nil
}

func (consumer *KafkaConsumer) writeInfoReport(
	msg *sarama.ConsumerMessage, message incomingMessage, infoStoredAtTime time.Time,
) error {
	// it is expected that message.ParsedInfo contains at least one item:
	// result from special INFO rule containing cluster version that is
	// used just in external data pipeline
	if ocpStorage, ok := consumer.Storage.(storage.OCPRecommendationsStorage); ok {
		err := ocpStorage.WriteReportInfoForCluster(
			*message.Organization,
			*message.ClusterName,
			message.ParsedInfo,
			infoStoredAtTime,
		)
		if errors.Is(err, types.ErrOldReport) {
			logMessageInfo(consumer, msg, &message, "Skipping because a more recent info report already exists for this cluster")
			return nil
		} else if err != nil {
			logMessageError(consumer, msg, &message, "Error writing info report to database", err)
			return err
		}
		logMessageInfo(consumer, msg, &message, "Stored info report")
		return nil
	}
	logMessageError(consumer, msg, &message, unexpectedStorageType, errors.New("info report could not be stored"))
	return nil
}

// processMessage processes an incoming message
func (processor OCPRulesProcessor) processMessage(consumer *KafkaConsumer, msg *sarama.ConsumerMessage) (types.RequestID, incomingMessage, error) {
	return commonProcessMessage(consumer, msg, processor.storeInDB)
}

func (OCPRulesProcessor) storeInDB(consumer *KafkaConsumer, msg *sarama.ConsumerMessage, message incomingMessage) (types.RequestID, incomingMessage, error) {
	tStart := time.Now()
	lastCheckedTime, err := consumer.retrieveLastCheckedTime(msg, &message)
	if err != nil {
		return message.RequestID, message, err
	}
	tTimeCheck := time.Now()
	logDuration(tStart, tTimeCheck, msg.Offset, "time_check")

	reportAsBytes, err := json.Marshal(*message.Report)
	if err != nil {
		logMessageError(consumer, msg, &message, "Error marshalling report", err)
		return message.RequestID, message, err
	}

	err = consumer.writeOCPReport(msg, message, reportAsBytes, lastCheckedTime)
	if err != nil {
		return message.RequestID, message, err
	}
	tStored := time.Now()
	logDuration(tTimeCheck, tStored, msg.Offset, "db_store_report")

	tRecommendationsStored, err := consumer.writeRecommendations(msg, message, reportAsBytes)
	if err != nil {
		return message.RequestID, message, err
	}
	logDuration(tStored, tRecommendationsStored, msg.Offset, "db_store_recommendations")

	// rule hits has been stored into database - time to log all these great info
	logClusterInfo(&message)

	infoStoredAtTime := time.Now()
	if err := consumer.writeInfoReport(msg, message, infoStoredAtTime); err != nil {
		return message.RequestID, message, err
	}
	infoStored := time.Now()
	logDuration(infoStoredAtTime, infoStored, msg.Offset, "db_store_info_report")

	return message.RequestID, message, nil
}

// shouldProcess determines if a parsed message should be processed further
func (OCPRulesProcessor) shouldProcess(consumer *KafkaConsumer, consumed *sarama.ConsumerMessage, parsed *incomingMessage) error {
	err := checkReportStructure(*parsed.Report)
	if err != nil {
		consumer.logReportStructureError(err, consumed)
		return err
	}
	return nil
}

func verifySystemAttributeIsEmpty(r Report) bool {
	var s system
	if err := json.Unmarshal(*r[reportAttributeSystem], &s); err != nil {
		return false
	}
	if s.Hostname != "" {
		return false
	}
	return true
}

// isReportWithEmptyAttributes checks if the report is empty, or if the attributes
// expected in the report, minus the analysis_metadata, are empty.
// If this function returns true, this report will not be processed further as it is
// PROBABLY the result of an archive that was not processed by insights-core.
// see https://github.com/RedHatInsights/insights-results-aggregator/issues/1834
func isReportWithEmptyAttributes(r Report) bool {
	// Create attribute checkers for each attribute
	for attr, attrData := range r {
		// we don't care about the analysis_metadata attribute
		if attr == reportAttributeMetadata {
			continue
		}
		// special handling for the system attribute, as it comes with data when empty
		if attr == reportAttributeSystem {
			if !verifySystemAttributeIsEmpty(r) {
				return false
			}
			continue
		}
		// Check if this attribute of the report is empty
		checker := JSONAttributeChecker{data: *attrData}
		if !checker.IsEmpty() {
			return false
		}
	}
	return true
}

// checkReportStructure tests if the report has correct structure
func checkReportStructure(r Report) error {
	// the structure is not well-defined yet, so all we should do is to check if all keys are there

	// 'skips' key is now optional, we should not expect it anymore:
	// https://github.com/RedHatInsights/insights-results-aggregator/issues/1206
	keysNotFound := make([]string, 0, numberOfExpectedKeysInReport)
	keysFound := 0
	// check if the structure contains all expected keys
	for _, expectedKey := range expectedKeysInReport {
		_, found := r[expectedKey]
		if !found {
			keysNotFound = append(keysNotFound, expectedKey)
		} else {
			keysFound++
		}
	}

	if keysFound == numberOfExpectedKeysInReport {
		return nil
	}

	// empty reports mean that this message should not be processed further
	isEmpty := len(r) == 0 || isReportWithEmptyAttributes(r)
	if isEmpty {
		log.Debug().Msg("Empty report or report with only empty attributes. Processing of this message will be skipped.")
		return types.ErrEmptyReport
	}

	// report is not empty, and some keys have not been found -> malformed
	if len(keysNotFound) != 0 {
		return fmt.Errorf("improper report structure, missing key(s) with name '%v'", keysNotFound)
	}

	return nil
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

// parseMessage is the entry point for parsing the received message.
// It should be the first method called within ProcessMessage in order
// to convert the message into a struct that can be worked with
func (OCPRulesProcessor) parseMessage(consumer *KafkaConsumer, msg *sarama.ConsumerMessage) (incomingMessage, error) {
	message, err := consumer.MessageProcessor.deserializeMessage(msg.Value)
	if err != nil {
		consumer.logMsgForFurtherAnalysis(msg)
		logUnparsedMessageError(consumer, msg, "Error parsing message from Kafka", err)
		return message, err
	}

	consumer.updatePayloadTracker(message.RequestID, time.Now(), message.Organization, message.Account, producer.StatusReceived)

	if err := consumer.MessageProcessor.shouldProcess(consumer, msg, &message); err != nil {
		return message, err
	}

	err = parseReportContent(&message)
	if err != nil {
		consumer.logReportStructureError(err, msg)
		return message, err
	}

	return message, nil
}
