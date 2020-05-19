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

// Package producer contains functions that can be used to produce (i.e. send)
// messages to properly configured Kafka broker.
package producer

import (
	"encoding/json"
	"time"

	"github.com/Shopify/sarama"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/metrics"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const (
	// StatusReceived is reported when a new payload is received.
	StatusReceived = "received"
	// StatusMessageProcessed is reported when the message of a payload has been processed.
	StatusMessageProcessed = "processed"
	// StatusSuccess is reported upon a successful handling of a payload.
	StatusSuccess = "success"
	// StatusError is reported when the handling of a payload fails for any reason.
	StatusError = "error"
)

// Producer represents any producer
type Producer interface {
	Close() error
}

// KafkaProducer is an implementation of Producer interface
type KafkaProducer struct {
	Configuration broker.Configuration
	Producer      sarama.SyncProducer
}

// New constructs new implementation of Producer interface
func New(brokerCfg broker.Configuration) (*KafkaProducer, error) {
	producer, err := sarama.NewSyncProducer([]string{brokerCfg.Address}, nil)
	if err != nil {
		log.Error().Err(err).Msg("unable to create a new Kafka producer")
		return nil, err
	}

	return &KafkaProducer{
		Configuration: brokerCfg,
		Producer:      producer,
	}, nil
}

// PayloadTrackerMessage represents content of messages
// sent to the Payload Tracker topic in Kafka.
type PayloadTrackerMessage struct {
	Service   string `json:"service"`
	RequestID string `json:"request_id"`
	Status    string `json:"status"`
	Date      string `json:"date"`
}

// produceMessage produces message to selected topic. That function returns
// partition ID and offset of new message or an error value in case of any
// problem on broker side.
func (producer *KafkaProducer) produceMessage(trackerMsg PayloadTrackerMessage) (int32, int64, error) {
	jsonBytes, err := json.Marshal(trackerMsg)
	if err != nil {
		return 0, 0, err
	}

	producerMsg := &sarama.ProducerMessage{
		Topic: producer.Configuration.PayloadTrackerTopic,
		Value: sarama.ByteEncoder(jsonBytes),
	}

	partition, offset, err := producer.Producer.SendMessage(producerMsg)
	if err != nil {
		log.Error().Err(err).Msg("failed to produce message to Kafka")
	} else {
		log.Info().Msgf("message sent to partition %d at offset %d\n", partition, offset)
		metrics.ProducedMessages.Inc()
	}
	return partition, offset, err
}

// TrackPayload publishes the status of a payload with the given request ID to
// the payload tracker Kafka topic. Please keep in mind that if the request ID
// is empty, the payload will not be tracked and no error will be raised because
// this can happen in some scenarios and it is not considered an error.
// Instead, only a warning is logged and no error is returned.
func (producer *KafkaProducer) TrackPayload(reqID types.RequestID, timestamp time.Time, status string) error {
	if len(reqID) == 0 {
		log.Warn().Msg("request ID is missing, null or empty")
		return nil
	}

	_, _, err := producer.produceMessage(PayloadTrackerMessage{
		Service:   producer.Configuration.ServiceName,
		RequestID: string(reqID),
		Status:    status,
		Date:      timestamp.UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		log.Error().Err(err).Msgf(
			"unable to produce payload tracker message (request ID: '%s', timestamp: %v, status: '%s')",
			reqID, timestamp, status)

		return err
	}

	return nil
}

// Close allow the Sarama producer to be gracefully closed
func (producer *KafkaProducer) Close() error {
	if err := producer.Producer.Close(); err != nil {
		log.Error().Err(err).Msg("unable to close Kafka producer")
		return err
	}

	return nil
}
