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
		Err(err).
		Msg(event)
}

func logDuration(tStart time.Time, tEnd time.Time, offset int64, key string) {
	duration := tEnd.Sub(tStart)
	log.Info().Int64(durationKey, duration.Microseconds()).Int64(offsetKey, offset).Msg(key)
}
