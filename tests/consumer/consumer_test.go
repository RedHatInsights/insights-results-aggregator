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
	kafkaAddress      = "localhost:9092"
	testTopicName     = "topic"
	testCaseTimeLimit = 10 * time.Second
)

var (
	testOrgWhiteList = mapset.NewSetWith(1, 2, 3)
	testBrokerCfg    = broker.Configuration{
		Address:      kafkaAddress,
		Topic:        testTopicName,
		Group:        "",
		Enabled:      true,
		OrgWhitelist: testOrgWhiteList,
	}
)

func mustNewKafkaConsumer(t *testing.T) *consumer.KafkaConsumer {
	kafkaConsumer, err := consumer.New(testBrokerCfg, helpers.MustGetMockStorage(t, true))
	if err != nil {
		t.Fatal(err)
	}
	return kafkaConsumer
}

func mustProcessNextMessage(t *testing.T, kafkaConsumer *consumer.KafkaConsumer) string {
	msg := <-kafkaConsumer.PartitionConsumer.Messages()
	if msg == nil {
		t.Fatal("message expected")
	}

	err := kafkaConsumer.ProcessMessage(msg)
	if err != nil {
		t.Fatal(err)
	}

	return string(msg.Value)
}

// TestConsumerOffset test that kafka consumes messages from the correct offset
func TestConsumerOffset(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		const (
			message1 = `{
				"OrgID": 1,
				"ClusterName": "aaaaaaaa-bbbb-cccc-dddd-000000000001",
				"Report": "{}",
				"LastChecked": "2020-01-23T16:15:59.478901889Z"
			}`
			message2 = `{
				"OrgID": 2,
				"ClusterName": "aaaaaaaa-bbbb-cccc-dddd-000000000002",
				"Report": "{}",
				"LastChecked": "2020-01-23T16:15:59.478901889Z"
			}`
			message3 = `{
				"OrgID": 3,
				"ClusterName": "aaaaaaaa-bbbb-cccc-dddd-000000000003",
				"Report": "{}",
				"LastChecked": "2020-01-23T16:15:59.478901889Z"
			}`
		)

		saramaConf := sarama.NewConfig()
		// deleting topics is supported from this version
		saramaConf.Version = sarama.V0_10_1_0

		// remove topic with all messages
		clusterAdmin, err := sarama.NewClusterAdmin([]string{kafkaAddress}, saramaConf)
		if err != nil {
			t.Fatal(err)
		}
		err = clusterAdmin.DeleteTopic(testTopicName)
		if err != nil {
			t.Fatal(err)
		}

		mustProduceMessage := func(message string) {
			_, _, err := producer.ProduceMessage(testBrokerCfg, message)
			if err != nil {
				t.Fatal(err)
			}
		}

		// produce some messages
		mustProduceMessage(message1)
		mustProduceMessage(message2)

		// then connect
		kafkaConsumer := mustNewKafkaConsumer(t)

		// and produce one more message
		mustProduceMessage(message3)

		message := mustProcessNextMessage(t, kafkaConsumer)

		// we expect it to start consuming from the beginning
		assert.Equal(t, message1, message)

		// disconnect, connect again and expect it to remember offset
		err = kafkaConsumer.Close()
		if err != nil {
			t.Fatal(err)
		}

		kafkaConsumer = mustNewKafkaConsumer(t)

		message = mustProcessNextMessage(t, kafkaConsumer)

		assert.Equal(t, message2, message)
	}, testCaseTimeLimit)
}
