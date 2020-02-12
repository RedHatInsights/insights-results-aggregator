package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var ApiRequests = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "api_endpoints_requests",
	Help: "The total number of requests per endpoint",
}, []string{"url"})

var ConsumedMessages = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "consumed_messages",
	Help: "The total number of messages consumed from kafka",
}, []string{"orgID", "clusterName"})

var ProducedMessages = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "produced_messages",
	Help: "The total number of produced messages",
}, []string{"topic"})

var WrittenReports = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "written_reports",
	Help: "The total number of reports written to the storage",
}, []string{"orgID", "clusterName"})
