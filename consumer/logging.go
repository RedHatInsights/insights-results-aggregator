// Copyright 2020, 2021, 2022 Red Hat, Inc
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

	"github.com/IBM/sarama"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func printableRequestID(message *incomingMessage) string {
	var requestID = message.RequestID
	if requestID == "" {
		return "missing"
	}
	return string(requestID)
}

func logMessageDebug(consumer *KafkaConsumer, originalMessage *sarama.ConsumerMessage, parsedMessage *incomingMessage, event string) {
	fillEvent(log.Debug(), consumer, originalMessage, parsedMessage).Msg(event)
}

func logMessageInfo(consumer *KafkaConsumer, originalMessage *sarama.ConsumerMessage, parsedMessage *incomingMessage, event string) {
	fillEvent(log.Info(), consumer, originalMessage, parsedMessage).Msg(event)
}

func logClusterInfo(message *incomingMessage) {
	if message == nil {
		log.Info().Msg("nil incoming message, no cluster info will be logged")
		return
	}

	logMessage := fmt.Sprintf("rule hits for %d.%s (request ID %s):",
		*message.Organization,
		*message.ClusterName,
		printableRequestID(message))
	if message.ParsedHits != nil && len(message.ParsedHits) > 0 {
		for _, ph := range message.ParsedHits {
			newLine := fmt.Sprintf("\n\trule: %s; error key: %s", ph.Module, ph.ErrorKey)
			logMessage += newLine
		}
		log.Debug().Msg(logMessage)
	} else {
		log.Debug().Msg("no rule hits found")
	}
}

func logMessageError(consumer *KafkaConsumer, originalMessage *sarama.ConsumerMessage, parsedMessage *incomingMessage, event string, err error) {
	fillEvent(log.Error(), consumer, originalMessage, parsedMessage).Err(err).Msg(event)
}

func logMessageWarning(consumer *KafkaConsumer, originalMessage *sarama.ConsumerMessage, parsedMessage *incomingMessage, event string) {
	fillEvent(log.Warn(), consumer, originalMessage, parsedMessage).Msg(event)
}

func logDuration(tStart, tEnd time.Time, offset int64, key string) {
	duration := tEnd.Sub(tStart)
	log.Debug().Int64(durationKey, duration.Microseconds()).Int64(offsetKey, offset).Msg(key)
}

func fillEvent(baseEvent *zerolog.Event, consumer *KafkaConsumer, originalMessage *sarama.ConsumerMessage, parsedMessage *incomingMessage) *zerolog.Event {
	baseEvent = baseEvent.Str(topicKey, consumer.Configuration.Topic)

	// Check for nil pointers before raising the log error (CCXDEV-12426)
	if originalMessage == nil {
		log.Debug().Msg("originalMessage is nil")
	} else {
		baseEvent = baseEvent.
			Int(offsetKey, int(originalMessage.Offset)).
			Int(partitionKey, int(originalMessage.Partition))
	}
	if parsedMessage == nil {
		log.Debug().Msg("parsedMessage is nil")
	} else {
		baseEvent = baseEvent.
			Int(versionKey, int(parsedMessage.Version)).
			Str(requestIDKey, printableRequestID(parsedMessage))
		if parsedMessage.Organization == nil {
			log.Debug().Msg("*parsedMessage.Organization is nil")
		} else {
			baseEvent = baseEvent.Int(organizationKey, int(*parsedMessage.Organization))
		}
		if parsedMessage.ClusterName == nil {
			log.Debug().Msg("parsedMessage.ClusterName is nil")
		} else {
			baseEvent = baseEvent.Str(clusterKey, string(*parsedMessage.ClusterName))
		}
	}

	return baseEvent
}
