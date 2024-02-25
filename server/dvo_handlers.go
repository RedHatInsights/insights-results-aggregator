/*
Copyright © 2024 Red Hat, Inc.

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
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/RedHatInsights/insights-operator-utils/generators"
	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

const (
	namespaceIDParam     = "namespace"
	RecommendationSuffix = ".recommendation"
)

// Cluster structure contains cluster UUID and cluster name
type Cluster struct {
	UUID        string `json:"uuid"`
	DisplayName string `json:"display_name"`
}

// Namespace structure contains basic information about namespace
type Namespace struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

// Metadata structure contains basic information about workload metadata
type Metadata struct {
	Recommendations int         `json:"recommendations"`
	Objects         int         `json:"objects"`
	ReportedAt      string      `json:"reported_at"`
	LastCheckedAt   string      `json:"last_checked_at"`
	HighestSeverity int         `json:"highest_severity"`
	HitsBySeverity  map[int]int `json:"hits_by_severity"`
}

// WorkloadsForNamespace structure represents a single entry of the namespace list with some aggregations
type WorkloadsForNamespace struct {
	Cluster                 Cluster        `json:"cluster"`
	Namespace               Namespace      `json:"namespace"`
	Metadata                Metadata       `json:"metadata"`
	RecommendationsHitCount map[string]int `json:"recommendations_hit_count"`
}

// WorkloadsForCluster structure represents workload for one selected cluster
type WorkloadsForCluster struct {
	Status          string              `json:"status"`
	Cluster         Cluster             `json:"cluster"`
	Namespace       Namespace           `json:"namespace"`
	Metadata        Metadata            `json:"metadata"`
	Recommendations []DVORecommendation `json:"recommendations"`
}

// DVORecommendation structure represents one DVO-related recommendation
type DVORecommendation struct {
	Check        string                 `json:"check"`
	Details      string                 `json:"details"`
	Resolution   string                 `json:"resolution"`
	Modified     string                 `json:"modified"`
	MoreInfo     string                 `json:"more_info"`
	TemplateData map[string]interface{} `json:"extra_data"`
	Objects      []DVOObject            `json:"objects"`
}

// DVOObject structure
type DVOObject struct {
	Kind string `json:"kind"`
	UID  string `json:"uid"`
}

// readNamespace retrieves namespace UUID from request
// if it's not possible, it writes http error to the writer and returns error
func readNamespace(writer http.ResponseWriter, request *http.Request) (
	namespace string, err error,
) {
	namespaceID, err := httputils.GetRouterParam(request, namespaceIDParam)
	if err != nil {
		handleServerError(writer, err)
		return
	}

	validatedNamespaceID, err := validateNamespaceID(namespaceID)
	if err != nil {
		err = &RouterParsingError{
			ParamName:  namespaceIDParam,
			ParamValue: namespaceID,
			ErrString:  err.Error(),
		}
		handleServerError(writer, err)
		return
	}

	return validatedNamespaceID, nil
}

func validateNamespaceID(namespace string) (string, error) {
	if _, err := uuid.Parse(namespace); err != nil {
		message := fmt.Sprintf("invalid namespace ID: '%s'. Error: %s", namespace, err.Error())

		log.Error().Err(err).Msg(message)

		return "", &RouterParsingError{
			ParamName:  namespaceIDParam,
			ParamValue: namespace,
			ErrString:  err.Error(),
		}
	}

	return namespace, nil
}

// getWorkloads retrieves all namespaces and workloads for given organization
func (server *HTTPServer) getWorkloads(writer http.ResponseWriter, request *http.Request) {
	tStart := time.Now()

	// extract org_id from URL
	orgID, ok := readOrgID(writer, request)
	if !ok {
		// everything has been handled
		return
	}
	log.Debug().Int(orgIDStr, int(orgID)).Msg("getWorkloads")

	workloads, err := server.StorageDvo.ReadWorkloadsForOrganization(orgID)
	if err != nil {
		log.Error().Err(err).Msg("Errors retrieving DVO workload recommendations from storage")
		handleServerError(writer, err)
		return
	}

	processedWorkloads := server.processDVOWorkloads(workloads)

	log.Info().Uint32(orgIDStr, uint32(orgID)).Msgf(
		"getWorkloads took %s", time.Since(tStart),
	)
	err = responses.SendOK(writer, responses.BuildOkResponseWithData("workloads", processedWorkloads))
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

func (server *HTTPServer) processDVOWorkloads(workloads []types.DVOReport) (
	processedWorkloads []WorkloadsForNamespace,
) {
	for _, workload := range workloads {
		processedWorkloads = append(processedWorkloads, WorkloadsForNamespace{
			Cluster: Cluster{
				UUID: workload.ClusterID,
			},
			Namespace: Namespace{
				UUID: workload.NamespaceID,
				Name: workload.NamespaceName,
			},
			Metadata: Metadata{
				Recommendations: int(workload.Recommendations),
				Objects:         int(workload.Objects),
				ReportedAt:      string(workload.ReportedAt),
				LastCheckedAt:   string(workload.LastCheckedAt),
			},
			// TODO: fill RecommendationsHitCount map efficiently instead of processing the report again every time
		})
	}

	return
}

// getWorkloadsForNamespace retrieves data about a single namespace within a cluster
func (server *HTTPServer) getWorkloadsForNamespace(writer http.ResponseWriter, request *http.Request) {
	tStart := time.Now()

	orgID, ok := readOrgID(writer, request)
	if !ok {
		// everything has been handled
		return
	}

	clusterName, successful := readClusterName(writer, request)
	if !successful {
		// everything has been handled already
		return
	}

	namespaceID, err := readNamespace(writer, request)
	if err != nil {
		return
	}

	log.Debug().Int(orgIDStr, int(orgID)).Str("namespaceID", namespaceID).Msgf("getWorkloadsForNamespace cluster %v", clusterName)

	workload, err := server.StorageDvo.ReadWorkloadsForClusterAndNamespace(orgID, clusterName, namespaceID)
	if err != nil {
		log.Error().Err(err).Msg("Errors retrieving DVO workload recommendations from storage")
		handleServerError(writer, err)
		return
	}

	processedWorkload := server.processSingleDVONamespace(workload)

	log.Info().Uint32(orgIDStr, uint32(orgID)).Msgf(
		"getWorkloadsForNamespace -\n\n\n\n\n\n\n %v", processedWorkload,
	)
	log.Info().Uint32(orgIDStr, uint32(orgID)).Msgf(
		"getWorkloadsForNamespace took %s", time.Since(tStart),
	)
	err = responses.SendOK(writer, responses.BuildOkResponseWithData("workloads", processedWorkload))
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// processSingleDVONamespace processes a report, filters out mismatching namespaces, returns processed results
func (server *HTTPServer) processSingleDVONamespace(workload types.DVOReport) (
	processedWorkloads WorkloadsForCluster,
) {
	processedWorkloads = WorkloadsForCluster{
		Cluster: Cluster{
			UUID: workload.ClusterID,
		},
		Namespace: Namespace{
			UUID: workload.NamespaceID,
			Name: workload.NamespaceName,
		},
		Metadata: Metadata{
			Recommendations: int(workload.Recommendations),
			Objects:         int(workload.Objects),
			ReportedAt:      string(workload.ReportedAt),
			LastCheckedAt:   string(workload.LastCheckedAt),
		},
		Recommendations: []DVORecommendation{},
	}

	var dvoReport types.DVOMetrics
	// remove doubled escape characters due to improper encoding during storage
	s := strings.Replace(workload.Report, "\\", "", -1)
	json.Unmarshal(json.RawMessage(s), &dvoReport)

	for _, recommendation := range dvoReport.WorkloadRecommendations {
		filteredObjects := make([]DVOObject, 0)
		for i := range recommendation.Workloads {
			object := &recommendation.Workloads[i]

			// filter out other namespaces
			if object.NamespaceUID != processedWorkloads.Namespace.UUID {
				continue
			}
			filteredObjects = append(filteredObjects, DVOObject{
				Kind: object.Kind,
				UID:  object.UID,
			})
		}

		compositeRuleID, err := generators.GenerateCompositeRuleID(
			// for some unknown reason, there's a `.recommendation` suffix for each rule hit. no idea why...
			types.RuleFQDN(strings.TrimSuffix(recommendation.Component, RecommendationSuffix)),
			types.ErrorKey(recommendation.Key),
		)
		if err != nil {
			log.Error().Err(err).Msgf("error generating composite rule ID for rule [%+v]", recommendation)
			continue
		}

		processedWorkloads.Recommendations = append(processedWorkloads.Recommendations, DVORecommendation{
			Check:        string(compositeRuleID),
			Objects:      filteredObjects,
			TemplateData: recommendation.Details,
		})
	}

	return
}
