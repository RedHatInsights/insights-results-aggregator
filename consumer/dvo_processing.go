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

	"github.com/RedHatInsights/insights-results-aggregator/metrics"
	"github.com/RedHatInsights/insights-results-aggregator/producer"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/Shopify/sarama"
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
		logUnparsedMessageError(consumer, msg, "Error parsing message from Kafka", err)
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

func (DVORulesProcessor) processMessage(consumer *KafkaConsumer, msg *sarama.ConsumerMessage) (types.RequestID, incomingMessage, error) {
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
	// log durations for every message consumption steps
	logDuration(tStart, tRead, msg.Offset, "read")
	return message.RequestID, message, nil
}

func (DVORulesProcessor) shouldProcess(consumer *KafkaConsumer, consumed *sarama.ConsumerMessage, parsed *incomingMessage) error {
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
