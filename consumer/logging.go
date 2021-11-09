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
	"fmt"
	"time"

	"github.com/Shopify/sarama"
	"github.com/rs/zerolog/log"
)

func logMessageInfo(consumer *KafkaConsumer, originalMessage *sarama.ConsumerMessage, parsedMessage incomingMessage, event string) {
	log.Info().
		Int(offsetKey, int(originalMessage.Offset)).
		Int(partitionKey, int(originalMessage.Partition)).
		Str(topicKey, consumer.Configuration.Topic).
		Int(organizationKey, int(*parsedMessage.Organization)).
		Str(clusterKey, string(*parsedMessage.ClusterName)).
		Int(versionKey, int(parsedMessage.Version)).
		Msg(event)
}

func logClusterInfo(message *incomingMessage) {
	if message == nil {
		log.Info().Msg("nil incoming message, no cluster info will be logged")
		return
	}

	logMessage := fmt.Sprintf("rule hits for %d.%s:", *message.Organization, *message.ClusterName)
	if message.ParsedHits != nil && len(message.ParsedHits) > 0 {
		for _, ph := range message.ParsedHits {
			newLine := fmt.Sprintf("\n\trule: %s; error key: %s", ph.Module, ph.ErrorKey)
			logMessage += newLine
		}
		log.Info().Msg(logMessage)
	} else {
		log.Info().Msg("no rule hits found")
	}
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
		Int(versionKey, int(parsedMessage.Version)).
		Err(err).
		Msg(event)
}

func logMessageWarning(consumer *KafkaConsumer, originalMessage *sarama.ConsumerMessage, parsedMessage incomingMessage, event string) {
	log.Warn().
		Int(offsetKey, int(originalMessage.Offset)).
		Int(partitionKey, int(originalMessage.Partition)).
		Str(topicKey, consumer.Configuration.Topic).
		Int(organizationKey, int(*parsedMessage.Organization)).
		Str(clusterKey, string(*parsedMessage.ClusterName)).
		Int(versionKey, int(parsedMessage.Version)).
		Msg(event)
}

func logDuration(tStart time.Time, tEnd time.Time, offset int64, key string) {
	duration := tEnd.Sub(tStart)
	log.Info().Int64(durationKey, duration.Microseconds()).Int64(offsetKey, offset).Msg(key)
}
