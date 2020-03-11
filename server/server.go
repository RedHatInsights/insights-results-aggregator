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
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/RedHatInsights/insights-results-aggregator/metrics"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
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
	log.Println("Request URI: " + request.RequestURI)
	log.Println("Request method: " + request.Method)
	metrics.APIRequests.With(prometheus.Labels{"url": request.RequestURI}).Inc()
	startTime := time.Now()
	nextHandler.ServeHTTP(writer, request)
	duration := time.Since(startTime)
	metrics.APIResponsesTime.With(prometheus.Labels{"url": request.RequestURI}).Observe(float64(duration.Microseconds()))
}

// LogRequest - middleware for loging requests
func (server *HTTPServer) LogRequest(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			logRequestHandler(writer, request, nextHandler)
		})
}

func (server *HTTPServer) mainEndpoint(writer http.ResponseWriter, request *http.Request) {
	responses.SendResponse(writer, responses.BuildOkResponse())
}

func (server *HTTPServer) listOfOrganizations(writer http.ResponseWriter, request *http.Request) {
	organizations, err := server.Storage.ListOfOrgs()
	if err != nil {
		log.Println("Unable to get list of organizations", err)
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

	clusters, err := server.Storage.ListOfClustersForOrg(types.OrgID(organizationID))
	if err != nil {
		log.Println("Unable to get list of clusters", err)
		responses.SendInternalServerError(writer, err.Error())
	} else {
		responses.SendResponse(writer, responses.BuildOkResponseWithData("clusters", clusters))
	}
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

	report, err := server.Storage.ReadReportForCluster(organizationID, clusterName)
	if _, ok := err.(*storage.ItemNotFoundError); ok {
		responses.Send(http.StatusNotFound, writer, err.Error())
	} else if err != nil {
		log.Println("Unable to read report for cluster", err)
		responses.SendInternalServerError(writer, err.Error())
	} else {
		responses.SendResponse(writer, responses.BuildOkResponseWithData("report", report))
	}
}

// serveAPISpecFile serves an OpenAPI specifications file specified in config file
func (server HTTPServer) serveAPISpecFile(writer http.ResponseWriter, request *http.Request) {
	absPath, err := filepath.Abs(server.Config.APISpecFile)
	if err != nil {
		log.Printf("Error creating absolute path of OpenAPI spec file. %v", err)
		responses.Send(
			http.StatusInternalServerError,
			writer,
			"Error creating absolute path of OpenAPI spec file",
		)
		return
	}

	http.ServeFile(writer, request, absPath)
}

// Initialize perform the server initialization
func (server *HTTPServer) Initialize(address string) http.Handler {
	log.Println("Initializing HTTP server at", address)

	router := mux.NewRouter().StrictSlash(true)
	router.Use(server.LogRequest)
	if !server.Config.Debug {
		router.Use(server.Authentication)
	}

	// common REST API endpoints
	router.HandleFunc(server.Config.APIPrefix, server.mainEndpoint).Methods(http.MethodGet)
	router.HandleFunc(server.Config.APIPrefix+"organizations", server.listOfOrganizations).Methods(http.MethodGet)
	router.HandleFunc(server.Config.APIPrefix+"organizations/{organization}/clusters", server.listOfClustersForOrganization).Methods(http.MethodGet)
	router.HandleFunc(server.Config.APIPrefix+"report/{organization}/{cluster}", server.readReportForCluster).Methods(http.MethodGet)

	// Prometheus metrics
	router.Handle(server.Config.APIPrefix+"metrics", promhttp.Handler()).Methods(http.MethodGet)

	// OpenAPI specs
	router.HandleFunc(server.Config.APIPrefix+filepath.Base(server.Config.APISpecFile), server.serveAPISpecFile).Methods(http.MethodGet)

	return router
}

// Start starts server
func (server *HTTPServer) Start() error {
	address := server.Config.Address
	log.Println("Starting HTTP server at", address)
	router := server.Initialize(address)
	server.Serv = &http.Server{Addr: address, Handler: router}

	err := server.Serv.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Printf("Unable to start HTTP server %v", err)
		return err
	}

	return nil
}

// Stop stops server's execution
func (server *HTTPServer) Stop(ctx context.Context) error {
	return server.Serv.Shutdown(ctx)
}
