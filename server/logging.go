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

package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/metrics"
)

type loggingResponseWriter struct {
	http.ResponseWriter
}

func (writer loggingResponseWriter) WriteHeader(statusCode int) {
	writer.ResponseWriter.WriteHeader(statusCode)
	metrics.APIResponseStatusCodes.With(
		prometheus.Labels{"status_code": fmt.Sprint(statusCode)},
	).Inc()
}

func logRequestHandler(writer http.ResponseWriter, request *http.Request, nextHandler http.Handler) {
	log.Print("Request URI: " + request.RequestURI)
	log.Print("Request method: " + request.Method)
	metrics.APIRequests.With(prometheus.Labels{"url": request.RequestURI}).Inc()

	startTime := time.Now()
	nextHandler.ServeHTTP(&loggingResponseWriter{ResponseWriter: writer}, request)
	duration := time.Since(startTime)

	metrics.APIResponsesTime.With(
		prometheus.Labels{"url": request.RequestURI},
	).Observe(float64(duration.Microseconds()))
}

// LogRequest - middleware for logging requests
func (server *HTTPServer) LogRequest(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			logRequestHandler(writer, request, nextHandler)
		})
}
