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
// consumed_messages - total number of messages consumed from selected broker
//
// consuming_errors - total number of errors during consuming messages from selected broker
//
// successful_messages_processing_time - time to process successfully message
//
// failed_messages_processing_time - time to process message fail
//
// last_checked_timestamp_lag_minutes - shows how slow we get messages from clusters
//
// produced_messages - total number of produced messages sent to Payload Tracker's Kafka topic
//
// written_reports - total number of reports written into the storage (cache)
//
// feedback_on_rules - total number of left feedback
//
// sql_queries_counter - total number of SQL queries
//
// sql_queries_durations - SQL queries durations
//
// sql_recommendations_updates - number of insert and deletes in recommendations table
package metrics

import (
	"github.com/RedHatInsights/insights-operator-utils/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

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
	Help: "The total number of produced messages sent to Payload Tracker's Kafka topic",
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

// RatingOnRules shows how many times users sends a rating on rules
var RatingOnRules = promauto.NewCounter(prometheus.CounterOpts{
	Name: "rating_on_rules",
	Help: "The total number of left rating",
})

// SQLQueriesCounter shows number of sql queries
var SQLQueriesCounter = promauto.NewCounter(prometheus.CounterOpts{
	Name: "sql_queries_counter",
	Help: "Number of SQL queries",
})

// SQLQueriesDurations shows durations for sql queries (without parameters).
var SQLQueriesDurations = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name: "sql_queries_durations",
	Help: "SQL queries durations",
}, []string{"query"})

/*
// SQLRecommendationsDeletes shows deleted entries in recommendations table.
var SQLRecommendationsDeletes = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name: "sql_recommendations_deletes",
	Help: "Number of rows removed from the SQL recommendation table when new report is processed",
}, []string{"cluster"})

// SQLRecommendationsInserts shows inserted entries in recommendations table.
var SQLRecommendationsInserts = promauto.NewHistogramVec(prometheus.HistogramOpts{
	Name: "sql_recommendations_inserts",
	Help: "Number of rows added to the SQL recommendation table when new report is processed",
}, []string{"cluster"})
*/

// AddMetricsWithNamespace register the desired metrics using a given namespace
func AddMetricsWithNamespace(namespace string) {
	metrics.AddAPIMetricsWithNamespace(namespace)

	prometheus.Unregister(ConsumedMessages)
	prometheus.Unregister(ConsumingErrors)
	prometheus.Unregister(SuccessfulMessagesProcessingTime)
	prometheus.Unregister(FailedMessagesProcessingTime)
	prometheus.Unregister(LastCheckedTimestampLagMinutes)
	prometheus.Unregister(ProducedMessages)
	prometheus.Unregister(WrittenReports)
	prometheus.Unregister(FeedbackOnRules)
	prometheus.Unregister(SQLQueriesCounter)
	prometheus.Unregister(SQLQueriesDurations)
	//prometheus.Unregister(SQLRecommendationsDeletes)
	//prometheus.Unregister(SQLRecommendationsInserts)

	ConsumedMessages = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "consumed_messages",
		Help:      "The total number of messages consumed from Kafka",
	})
	ConsumingErrors = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "consuming_errors",
		Help:      "The total number of errors during consuming messages from Kafka",
	})
	SuccessfulMessagesProcessingTime = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "successful_messages_processing_time",
		Help:      "Time to process successfully message",
	})
	FailedMessagesProcessingTime = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "failed_messages_processing_time",
		Help:      "Time to process message fail",
	})
	LastCheckedTimestampLagMinutes = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "last_checked_timestamp_lag_minutes",
		Help:      "Shows how slow we get messages from clusters",
	})
	ProducedMessages = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "produced_messages",
		Help:      "The total number of produced messages sent to Payload Tracker's Kafka topic",
	})
	WrittenReports = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "written_reports",
		Help:      "The total number of reports written to the storage",
	})
	FeedbackOnRules = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "feedback_on_rules",
		Help:      "The total number of left feedback",
	})
	SQLQueriesCounter = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Name:      "sql_queries_counter",
		Help:      "Number of SQL queries",
	})
	SQLQueriesDurations = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Name:      "sql_queries_durations",
		Help:      "SQL queries durations",
	}, []string{"query"})
	/*
		SQLRecommendationsDeletes = promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "sql_recommendations_deletes",
			Help:      "Number of rows removed from the SQL recommendation table when new report is processed",
		}, []string{"cluster"})
		SQLRecommendationsInserts = promauto.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "sql_recommendations_inserts",
			Help:      "Number of rows added to the SQL recommendation table when new report is processed",
		}, []string{"cluster"})
	*/
}
