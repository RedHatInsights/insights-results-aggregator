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

package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const includeTimestamp = false

// validateClusterID function checks if the cluster ID is a valid UUID.
func validateClusterID(clusterID string) error {
	_, err := uuid.Parse(clusterID)
	if err != nil {
		message := fmt.Sprintf("invalid cluster ID: '%s'. Error: %s", clusterID, err.Error())
		return errors.New(message)
	}

	// cluster ID seems to be in UUID format
	return nil
}

// sendWrongClusterIDResponse function sends response to client when
// bad/improper cluster ID is detected
func sendWrongClusterIDResponse(writer http.ResponseWriter, err error) {
	log.Error().Err(err).Msg("wrong cluster identifier detected")
	err = responses.SendBadRequest(writer, err.Error())
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// sendWrongClusterOrgIDResponse function sends response to client when
// bad/improper cluster organization ID is detected
func sendWrongClusterOrgIDResponse(writer http.ResponseWriter, orgID types.OrgID) {
	log.Error().Int("organization", int(orgID)).Msg("wrong cluster organization ID detected")
	err := responses.SendBadRequest(writer, "Improper organization ID")
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// sendDBErrorResponse function sends response to client when DB error occurs.
func sendDBErrorResponse(writer http.ResponseWriter, err error) {
	log.Error().Err(err).Msg("try to read reports for clusters")
	err = responses.SendBadRequest(writer, err.Error())
	if err != nil {
		log.Error().Err(err).Msg("read reports for clusters from database")
	}
}

// sendMarshallErrorResponse function sends response to client when marshalling
// error occurs.
func sendMarshallErrorResponse(writer http.ResponseWriter, err error) {
	log.Error().Err(err).Msg("marshalling error")
	err = responses.SendBadRequest(writer, err.Error())
	if err != nil {
		log.Error().Err(err).Msg("marshalling error")
	}
}

// constructClusterNames function constructs array of cluster names from given
// array of strings
func constructClusterNames(clusters []string) []types.ClusterName {
	clusterNames := make([]types.ClusterName, len(clusters))
	for i, clusterName := range clusters {
		clusterNames[i] = types.ClusterName(clusterName)
	}
	return clusterNames
}

// fillInGeneratedReports function constructs data structure
// `types.ClusterReports` and fills it by cluster reports read from database.
func fillInGeneratedReports(clusterNames []types.ClusterName, reports map[types.ClusterName]types.ClusterReport) types.ClusterReports {
	// construct the data structure
	var generatedReports types.ClusterReports

	// prepare its attributes
	// TODO: make sure it is really needed
	if includeTimestamp {
		generatedReports.GeneratedAt = time.Now().UTC().Format(time.RFC3339)
	}
	generatedReports.Reports = make(map[types.ClusterName]json.RawMessage)

	// fill it by real cluster reports
	for _, clusterName := range clusterNames {
		stringReport, ok := reports[clusterName]
		// report for given cluster has been found
		if ok {
			var jsonReport json.RawMessage
			err := json.Unmarshal([]byte(stringReport), &jsonReport)
			if err != nil {
				log.Error().Err(err).Msg("Unable to unmarshal report for cluster")
				generatedReports.Errors = append(generatedReports.Errors, clusterName)
				// if error happens, simply go to the next cluster
				continue
			}
			generatedReports.ClusterList = append(generatedReports.ClusterList, clusterName)
			generatedReports.Reports[clusterName] = jsonReport
		} else {
			generatedReports.Errors = append(generatedReports.Errors, clusterName)
		}
	}

	// it must be ok now
	generatedReports.Status = "ok"

	return generatedReports
}

// processListOfClusters function retrieves list of cluster IDs and process
// them accordingly: check, read report from DB, serialize etc.
func processListOfClusters(server *HTTPServer, writer http.ResponseWriter, request *http.Request, orgID types.OrgID, clusters []string) {
	log.Info().Int("number of clusters", len(clusters)).Str("list", strings.Join(clusters, ", ")).Msg("processListOfClusters")

	// first step: check if all cluster IDs have proper format
	for _, clusterID := range clusters {
		// all clusters should be identified by proper ID
		err := validateClusterID(clusterID)
		if err != nil {
			sendWrongClusterIDResponse(writer, err)
			return
		}
	}
	log.Debug().Msg("all clusters have proper UUID format")

	clusterNames := constructClusterNames(clusters)
	orgIDs, err := server.Storage.ReadOrgIDsForClusters(clusterNames)
	if err != nil {
		log.Error().Err(err).Msg("try to read org IDs for list of clusters")
	}

	// second step: check if all clusters belongs to given organization ID
	for _, id := range orgIDs {
		if id != orgID {
			sendWrongClusterOrgIDResponse(writer, id)
			return
		}
	}
	log.Debug().Msg("all clusters have proper organization ID")

	reports, err := server.Storage.ReadReportsForClusters(clusterNames)
	if err != nil {
		sendDBErrorResponse(writer, err)
		return
	}

	generatedReports := fillInGeneratedReports(clusterNames, reports)

	bytes, err := json.MarshalIndent(generatedReports, "", "\t")
	if err != nil {
		sendMarshallErrorResponse(writer, err)
		return
	}

	err = responses.Send(http.StatusOK, writer, bytes)
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// reportForListOfClusters function returns reports for several clusters that
// all need to belong to one organization specified in request path. List of
// clusters is specified in request path as well which means that clients needs
// to deal with URL limit (around 2000 characters).
func (server *HTTPServer) reportForListOfClusters(writer http.ResponseWriter, request *http.Request) {
	// first thing first - try to read organization ID from request
	orgID, successful := readOrgID(writer, request)
	if !successful {
		// wrong state has been handled already
		return
	}
	log.Info().Int("orgID", int(orgID)).Msg("reportForListOfClusters")

	// try to read list of cluster IDs
	listOfClusters, successful := readClusterListFromPath(writer, request)
	if !successful {
		// wrong state has been handled already
		return
	}

	// we were able to read the cluster IDs, let's process them
	processListOfClusters(server, writer, request, orgID, listOfClusters)
}

// reportForListOfClustersPayload function returns reports for several clusters
// that all need to belong to one organization specified in request path. List
// of clusters is specified in request body which means that clients can use as
// many cluster ID as the wont without any (real) limits.
func (server *HTTPServer) reportForListOfClustersPayload(writer http.ResponseWriter, request *http.Request) {
	// first thing first - try to read organization ID from request
	orgID, successful := readOrgID(writer, request)
	if !successful {
		// wrong state has been handled already
		return
	}
	log.Info().Int("orgID", int(orgID)).Msg("reportForListOfClustersPayload")

	// try to read list of cluster IDs
	listOfClusters, successful := readClusterListFromBody(writer, request)
	if !successful {
		// wrong state has been handled already
		return
	}

	// we were able to read the cluster IDs, let's process them
	processListOfClusters(server, writer, request, orgID, listOfClusters)
}
