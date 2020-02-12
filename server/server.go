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

// Package server contains implementation of REST API server for the
// Insights results aggregator service. In current version, the following
// REST API endpoints are available:
//
// API_PREFIX/organization - list of all organizations (HTTP GET)
// API_PREFIX/cluster/{organization} - list of all clusters for given organizations (HTTP GET)
// API_PREFIX/report/{organization}/{cluster} - insights OCP results for given cluster name (HTTP GET)
//
// Please note that API_PREFIX is part of server configuration (see Configuration)
package server

import (
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/gorilla/mux"

	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// Server represents any REST API HTTP server
type Server interface {
	Start() error
}

// Impl in an implementation of Server interface
type Impl struct {
	Config  Configuration
	Storage storage.Storage
}

// New constructs new implementation of Server interface
func New(configuration Configuration, storage storage.Storage) Server {
	return Impl{Config: configuration,
		Storage: storage,
	}
}

func logRequestHandler(writer http.ResponseWriter, request *http.Request, nextHandler http.Handler) {
	log.Println("Request URI: " + request.RequestURI)
	log.Println("Request method: " + request.Method)
	nextHandler.ServeHTTP(writer, request)
}

// LogRequest - middleware for loging requests
func (server Impl) LogRequest(nextHandler http.Handler) http.Handler {
	return http.HandlerFunc(
		func(writer http.ResponseWriter, request *http.Request) {
			logRequestHandler(writer, request, nextHandler)
		})
}

func (server Impl) mainEndpoint(writer http.ResponseWriter, request *http.Request) {
	// TODO: just a stub!
	io.WriteString(writer, "Hello world!\n")
}

func (server Impl) listOfOrganizations(writer http.ResponseWriter, request *http.Request) {
	organizations, err := server.Storage.ListOfOrgs()
	if err != nil {
		log.Println("Unable to get list of organizations", err)
		responses.SendInternalServerError(writer, err.Error())
	} else {
		responses.SendResponse(writer, responses.BuildOkResponseWithData("organizations", organizations))
	}
}

func (server Impl) readOrganizationID(writer http.ResponseWriter, request *http.Request) (types.OrgID, error) {
	organizationIDParam, found := mux.Vars(request)["organization"]

	if !found {
		// query parameter 'organization' can't be found in request, which might be caused by issue in Gorilla mux
		// (not on client side)
		const message = "Organization ID is not provided"
		log.Println(message)
		responses.SendInternalServerError(writer, message)
		return 0, errors.New(message)
	}

	organizationID, err := strconv.ParseInt(organizationIDParam, 10, 0)
	if err != nil {
		const message = "Wrong organization ID provided"
		log.Println(message, err)
		responses.SendError(writer, err.Error())
		return 0, errors.New(message)
	}

	return types.OrgID(int(organizationID)), nil
}

func (server Impl) readClusterName(writer http.ResponseWriter, request *http.Request) (types.ClusterName, error) {
	clusterName, found := mux.Vars(request)["cluster"]
	if !found {
		// query parameter 'cluster' can't be found in request, which might be caused by issue in Gorilla mux
		// (not on client side)
		const message = "Cluster name is not provided"
		log.Println(message)
		responses.SendInternalServerError(writer, message)
		return types.ClusterName(""), errors.New(message)
	}
	// TODO: add check for GUID-like name
	return types.ClusterName(clusterName), nil
}

func (server Impl) listOfClustersForOrganization(writer http.ResponseWriter, request *http.Request) {
	organizationID, err := server.readOrganizationID(writer, request)
	if err != nil {
		// everything has been handled already
		return
	}

	clusters, err := server.Storage.ListOfClustersForOrg(types.OrgID(int(organizationID)))
	if err != nil {
		log.Println("Unable to get list of clusters", err)
		responses.SendInternalServerError(writer, err.Error())
	} else {
		responses.SendResponse(writer, responses.BuildOkResponseWithData("clusters", clusters))
	}
}

func (server Impl) readReportForCluster(writer http.ResponseWriter, request *http.Request) {
	organizationID, err := server.readOrganizationID(writer, request)
	if err != nil {
		// everything has been handled already
		return
	}

	clusterName, err := server.readClusterName(writer, request)
	if err != nil {
		// everything has been handled already
		return
	}

	// TODO: error is not reported if cluster does not exist
	report, err := server.Storage.ReadReportForCluster(organizationID, clusterName)
	if err != nil {
		log.Println("Unable to read report for cluster", err)
		responses.SendInternalServerError(writer, err.Error())
	} else {
		responses.SendResponse(writer, responses.BuildOkResponseWithData("report", report))
	}
}

// Initialize perform the server initialization
func (server Impl) Initialize(address string) http.Handler {
	log.Println("Initializing HTTP server at", address)

	router := mux.NewRouter().StrictSlash(true)
	router.Use(server.LogRequest)

	// common REST API endpoints
	router.HandleFunc(server.Config.APIPrefix, server.mainEndpoint).Methods("GET")
	router.HandleFunc(server.Config.APIPrefix+"organization", server.listOfOrganizations).Methods("GET")
	router.HandleFunc(server.Config.APIPrefix+"cluster/{organization}", server.listOfClustersForOrganization).Methods("GET")
	router.HandleFunc(server.Config.APIPrefix+"report/{organization}/{cluster}", server.readReportForCluster).Methods("GET")
	return router
}

// Start starts server
func (server Impl) Start() error {
	address := server.Config.Address
	router := server.Initialize(address)
	log.Println("Starting HTTP server at", address)

	err := http.ListenAndServe(address, router)
	if err != nil {
		log.Fatal("Unable to start HTTP server", err)
	}
	return nil
}
