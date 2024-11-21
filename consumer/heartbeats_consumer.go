package consumer

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	// "github.com/RedHatInsights/insights-results-aggregator/producer"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/rs/zerolog/log"

	// "github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/Shopify/sarama"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
)

// Metadata taken from https://github.com/redhatinsights/insights-ingress-go/internal/validators/types.go
type Metadata struct {
	IPAddresses    []string          `json:"ip_addresses,omitempty"`
	Account        string            `json:"account,omitempty"`
	OrgID          string            `json:"org_id,omitempty"`
	InsightsID     string            `json:"insights_id,omitempty"`
	MachineID      string            `json:"machine_id,omitempty"`
	SubManID       string            `json:"subscription_manager_id,omitempty"`
	MacAddresses   []string          `json:"mac_addresses,omitempty"`
	FQDN           string            `json:"fqdn,omitempty"`
	BiosUUID       string            `json:"bios_uuid,omitempty"`
	DisplayName    string            `json:"display_name,omitempty"`
	AnsibleHost    string            `json:"ansible_host,omitempty"`
	CustomMetadata map[string]string `json:"custom_metadata,omitempty"`
	Reporter       string            `json:"reporter"`
	StaleTimestamp time.Time         `json:"stale_timestamp"`
	QueueKey       string            `json:"queue_key,omitempty"`
}

// Request taken from https://github.com/redhatinsights/insights-ingress-go/internal/validators/types.go
type Request struct {
	Account     string    `json:"account"`
	Category    string    `json:"category"`
	ContentType string    `json:"content_type"`
	Metadata    Metadata  `json:"metadata"`
	RequestID   string    `json:"request_id"`
	Principal   string    `json:"principal"`
	OrgID       string    `json:"org_id"`
	Service     string    `json:"service"`
	Size        int64     `json:"size"`
	URL         string    `json:"url"`
	ID          string    `json:"id,omitempty"`
	B64Identity string    `json:"b64_identity"`
	Timestamp   time.Time `json:"timestamp"`
}

// KkfMessageProcessor Processor for kafka messages used by NewKafkaConsumer
type KkfMessageProcessor interface {
	HandleMessage(msg *sarama.ConsumerMessage) error
}

// KfkConsumer ...
type KfkConsumer struct {
	configuration    broker.Configuration
	client           sarama.ConsumerGroup
	Storage          storage.Storage
	MessageProcessor KkfMessageProcessor
	ready            chan bool
	// cancel                               context.CancelFunc
}

// TODO
// &HearbeatMessageProcessor{Storage: storage}

// NewKfkConsumer constructs a kafka consumer with a message processor
func NewKfkConsumer(brokerCfg broker.Configuration, processor KkfMessageProcessor) (*KfkConsumer, error) {
	saramaConfig, err := broker.SaramaConfigFromBrokerConfig(brokerCfg)
	if err != nil {
		log.Error().Err(err).Msg("unable to create sarama configuration from current broker configuration")
		return nil, err
	}
	log.Info().
		Str("addresses", brokerCfg.Addresses).
		Str("group", brokerCfg.Group).
		Msg("New consumer group")

	consumerGroup, err := sarama.NewConsumerGroup(strings.Split(brokerCfg.Addresses, ","), brokerCfg.Group, saramaConfig)
	if err != nil {
		log.Error().Err(err).Msg("Unable to create consumer group")
		return nil, err
	}
	log.Info().Msg("Consumer group has been created")

	consumer := &KfkConsumer{
		configuration:    brokerCfg,
		client:           consumerGroup,
		MessageProcessor: processor,
		ready:            make(chan bool),
	}

	return consumer, nil
}

// Serve starts listening for messages and processing them. It blocks current thread.
// TODO pass ctx, cancel := context.WithCancel(context.Background())
func (consumer *KfkConsumer) Serve(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)

	go func() {
		for {
			// `Consume` should be called inside an infinite loop, when a
			// server-side rebalance happens, the consumer session will need to be
			// recreated to get the new claims
			if err := consumer.client.Consume(ctx, []string{consumer.configuration.Topic}, consumer); err != nil {
				log.Fatal().Err(err).Msg("unable to recreate kafka session")
			}

			// check if context was cancelled, signaling that the consumer should stop
			if ctx.Err() != nil {
				return
			}

			log.Info().Msg("created new kafka session")

			consumer.ready = make(chan bool)
		}
	}()

	// Wait for the consumer to be set up
	log.Info().Msg("waiting for consumer to become ready")
	<-consumer.ready
	log.Info().Msg("finished waiting for consumer to become ready")

	// Actual processing is done in goroutine created by sarama (see ConsumeClaim below)
	log.Info().Msg("started serving consumer")
	<-ctx.Done()
	log.Info().Msg("context cancelled, exiting")

	cancel()
}

// Setup is run at the beginning of a new session, before ConsumeClaim
func (consumer *KfkConsumer) Setup(sarama.ConsumerGroupSession) error {
	log.Info().Msg("new session has been setup")
	// Mark the consumer as ready
	close(consumer.ready)
	return nil
}

// Cleanup is run at the end of a session, once all ConsumeClaim goroutines have exited
func (consumer *KfkConsumer) Cleanup(sarama.ConsumerGroupSession) error {
	log.Info().Msg("new session has been finished")
	return nil
}

// ConsumeClaim starts a consumer loop of ConsumerGroupClaim's Messages().
func (consumer *KfkConsumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	log.Info().
		Int64(offsetKey, claim.InitialOffset()).
		Msg("starting messages loop")

	for {
		select {
		case message, ok := <-claim.Messages():
			if !ok {
				log.Printf("message channel was closed")
				return nil
			}
			log.Printf("Message claimed: value = %s, timestamp = %v, topic = %s", string(message.Value), message.Timestamp, message.Topic)
			err := consumer.MessageProcessor.HandleMessage(message)
			if err != nil {
				log.Error().Err(err).Msg("Error processing message")
			}
			session.MarkMessage(message, "")
		case <-session.Context().Done():
			return nil
		}
	}
}

// Close method closes all resources used by consumer
func (consumer *KfkConsumer) Close() error {
	if consumer.client != nil {
		if err := consumer.client.Close(); err != nil {
			log.Error().Err(err).Msg("unable to close consumer group")
		}
	}

	return nil
}

// HearbeatMessageProcessor implementation of KkfMessageProccessor for heartbeats
type HearbeatMessageProcessor struct {
	Storage storage.Storage
}

// HandleMessage get a kafka message, deserializes it, parses it and finally send the result to be stored
func (p *HearbeatMessageProcessor) HandleMessage(msg *sarama.ConsumerMessage) error {
	log.Info().
		Int64(offsetKey, msg.Offset).
		Int32(partitionKey, msg.Partition).
		Str(topicKey, msg.Topic).
		Time("message_timestamp", msg.Timestamp).
		Msgf("started processing message")

	// process message
	messageRequest, err := deserializeMessage(msg.Value)
	if err != nil {
		log.Error().Err(err).Msg("Error deserializing message from Kafka")
		return err
	}

	// skip messages not for us
	if messageRequest.ContentType != "application/vnd.redhat.runtimes-java-general.analytics+tgz" {
		log.Info().Msg("Content not for runtimes. Skipping")
		return nil
	}

	instanceID, err := parseHearbeat(messageRequest.URL)

	// Something went wrong while processing the message.
	if err != nil {
		log.Error().Err(err).Msg("Error processing message consumed from Kafka")
		return err
	}
	log.Info().Int64(offsetKey, msg.Offset).Msg("Message consumed")

	err = p.updateHeartbeat(instanceID, messageRequest.Timestamp)
	if err != nil {
		log.Error().Err(err).Msg("Error updating hearbeat Kafka")
		return err
	}
	log.Info().Msg("Heartbeat updated")

	return nil
}

// deserializeMessage deserialize a kafka meesage
func deserializeMessage(messageValue []byte) (Request, error) {
	var deserialized Request

	received, err := DecompressMessage(messageValue)
	if err != nil {
		return deserialized, err
	}

	err = json.Unmarshal(received, &deserialized)
	if err != nil {
		return deserialized, err
	}
	return deserialized, nil
}

// HeartbeatDetails details of the hearbeat
type HeartbeatDetails struct {
	ObjectUID string `json:"object_uid"`
}

// Heartbeat data
type Heartbeat struct {
	Details HeartbeatDetails `json:"details"`
}

func parseHearbeat(url string) (string, error) {
	resp, err := http.Get(url) /* #nosec G107 */
	if err != nil {
		log.Error().Err(err).Msg("Error downloading remote file")
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Error().Err(err).Msg("Error reading remote file")
		return "", err
	}

	var heartbeat Heartbeat
	err = json.Unmarshal(bodyBytes, &heartbeat)
	if err != nil {
		log.Error().Err(err).Msg("Error parsing JSON")
		return "", err
	}

	if heartbeat.Details.ObjectUID == "" {
		return "", errors.New("Empty value for Object UID")
	}

	log.Info().Msg("Read heartbeat")
	log.Debug().Msgf("Processed heartbeat for %s", heartbeat.Details.ObjectUID)
	return heartbeat.Details.ObjectUID, nil
}

func (p *HearbeatMessageProcessor) updateHeartbeat(
	objectUID string, timestamp time.Time,
) error {
	if dvoStorage, ok := p.Storage.(storage.DVORecommendationsStorage); ok {
		err := dvoStorage.UpdateHeartbeat(
			objectUID,
			timestamp,
		)
		if err != nil {
			log.Error().Err(err).Msg("Error updating heartbeat in database")
			return err
		}
		log.Debug().Msgf("Update heartbeat %s", objectUID)
		return nil
	}
	err := errors.New("heartbeats could not be updated")
	log.Error().Err(err).Msg(unexpectedStorageType)
	return err
}
