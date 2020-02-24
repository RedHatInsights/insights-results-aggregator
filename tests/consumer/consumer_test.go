package main

import (
	"testing"
	"time"

	"github.com/Shopify/sarama"
	mapset "github.com/deckarep/golang-set"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/consumer"
	"github.com/RedHatInsights/insights-results-aggregator/producer"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

const (
	kafkaAddress  = "localhost:9092"
	testTopicName = "topic"
	testMessage   = `{
		"OrgID": 1,
		"ClusterName": "aaaaaaaa-bbbb-cccc-dddd-000000000000",
		"Report": "{}",
		"LastChecked": "2020-01-23T16:15:59.478901889Z"
	}`
	testCaseTimeLimit = 10 * time.Second
)

var (
	testOrgWhiteList = mapset.NewSetWith(1)
)

// TestConsumerOffset test that kafka consumes messages from the correct offset
func TestConsumerOffset(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		brokerCfg := broker.Configuration{
			Address:      kafkaAddress,
			Topic:        testTopicName,
			Group:        "",
			Enabled:      true,
			OrgWhitelist: testOrgWhiteList,
		}

		// remove topic with all messages
		// b := sarama.NewBroker(kafkaAddress)
		// b.DeleteTopic()
		saramaConf := sarama.NewConfig()
		saramaConf.Version = sarama.V0_10_1_0

		clusterAdmin, err := sarama.NewClusterAdmin([]string{kafkaAddress}, saramaConf)
		if err != nil {
			t.Fatal(err)
		}
		err = clusterAdmin.DeleteTopic(testTopicName)
		if err != nil {
			t.Fatal(err)
		}

		mustProduceMessage := func(message string) {
			_, _, err := producer.ProduceMessage(brokerCfg, message)
			if err != nil {
				t.Fatal(err)
			}
		}

		// produce some messages
		mustProduceMessage("message1")
		mustProduceMessage("message2")

		// then connect
		kafkaConsumer, err := consumer.New(brokerCfg, helpers.MustGetMockStorage(t, true))
		if err != nil {
			t.Fatal(err)
		}

		// and produce one more message
		mustProduceMessage("message3")

		msg := <-kafkaConsumer.PartitionConsumer.Messages()
		if msg == nil {
			t.Fatal("message expected")
		}

		// we expect it to start consuming from the beginning
		assert.Equal(t, "message1", string(msg.Value))

		// disconnect, connect again and expect it to remember offset
		err = kafkaConsumer.Close()
		if err != nil {
			t.Fatal(err)
		}

		kafkaConsumer, err = consumer.New(brokerCfg, helpers.MustGetMockStorage(t, true))
		if err != nil {
			t.Fatal(err)
		}

		msg = <-kafkaConsumer.PartitionConsumer.Messages()
		if msg == nil {
			t.Fatal("message expected")
		}

		assert.Equal(t, "message2", string(msg.Value))
	}, testCaseTimeLimit)
}
