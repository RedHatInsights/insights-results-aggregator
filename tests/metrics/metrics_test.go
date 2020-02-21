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

// TODO: make it unit tests with mocked kafka

package tests

import (
	"fmt"
	"testing"

	"github.com/RedHatInsights/insights-results-aggregator/metrics"
	"github.com/RedHatInsights/insights-results-aggregator/producer"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/prometheus/client_golang/prometheus"
	prom_models "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func getCounterValue(t *testing.T, counter prometheus.Counter) float64 {
	pb := &prom_models.Metric{}

	err := counter.Write(pb)
	if err != nil {
		t.Fatal(err)
	}

	return pb.GetCounter().GetValue()
}

func getCounterVecValue(t *testing.T, counterVec *prometheus.CounterVec, labels map[string]string) float64 {
	counter, err := counterVec.GetMetricWith(labels)
	if err != nil {
		panic(fmt.Sprintf("Unable to get counter from counterVec %v", err))
	}

	return getCounterValue(t, counter)
}

const (
	testTopic   = "ccx.ocp.results"
	testMessage = `{
	"OrgID": 1,
	"ClusterName": "c506a085-201d-4886-9d54-dbc28b97be57",
	"Report": "{}"
}`
)

// TestProducedMessagesMetric tests that produced messages metric works
func TestProducedMessagesMetric(t *testing.T) {
	// this approach won't work with consumed messages
	// because they are consumed in another process
	brokerCfg := helpers.GetTestBrokerCfg(t, testTopic)

	assert.Equal(t, 0.0, getCounterValue(t, metrics.ProducedMessages))

	_, _, err := producer.ProduceMessage(brokerCfg, testMessage)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 1.0, getCounterValue(t, metrics.ProducedMessages))

	for i := 0; i < 3; i++ {
		_, _, err := producer.ProduceMessage(brokerCfg, testMessage)
		if err != nil {
			t.Fatal(err)
		}
	}

	assert.Equal(t, 4.0, getCounterValue(t, metrics.ProducedMessages))
}
