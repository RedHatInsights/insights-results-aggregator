package metrics_test

import (
	"fmt"
	"testing"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/prometheus/client_golang/prometheus"
	prom_models "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/metrics"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
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

const (
	testTopicName     = "ccx.ocp.results"
	testOrgID         = 1
	testClusterName   = "c506a085-201d-4886-9d54-dbc28b97be57"
	testCaseTimeLimit = 10 * time.Second
)

var (
	testMessage = `{
		"OrgID": ` + fmt.Sprint(testOrgID) + `,
		"ClusterName": "` + testClusterName + `",
		"Report": "{}",
		"LastChecked": "2020-01-23T16:15:59.478901889Z"
	}`
	testOrgWhiteList = mapset.NewSetWith(1)
)

// TestProducedMessagesMetric tests that produced messages metric works
func TestConsumedMessagesMetric(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		mockConsumer := helpers.MustGetMockKafkaConsumerWithExpectedMessages(
			t, testTopicName, testOrgWhiteList, []string{testMessage, testMessage},
		)

		assert.Equal(t, 0.0, getCounterValue(metrics.ConsumedMessages))

		go mockConsumer.Serve()

		helpers.WaitForMockConsumerToHaveNConsumedMessages(mockConsumer, 2)

		assert.Equal(t, 2.0, getCounterValue(metrics.ConsumedMessages))
	}, testCaseTimeLimit)
}

// TODO: metrics.APIRequests
// TODO: metrics.APIResponsesTime
// TODO: metrics.ProducedMessages
// TODO: metrics.WrittenReports
