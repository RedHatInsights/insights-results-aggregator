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
