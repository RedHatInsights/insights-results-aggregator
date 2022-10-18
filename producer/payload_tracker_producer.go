/*
Copyright © 2022 Red Hat, Inc.

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

// Package producer contains functions that can be used to produce (that is
// send) messages to properly configured Kafka broker.
package producer

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// PayloadTrackerProducer is a producer for payload tracker topic
type PayloadTrackerProducer struct {
	KafkaProducer KafkaProducer
	Configuration broker.Configuration
}

// NewPayloadTrackerProducer constructs producer for payload tracker topic.
// It is implemented as variable in order to allow monkey patching in unit tests.
var NewPayloadTrackerProducer = func(brokerCfg broker.Configuration) (*PayloadTrackerProducer, error) {
	if brokerCfg.PayloadTrackerTopic == "" {
		return nil, nil
	}

	p, err := New(brokerCfg)
	if err != nil {
		log.Error().Err(err).Msg("unable to create a new payload tracker producer")
		return nil, err
	}
	return &PayloadTrackerProducer{
		KafkaProducer: *p,
		Configuration: brokerCfg,
	}, nil
}

// PayloadTrackerMessage represents content of messages
// sent to the Payload Tracker topic in Kafka.
type PayloadTrackerMessage struct {
	Service   string `json:"service"`
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
	Date      string `json:"date"`
	OrgID     string `json:"org_id,omitempty"`
	Account   string `json:"account,omitempty"`
}

// produceMessage produces message to selected topic. That function returns
// partition ID and offset of new message or an error value in case of any
// problem on broker side.
func (producer *PayloadTrackerProducer) produceMessage(trackerMsg PayloadTrackerMessage) (int32, int64, error) {
	jsonBytes, err := json.Marshal(trackerMsg)
	if err != nil {
		return 0, 0, err
	}
	return producer.KafkaProducer.produceMessage(jsonBytes, producer.Configuration.PayloadTrackerTopic)
}

// TrackPayload publishes the status of a payload with the given request ID to
// the payload tracker Kafka topic. Please keep in mind that if the request ID
// is empty, the payload will not be tracked and no error will be raised because
// this can happen in some scenarios and it is not considered an error.
// Instead, only a warning is logged and no error is returned.
func (producer *PayloadTrackerProducer) TrackPayload(
	reqID types.RequestID,
	timestamp time.Time,
	orgID *types.OrgID,
	account *types.Account,
	status string,
) error {
	if len(reqID) == 0 {
		log.Warn().Str("Operation", "TrackPayload").Msg("request ID is missing, null or empty")
		return nil
	}

	statusUpdate := PayloadTrackerMessage{
		Service:   producer.Configuration.ServiceName,
		RequestID: string(reqID),
		Status:    status,
		Date:      timestamp.UTC().Format(time.RFC3339Nano),
	}

	if orgID != nil {
		statusUpdate.OrgID = fmt.Sprintf("%d", *orgID)
	}

	if account != nil {
		statusUpdate.Account = fmt.Sprintf("%d", *account)
	}

	_, _, err := producer.produceMessage(statusUpdate)
	if err != nil {
		log.Error().Err(err).Msgf(
			"unable to produce payload tracker message (request ID: '%s', timestamp: %v, status: '%s')",
			reqID, timestamp, status)

		return err
	}

	return nil
}

// Close allow the Sarama producer to be gracefully closed
func (producer *PayloadTrackerProducer) Close() error {
	if err := producer.KafkaProducer.Close(); err != nil {
		log.Error().Err(err).Msg("unable to close payload tracker producer")
		return err
	}

	return nil
}
