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

package tests

import (
	"fmt"
	"os"
	"testing"

	"github.com/RedHatInsights/insights-results-aggregator/broker"
	"github.com/RedHatInsights/insights-results-aggregator/metrics"
	"github.com/RedHatInsights/insights-results-aggregator/producer"
	"github.com/prometheus/client_golang/prometheus"
	prom_models "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func getCounterValue(counter prometheus.Counter) float64 {
	pb := &prom_models.Metric{}
	counter.Write(pb)
	return pb.GetCounter().GetValue()
}

func getCounterVecValue(counterVec *prometheus.CounterVec, labels map[string]string) float64 {
	counter, err := counterVec.GetMetricWith(labels)
	if err != nil {
		panic(fmt.Sprintf("Unable to get counter from counterVec %v", err))
	}
	return getCounterValue(counter)
}

const testTopic = "ccx.ocp.results"
const testOrgID = 1
const testClusterName = "c506a085-201d-4886-9d54-dbc28b97be57"
const testMessage = `{
	"OrgID": 1,
	"ClusterName": "c506a085-201d-4886-9d54-dbc28b97be57",
	"Report": "{}"
}`

func getTestBrokerCfg(t *testing.T) broker.Configuration {
	brokerCfg := broker.Configuration{
		Address: os.Getenv("TEST_KAFKA_ADDRESS"),
		Topic:   testTopic,
		Group:   "",
	}

	assert.NotEmpty(
		t,
		brokerCfg.Address,
		`Please, set up TEST_KAFKA_ADDRESS env variable. For example "localhost:9092"`,
	)

	return brokerCfg
}

// TestProducedMessagesMetric tests that produced messages metric works
func TestProducedMessagesMetric(t *testing.T) {
	// this approach won't work with consumed messages
	// because they are consumed in another process

	brokerCfg := getTestBrokerCfg(t)

	assert.Equal(t, 0.0, getCounterValue(metrics.ProducedMessages))

	producer.ProduceMessage(brokerCfg, testMessage)

	assert.Equal(t, 1.0, getCounterValue(metrics.ProducedMessages))

	for i := 0; i < 3; i++ {
		producer.ProduceMessage(brokerCfg, testMessage)
	}

	assert.Equal(t, 4.0, getCounterValue(metrics.ProducedMessages))
}
