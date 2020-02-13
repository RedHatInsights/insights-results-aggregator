package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// APIRequests is a counter vector for requests to endpoints
var APIRequests = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "api_endpoints_requests",
	Help: "The total number of requests per endpoint",
}, []string{"url"})

// APIResponsesTime collects the information about api response time per endpoint
var APIResponsesTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "api_endpoints_response_time",
	Help:    "API endpoints response time",
	Buckets: prometheus.LinearBuckets(0, 20, 20),
}, []string{"url"})

// ConsumedMessages shows number of messages consumed from kafka by aggregator
var ConsumedMessages = promauto.NewCounter(prometheus.CounterOpts{
	Name: "consumed_messages",
	Help: "The total number of messages consumed from kafka",
})

// ProducedMessages shows number of messages produced by producer package
// probably it will be used only in tests
var ProducedMessages = promauto.NewCounter(prometheus.CounterOpts{
	Name: "produced_messages",
	Help: "The total number of produced messages",
})

// WrittenReports shows number of reports written into the database
var WrittenReports = promauto.NewCounter(prometheus.CounterOpts{
	Name: "written_reports",
	Help: "The total number of reports written to the storage",
})
