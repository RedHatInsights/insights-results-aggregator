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
	"net/http"

	// we just have to import this package in order to expose pprof interface in debug mode
	// disable "G108 (CWE-): Profiling endpoint is automatically exposed on /debug/pprof"
	// #nosec G108
	_ "net/http/pprof"
	"path/filepath"

	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/gorilla/mux"
	"github.com/rs/zerolog/log"

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

func (server *HTTPServer) mainEndpoint(writer http.ResponseWriter, _ *http.Request) {
	err := responses.SendOK(writer, responses.BuildOkResponse())
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

func (server *HTTPServer) listOfOrganizations(writer http.ResponseWriter, _ *http.Request) {
	organizations, err := server.Storage.ListOfOrgs()
	if err != nil {
		log.Error().Err(err).Msg("Unable to get list of organizations")
		handleServerError(writer, err)
		return
	}
	err = responses.SendOK(writer, responses.BuildOkResponseWithData("organizations", organizations))
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

func (server *HTTPServer) listOfClustersForOrganization(writer http.ResponseWriter, request *http.Request) {
	organizationID, err := readOrganizationID(writer, request, server.Config.Auth)

	if err != nil {
		// everything has been handled already
		return
	}

	clusters, err := server.Storage.ListOfClustersForOrg(organizationID)
	if err != nil {
		log.Error().Err(err).Msg("Unable to get list of clusters")
		handleServerError(writer, err)
		return
	}
	err = responses.SendOK(writer, responses.BuildOkResponseWithData("clusters", clusters))
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

func (server *HTTPServer) readReportForCluster(writer http.ResponseWriter, request *http.Request) {
	clusterName, successful := readClusterName(writer, request)
	if !successful {
		// everything has been handled already
		return
	}

	userID, successful := readUserID(writer, request)
	if !successful {
		return
	}

	orgID, successful := readOrgID(writer, request)
	if !successful {
		return
	}

	report, lastChecked, err := server.Storage.ReadReportForCluster(orgID, clusterName)
	if err != nil {
		log.Error().Err(err).Msg("Unable to read report for cluster")
		handleServerError(writer, err)
		return
	}

	var reportRules types.ReportRules
	err = json.Unmarshal([]byte(report), &reportRules)
	if err != nil {
		log.Error().Err(err).Msg("Unable to parse cluster report")
		handleServerError(writer, err)
		return
	}

	hitRules := reportRules.HitRules
	hitRulesCount := len(hitRules)

	hitRules, err = server.getFeedbackAndTogglesOnRules(clusterName, userID, hitRules)

	if err != nil {
		log.Error().Err(err).Msg("An error has occurred when getting feedback or toggles")
		handleServerError(writer, err)
	}

	// -1 as count in response means there are no rules for this cluster
	// as opposed to no rules hit for the cluster
	if hitRulesCount == 0 {
		hitRulesCount = -1
	}

	response := types.ReportResponse{
		Meta: types.ReportResponseMeta{
			Count:         hitRulesCount,
			LastCheckedAt: lastChecked,
		},
		Report: hitRules,
	}

	err = responses.SendOK(writer, responses.BuildOkResponseWithData("report", response))
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// checkUserClusterPermissions retrieves organization ID by checking the owner of cluster ID, checks if it matches the one from request
func (server *HTTPServer) checkUserClusterPermissions(writer http.ResponseWriter, request *http.Request, clusterID types.ClusterName) error {
	if server.Config.Auth {
		orgID, err := server.Storage.GetOrgIDByClusterID(clusterID)
		if err != nil {
			log.Error().Err(err).Msg("Unable to get org id")
			handleServerError(writer, err)
			return err
		}

		err = checkPermissions(writer, request, orgID, server.Config.Auth)
		if err != nil {
			return err
		}
	}
	return nil
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
			handleServerError(writer, err)
			return
		}
	}

	err = responses.SendOK(writer, responses.BuildOkResponse())
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
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
			handleServerError(writer, err)
			return
		}
	}

	err = responses.SendOK(writer, responses.BuildOkResponse())
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// serveAPISpecFile serves an OpenAPI specifications file specified in config file
func (server HTTPServer) serveAPISpecFile(writer http.ResponseWriter, request *http.Request) {
	absPath, err := filepath.Abs(server.Config.APISpecFile)
	if err != nil {
		const message = "Error creating absolute path of OpenAPI spec file"
		log.Error().Err(err).Msg(message)
		handleServerError(writer, err)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	http.ServeFile(writer, request, absPath)
}

// handleOptionsMethod - middleware for handling OPTIONS method
func (server *HTTPServer) handleOptionsMethod(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
			} else {
				nextHandler.ServeHTTP(w, r)
			}
		})
}

// Initialize perform the server initialization
func (server *HTTPServer) Initialize() http.Handler {
	log.Info().Msgf("Initializing HTTP server at '%s'", server.Config.Address)

	router := mux.NewRouter().StrictSlash(true)
	router.Use(server.LogRequest)

	apiPrefix := server.Config.APIPrefix

	metricsURL := apiPrefix + MetricsEndpoint
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
			metricsURL + "?", // to be able to test using Frisby
			openAPIURL + "?", // to be able to test using Frisby
		}
		router.Use(func(next http.Handler) http.Handler { return server.Authentication(next, noAuthURLs) })
	}

	server.addEndpointsToRouter(router)

	return router
}

// Start starts server
func (server *HTTPServer) Start() error {
	address := server.Config.Address
	log.Info().Msgf("Starting HTTP server at '%s'", address)
	router := server.Initialize()
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
