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

package producer

import (
	"log"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/metrics"
	"github.com/Shopify/sarama"
	"github.com/prometheus/client_golang/prometheus"
)

// ProduceMessage produces message to selected topic
func ProduceMessage(brokerCfg broker.Configuration, message string) (partition int32, offset int64, errout error) {
	producer, err := sarama.NewSyncProducer([]string{brokerCfg.Address}, nil)
	if err != nil {
		log.Print(err)
		return -1, -1, err
	}
	defer func() {
		if err := producer.Close(); err != nil {
			log.Print(err)
			partition = -1
			offset = -1
			errout = err
		}
	}()

	msg := &sarama.ProducerMessage{Topic: brokerCfg.Topic, Value: sarama.StringEncoder(message)}
	partition, offset, err = producer.SendMessage(msg)
	if err != nil {
		log.Printf("FAILED to send message: %s\n", err)
	} else {
		log.Printf("message sent to partition %d at offset %d\n", partition, offset)
		metrics.ProducedMessages.With(prometheus.Labels{"topic": brokerCfg.Topic}).Inc()
	}
	return partition, offset, err
}
