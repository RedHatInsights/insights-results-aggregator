package consumer_test

import (
	_ "embed"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/consumer"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/Shopify/sarama"
	"github.com/stretchr/testify/assert"
)

type NoopProcessor struct{}

func (p *NoopProcessor) HandleMessage(msg *sarama.ConsumerMessage) error {
	return nil
}

//go:embed heartbeat_ingress.json
var heartbeatIngressMessage string

// var saramaMessage = sarama.ConsumerMessage{Value: []byte(heartbeatIngressMessage)}

//go:embed heartbeat.json
var heartbeatMessage string

func TestHeartbeatsConsumer_New(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t testing.TB) {
		sarama.Logger = log.New(os.Stdout, saramaLogPrefix, log.LstdFlags)

		mockBroker := sarama.NewMockBroker(t, 0)
		defer mockBroker.Close()

		mockBroker.SetHandlerByMap(ira_helpers.GetHandlersMapForMockConsumer(t, mockBroker, testTopicName))

		mockConsumer, err := consumer.NewKfkConsumer(broker.Configuration{
			Addresses: mockBroker.Addr(),
			Topic:     testTopicName,
			Enabled:   true,
		}, &NoopProcessor{})
		helpers.FailOnError(t, err)

		err = mockConsumer.Close()
		helpers.FailOnError(t, err)
	}, testCaseTimeLimit)
}

func TestHeartbeatsConsumer_NewNoKafka(t *testing.T) {
	_, err := consumer.NewKfkConsumer(broker.Configuration{
		Addresses: "localhost:1234",
		Topic:     testTopicName,
		Enabled:   true,
	}, &NoopProcessor{})
	assert.Error(t, err)
	assert.Contains(
		t, err.Error(), "kafka: client has run out of available brokers to talk to",
	)
}

func TestHeartbeatsConsumer_SetupCleanup(t *testing.T) {
	mockBroker := sarama.NewMockBroker(t, 0)
	defer mockBroker.Close()

	mockBroker.SetHandlerByMap(ira_helpers.GetHandlersMapForMockConsumer(t, mockBroker, testTopicName))

	mockConsumer, err := consumer.NewKfkConsumer(broker.Configuration{
		Addresses: mockBroker.Addr(),
		Topic:     testTopicName,
		Enabled:   true,
	}, &NoopProcessor{})
	helpers.FailOnError(t, err)

	defer func() {
		helpers.FailOnError(t, mockConsumer.Close())
	}()

	// The functions don't really use their arguments at all,
	// so it's possible to just pass nil into them.
	helpers.FailOnError(t, mockConsumer.Setup(nil))
	helpers.FailOnError(t, mockConsumer.Cleanup(nil))
}

func TestHeartbeatHandling(t *testing.T) {
	processor := consumer.HearbeatMessageProcessor{Storage: &storage.NoopDVOStorage{}}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, heartbeatMessage)
	}))
	defer ts.Close()

	msg := strings.Replace(heartbeatIngressMessage, "myserverurl", ts.URL, 1)
	saramaMessage := sarama.ConsumerMessage{Value: []byte(msg)}

	err := processor.HandleMessage(&saramaMessage)
	helpers.FailOnError(t, err)
}

func TestHeartbeatHandling_ProcessingError(t *testing.T) {
	processor := consumer.HearbeatMessageProcessor{Storage: &storage.NoopDVOStorage{}}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, ``)
	}))
	defer ts.Close()

	msg := strings.Replace(heartbeatIngressMessage, "myserverurl", ts.URL, 1)
	saramaMessage := sarama.ConsumerMessage{Value: []byte(msg)}

	err := processor.HandleMessage(&saramaMessage)
	assert.Error(t, err)
}

func TestHeartbeatHandling_EmptyData(t *testing.T) {
	processor := consumer.HearbeatMessageProcessor{Storage: &storage.NoopDVOStorage{}}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprintln(w, `{}`)
	}))
	defer ts.Close()

	msg := strings.Replace(heartbeatIngressMessage, "myserverurl", ts.URL, 1)
	saramaMessage := sarama.ConsumerMessage{Value: []byte(msg)}

	err := processor.HandleMessage(&saramaMessage)
	assert.Error(t, err)
}
