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

// Package metrics contains all metrics that needs to be exposed to Prometheus
// and indirectly to Grafana. Currently, the following metrics are exposed:
//
// api_endpoints_requests - number of requests made for each REST API endpoint
//
// api_endpoints_response_time - response times for all REST API endpoints
//
// consumed_messages - total number of messages consumed from selected broker
//
// produced_messages - total number of produced messages
//
// written_reports - total number of reports written into the storage (cache)
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// APIRequests is a counter vector for requests to endpoints
var APIRequests = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "api_endpoints_requests",
	Help: "The total number of requests per endpoint",
}, []string{"endpoint"})

// APIResponsesTime collects the information about api response time per endpoint
var APIResponsesTime = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name:    "api_endpoints_response_time",
	Help:    "API endpoints response time",
	Buckets: prometheus.LinearBuckets(0, 20, 20),
}, []string{"endpoint"})

// APIResponseStatusCodes collects the information about api response status codes
var APIResponseStatusCodes = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "api_endpoints_status_codes",
	Help: "API endpoints status codes",
}, []string{"status_code"})

// ConsumedMessages shows number of messages consumed from Kafka by aggregator
var ConsumedMessages = promauto.NewCounter(prometheus.CounterOpts{
	Name: "consumed_messages",
	Help: "The total number of messages consumed from Kafka",
})

// ConsumingErrors shows the total number of errors during consuming messages from Kafka
var ConsumingErrors = promauto.NewCounter(prometheus.CounterOpts{
	Name: "consuming_errors",
	Help: "The total number of errors during consuming messages from Kafka",
})

// SuccessfulMessagesProcessingTime collects the time to process message successfully
var SuccessfulMessagesProcessingTime = promauto.NewHistogram(prometheus.HistogramOpts{
	Name: "successful_messages_processing_time",
	Help: "Time to process successfully message",
})

// FailedMessagesProcessingTime collects the time of processing message when it failed
var FailedMessagesProcessingTime = promauto.NewHistogram(prometheus.HistogramOpts{
	Name: "failed_messages_processing_time",
	Help: "Time to process message fail",
})

// LastCheckedTimestampLagMinutes shows how slow we get messages from clusters
var LastCheckedTimestampLagMinutes = promauto.NewHistogram(prometheus.HistogramOpts{
	Name: "last_checked_timestamp_lag_minutes",
	Help: "Shows how slow we get messages from clusters",
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

// FeedbackOnRules shows how many times users left feedback on rules
var FeedbackOnRules = promauto.NewCounter(prometheus.CounterOpts{
	Name: "feedback_on_rules",
	Help: "The total number of left feedback",
})
