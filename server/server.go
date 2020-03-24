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

// Package server contains implementation of REST API server (HTTPServer) for the
// Insights results aggregator service. In current version, the following
// REST API endpoints are available:
//
// API_PREFIX/organizations - list of all organizations (HTTP GET)
//
// API_PREFIX/organizations/{organization}/clusters - list of all clusters for given organization (HTTP GET)
//
// API_PREFIX/report/{organization}/{cluster} - insights OCP results for given cluster name (HTTP GET)
//
// API_PREFIX/rule/{cluster}/{rule_id}/like - like a rule for cluster with current user (from auth token)
//
// API_PREFIX/rule/{cluster}/{rule_id}/dislike - dislike a rule for cluster with current user (from auth token)
//
// API_PREFIX/rule/{cluster}/{rule_id}/reset_vote- reset vote for a rule for cluster with current user (from auth token)
//
// Please note that API_PREFIX is part of server configuration (see Configuration). Also please note that
// JSON format is used to transfer data between server and clients.
//
// Configuration:
//
// It is possible to configure the HTTP server. Currently, two configuration options are available and can
// be changed by using Configuration structure:
//
// Address - usually just in a form ":8080", ie. just the port needs to be configured in most cases
// APIPrefix - usually "/api/v1/" used for all REST API calls
package server

import (
	"context"
	"encoding/json"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/RedHatInsights/insights-results-aggregator/metrics"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// HTTPServer in an implementation of Server interface
type HTTPServer struct {
	Config  Configuration
	Storage storage.Storage
	Serv    *http.Server
}

// New constructs new implementation of Server interface
func New(config Configuration, storage storage.Storage) *HTTPServer {
	return &HTTPServer{
		Config:  config,
		Storage: storage,
	}
}

func logRequestHandler(writer http.ResponseWriter, request *http.Request, nextHandler http.Handler) {
	log.Print("Request URI: " + request.RequestURI)
	log.Print("Request method: " + request.Method)
	metrics.APIRequests.With(prometheus.Labels{"url": request.RequestURI}).Inc()
	startTime := time.Now()
	nextHandler.ServeHTTP(writer, request)
	duration := time.Since(startTime)
	metrics.APIResponsesTime.With(prometheus.Labels{"url": request.RequestURI}).Observe(float64(duration.Microseconds()))
}

// LogRequest - middleware for logging requests
func (server *HTTPServer) LogRequest(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			logRequestHandler(writer, request, nextHandler)
		})
}

func (server *HTTPServer) mainEndpoint(writer http.ResponseWriter, _ *http.Request) {
	responses.SendResponse(writer, responses.BuildOkResponse())
}

func (server *HTTPServer) listOfOrganizations(writer http.ResponseWriter, _ *http.Request) {
	organizations, err := server.Storage.ListOfOrgs()
	if err != nil {
		log.Error().Err(err).Msg("Unable to get list of organizations")
		responses.SendInternalServerError(writer, err.Error())
	} else {
		responses.SendResponse(writer, responses.BuildOkResponseWithData("organizations", organizations))
	}
}

func (server *HTTPServer) listOfClustersForOrganization(writer http.ResponseWriter, request *http.Request) {
	organizationID, err := readOrganizationID(writer, request)

	if err != nil {
		// everything has been handled already
		return
	}

	clusters, err := server.Storage.ListOfClustersForOrg(organizationID)
	if err != nil {
		log.Error().Err(err).Msg("Unable to get list of clusters")
		responses.SendInternalServerError(writer, err.Error())
	} else {
		responses.SendResponse(writer, responses.BuildOkResponseWithData("clusters", clusters))
	}
}

func getTotalRuleCount(reportRules types.ReportRules) int {
	totalCount := len(reportRules.HitRules) +
		len(reportRules.SkippedRules) +
		len(reportRules.PassedRules)
	return totalCount
}

// getContentForRules returns the hit rules from the report, as well as total count of all rules (skipped, ..)
func (server *HTTPServer) getContentForRules(
	writer http.ResponseWriter,
	report types.ClusterReport,
) ([]types.RuleContentResponse, int, error) {
	var reportRules types.ReportRules

	err := json.Unmarshal([]byte(report), &reportRules)
	if err != nil {
		log.Error().Err(err).Msg("Unable to parse cluster report")
		responses.SendInternalServerError(writer, err.Error())
		return nil, 0, err
	}

	totalRules := getTotalRuleCount(reportRules)

	hitRules, err := server.Storage.GetContentForRules(reportRules)
	if err != nil {
		log.Error().Err(err).Msg("Unable to retrieve rules content from database")
		responses.SendInternalServerError(writer, err.Error())
		return nil, 0, err
	}

	return hitRules, totalRules, nil
}

func (server *HTTPServer) readReportForCluster(writer http.ResponseWriter, request *http.Request) {
	organizationID, err := readOrganizationID(writer, request)
	if err != nil {
		// everything has been handled already
		return
	}

	clusterName, err := readClusterName(writer, request)
	if err != nil {
		// everything has been handled already
		return
	}

	report, lastChecked, err := server.Storage.ReadReportForCluster(organizationID, clusterName)
	if _, ok := err.(*storage.ItemNotFoundError); ok {
		responses.Send(http.StatusNotFound, writer, err.Error())
		return
	} else if err != nil {
		log.Error().Err(err).Msg("Unable to read report for cluster")
		responses.SendInternalServerError(writer, err.Error())
		return
	}

	rulesContent, rulesCount, err := server.getContentForRules(writer, report)
	if err != nil {
		// everything has been handled already
		return
	}
	hitRulesCount := len(rulesContent)
	// -1 as count in response means there are no rules for this cluster
	// as opposed to no rules hit for the cluster
	if rulesCount == 0 {
		rulesCount = -1
	} else {
		rulesCount = hitRulesCount
	}

	response := types.ReportResponse{
		Meta: types.ReportResponseMeta{
			Count:         rulesCount,
			LastCheckedAt: lastChecked,
		},
		Rules: rulesContent,
	}

	responses.SendResponse(writer, responses.BuildOkResponseWithData("report", response))
}

// likeRule likes the rule for current user
func (server *HTTPServer) likeRule(writer http.ResponseWriter, request *http.Request) {
	server.voteOnRule(writer, request, storage.UserVoteLike)
}

// dislikeRule dislikes the rule for current user
func (server *HTTPServer) dislikeRule(writer http.ResponseWriter, request *http.Request) {
	server.voteOnRule(writer, request, storage.UserVoteDislike)
}

// resetVoteOnRule resets vote for the rule for current user
func (server *HTTPServer) resetVoteOnRule(writer http.ResponseWriter, request *http.Request) {
	server.voteOnRule(writer, request, storage.UserVoteNone)
}

func (server *HTTPServer) voteOnRule(writer http.ResponseWriter, request *http.Request, userVote storage.UserVote) {
	clusterID, err := readClusterName(writer, request)
	if err != nil {
		const message = "Unable to read cluster ID from request"
		log.Error().Err(err).Msg(message)
		responses.Send(http.StatusInternalServerError, writer, message)
		return
	}
	ruleID, err := getRouterParam(request, "rule_id")
	if err != nil {
		const message = "Unable to read rule ID from request"
		log.Error().Err(err).Msg(message)
		responses.Send(http.StatusInternalServerError, writer, message)
		return
	}

	userID, err := server.GetCurrentUserID(request)
	if err != nil {
		const message = "Unable to get user id"
		log.Error().Err(err).Msg(message)
		responses.Send(http.StatusInternalServerError, writer, message)
		return
	}

	err = server.Storage.VoteOnRule(clusterID, types.RuleID(ruleID), userID, userVote)
	if err != nil {
		responses.Send(http.StatusInternalServerError, writer, err.Error())
	} else {
		responses.Send(http.StatusOK, writer, responses.BuildOkResponse())
	}
}

func (server *HTTPServer) deleteOrganizations(writer http.ResponseWriter, request *http.Request) {
	orgIds, err := readOrganizationIDs(writer, request)
	if err != nil {
		// everything has been handled already
		return
	}

	for _, org := range orgIds {
		if err := server.Storage.DeleteReportsForOrg(org); err != nil {
			log.Error().Err(err).Msg("Unable to delete reports")
			responses.SendInternalServerError(writer, err.Error())
			return
		}
	}

	responses.SendResponse(writer, responses.BuildOkResponse())
}

func (server *HTTPServer) deleteClusters(writer http.ResponseWriter, request *http.Request) {
	clusterNames, err := readClusterNames(writer, request)
	if err != nil {
		// everything has been handled already
		return
	}

	for _, cluster := range clusterNames {
		if err := server.Storage.DeleteReportsForCluster(cluster); err != nil {
			log.Error().Err(err).Msg("Unable to delete reports")
			responses.SendInternalServerError(writer, err.Error())
			return
		}
	}

	responses.SendResponse(writer, responses.BuildOkResponse())
}

// serveAPISpecFile serves an OpenAPI specifications file specified in config file
func (server HTTPServer) serveAPISpecFile(writer http.ResponseWriter, request *http.Request) {
	absPath, err := filepath.Abs(server.Config.APISpecFile)
	if err != nil {
		const message = "Error creating absolute path of OpenAPI spec file"
		log.Error().Err(err).Msg(message)
		responses.Send(
			http.StatusInternalServerError,
			writer,
			message,
		)
		return
	}

	http.ServeFile(writer, request, absPath)
}

// Initialize perform the server initialization
func (server *HTTPServer) Initialize(address string) http.Handler {
	log.Print("Initializing HTTP server at", address)

	router := mux.NewRouter().StrictSlash(true)
	router.Use(server.LogRequest)

	apiPrefix := server.Config.APIPrefix

	metricsURL := apiPrefix + "metrics"
	openAPIURL := apiPrefix + filepath.Base(server.Config.APISpecFile)

	// enable authentication, but only if it is setup in configuration
	if server.Config.Auth {
		// we have to enable authentication for all endpoints, including endpoints
		// for Prometheus metrics and OpenAPI specification, because there is not
		// single prefix of other REST API calls. The special endpoints needs to
		// be handled in middleware which is not optimal
		noAuthURLs := []string{
			metricsURL,
			openAPIURL,
		}
		router.Use(func(next http.Handler) http.Handler { return server.Authentication(next, noAuthURLs) })
	}

	// it is possible to use special REST API endpoints in debug mode
	if server.Config.Debug {
		router.HandleFunc(apiPrefix+"organizations/{organizations}", server.deleteOrganizations).Methods(http.MethodDelete)
		router.HandleFunc(apiPrefix+"clusters/{clusters}", server.deleteClusters).Methods(http.MethodDelete)
	}

	// common REST API endpoints
	router.HandleFunc(apiPrefix, server.mainEndpoint).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+"organizations", server.listOfOrganizations).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+"report/{organization}/{cluster}", server.readReportForCluster).Methods(http.MethodGet)
	router.HandleFunc(apiPrefix+"clusters/{cluster}/rules/{rule_id}/like", server.likeRule).Methods(http.MethodPut)
	router.HandleFunc(apiPrefix+"clusters/{cluster}/rules/{rule_id}/dislike", server.dislikeRule).Methods(http.MethodPut)
	router.HandleFunc(apiPrefix+"clusters/{cluster}/rules/{rule_id}/reset_vote", server.resetVoteOnRule).Methods(http.MethodPut)
	router.HandleFunc(apiPrefix+"organizations/{organization}/clusters", server.listOfClustersForOrganization).Methods(http.MethodGet)

	// Prometheus metrics
	router.Handle(metricsURL, promhttp.Handler()).Methods(http.MethodGet)

	// OpenAPI specs
	router.HandleFunc(openAPIURL, server.serveAPISpecFile).Methods(http.MethodGet)

	return router
}

// Start starts server
func (server *HTTPServer) Start() error {
	address := server.Config.Address
	log.Print("Starting HTTP server at", address)
	router := server.Initialize(address)
	server.Serv = &http.Server{Addr: address, Handler: router}

	err := server.Serv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Error().Err(err).Msg("Unable to start HTTP server")
		return err
	}

	return nil
}

// Stop stops server's execution
func (server *HTTPServer) Stop(ctx context.Context) error {
	return server.Serv.Shutdown(ctx)
}
