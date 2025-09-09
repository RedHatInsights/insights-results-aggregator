// Copyright 2023 Red Hat, Inc
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

	"github.com/RedHatInsights/insights-results-aggregator/producer"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/rs/zerolog/log"

	"github.com/IBM/sarama"
	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/google/uuid"
)

// DVORulesProcessor satisfies MessageProcessor interface
type DVORulesProcessor struct {
}

func (DVORulesProcessor) deserializeMessage(messageValue []byte) (incomingMessage, error) {
	var deserialized incomingMessage

	received, err := DecompressMessage(messageValue)
	if err != nil {
		return deserialized, err
	}

	err = json.Unmarshal(received, &deserialized)
	if err != nil {
		return deserialized, err
	}
	if deserialized.Organization == nil {
		return deserialized, errors.New("missing required attribute 'OrgID'")
	}
	if deserialized.ClusterName == nil {
		return deserialized, errors.New("missing required attribute 'ClusterName'")
	}
	_, err = uuid.Parse(string(*deserialized.ClusterName))
	if err != nil {
		return deserialized, errors.New("cluster name is not a UUID")
	}
	if deserialized.DvoMetrics == nil {
		return deserialized, errors.New("missing required attribute 'Metrics'")
	}
	return deserialized, nil
}

// parseMessage is the entry point for parsing the received message.
// It should be the first method called within ProcessMessage in order
// to convert the message into a struct that can be worked with
func (DVORulesProcessor) parseMessage(consumer *KafkaConsumer, msg *sarama.ConsumerMessage) (incomingMessage, error) {
	message, err := consumer.MessageProcessor.deserializeMessage(msg.Value)
	if err != nil {
		consumer.logMsgForFurtherAnalysis(msg)
		return message, err
	}
	consumer.updatePayloadTracker(message.RequestID, time.Now(), message.Organization, message.Account, producer.StatusReceived)

	if err := consumer.MessageProcessor.shouldProcess(consumer, msg, &message); err != nil {
		return message, err
	}
	err = parseDVOContent(&message)
	if err != nil {
		consumer.logReportStructureError(err, msg)
		return message, err
	}

	return message, nil
}

func (processor DVORulesProcessor) processMessage(consumer *KafkaConsumer, msg *sarama.ConsumerMessage) (types.RequestID, incomingMessage, error) {
	return commonProcessMessage(consumer, msg, processor.storeInDB)
}

func (DVORulesProcessor) shouldProcess(_ *KafkaConsumer, _ *sarama.ConsumerMessage, parsed *incomingMessage) error {
	rawMetrics := *parsed.DvoMetrics
	if len(rawMetrics) == 0 {
		log.Debug().Msg("The 'Metrics' part of the JSON is empty. This message will be skipped")
		return types.ErrEmptyReport
	}
	if _, found := rawMetrics["workload_recommendations"]; !found {
		return fmt.Errorf("improper report structure, missing key with name 'workload_recommendations'")
	}
	return nil
}

// parseDVOContent verifies the content of the DVO structure and parses it into
// the relevant parts of the incomingMessage structure
func parseDVOContent(message *incomingMessage) error {
	return json.Unmarshal(*((*message.DvoMetrics)["workload_recommendations"]), &message.ParsedWorkloads)
}

func (DVORulesProcessor) storeInDB(consumer *KafkaConsumer, msg *sarama.ConsumerMessage, message incomingMessage) (types.RequestID, incomingMessage, error) {
	tStart := time.Now()
	lastCheckedTime, err := consumer.retrieveLastCheckedTime(msg, &message)
	if err != nil {
		return message.RequestID, message, err
	}
	tTimeCheck := time.Now()
	logDuration(tStart, tTimeCheck, msg.Offset, "time_check")

	reportAsBytes, err := json.Marshal(*message.DvoMetrics)
	if err != nil {
		logMessageError(consumer, msg, &message, "Error marshalling report", err)
		return message.RequestID, message, err
	}

	err = consumer.writeDVOReport(msg, message, reportAsBytes, lastCheckedTime)
	if err != nil {
		return message.RequestID, message, err
	}
	tStored := time.Now()
	logDuration(tTimeCheck, tStored, msg.Offset, "db_store_report")
	return message.RequestID, message, nil
}

func (consumer *KafkaConsumer) writeDVOReport(
	msg *sarama.ConsumerMessage, message incomingMessage,
	reportAsBytes []byte, lastCheckedTime time.Time,
) error {
	if dvoStorage, ok := consumer.Storage.(storage.DVORecommendationsStorage); ok {
		// timestamp when the report is about to be written into database
		storedAtTime := time.Now()

		err := dvoStorage.WriteReportForCluster(
			*message.Organization,
			*message.ClusterName,
			types.ClusterReport(reportAsBytes),
			message.ParsedWorkloads,
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
