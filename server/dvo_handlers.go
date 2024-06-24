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
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/RedHatInsights/insights-operator-utils/generators"
	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	"github.com/RedHatInsights/insights-operator-utils/responses"
	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/rs/zerolog/log"
)

const (
	namespaceIDParam = "namespace"
)

// WorkloadsForCluster structure represents workload for one selected cluster
type WorkloadsForCluster struct {
	Status          string              `json:"status"`
	Cluster         types.Cluster       `json:"cluster"`
	Namespace       types.Namespace     `json:"namespace"`
	Metadata        types.DVOMetadata   `json:"metadata"`
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
	IDValidator := regexp.MustCompile(`^.{1,256}$`)

	if !IDValidator.MatchString(namespace) {
		message := fmt.Sprintf("invalid namespace ID: '%s'", namespace)
		err := errors.New(message)
		log.Error().Err(err).Msg(message)
		return "", err
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

	log.Debug().Uint32(orgIDStr, uint32(orgID)).Msgf(
		"getWorkloads took %s", time.Since(tStart),
	)
	err = responses.SendOK(writer, responses.BuildOkResponseWithData("workloads", workloads))
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
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

	processedWorkload := server.ProcessSingleDVONamespace(workload)

	log.Info().Uint32(orgIDStr, uint32(orgID)).Msgf(
		"getWorkloadsForNamespace took %s", time.Since(tStart),
	)
	err = responses.SendOK(writer, responses.BuildOkResponseWithData("workloads", processedWorkload))
	if err != nil {
		log.Error().Err(err).Msg(responseDataError)
	}
}

// ProcessSingleDVONamespace processes a report, filters out mismatching namespaces, returns processed results
func (server *HTTPServer) ProcessSingleDVONamespace(workload types.DVOReport) (
	processedWorkloads WorkloadsForCluster,
) {
	processedWorkloads = WorkloadsForCluster{
		Cluster: types.Cluster{
			UUID: workload.ClusterID,
		},
		Namespace: types.Namespace{
			UUID: workload.NamespaceID,
			Name: workload.NamespaceName,
		},
		Metadata: types.DVOMetadata{
			Recommendations: int(workload.Recommendations),
			Objects:         int(workload.Objects),
			ReportedAt:      string(workload.ReportedAt),
			LastCheckedAt:   string(workload.LastCheckedAt),
		},
		Recommendations: []DVORecommendation{},
	}

	var report string

	switch string([]rune(workload.Report)[:1]) {
	case `"`:
		// we're dealing with a quoted `"{\"system\":{}}"` string
		// unmarshalling into a string first before unmarshalling into a struct will remove the leading/trailing quotes
		// and also take care of the escaped `\"` quotes and replaces them with valid `"`, producing a valid JSON
		err := json.Unmarshal(json.RawMessage(workload.Report), &report)
		if err != nil {
			log.Error().Err(err).Msgf("report has unknown structure: [%v]", string([]rune(workload.Report)[:100]))
		}
	case `{`:
		// we're dealing with either a valid JSON `{"system":{}}` or a string with escaped
		// quotes `{\"system\":{}}`. Stripping escape chars `\` if any, produces a valid JSON
		report = strings.Replace(workload.Report, `\`, "", -1)
	default:
		log.Error().Msgf("report has unknown structure: [%v]", string([]rune(workload.Report)[:100]))
		return
	}

	var dvoReport types.DVOMetrics
	err := json.Unmarshal([]byte(report), &dvoReport)
	if err != nil {
		log.Error().Err(err).Msgf("error unmarshalling full report: [%v]", string([]rune(report)[:100]))
		log.Info().Msgf("report without escape %v", string([]rune(workload.Report)[:100]))
		return
	}

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

		// because the whole report contains a list of recommendations and each rec. contains
		// a list of objects + namespaces, it can happen that upon filtering the objects to get rid
		// of namespaces that weren't requested, we can end up with 0 hitting objects in that namespace
		if len(filteredObjects) == 0 {
			continue
		}

		// recommendation.ResponseID doesn't contain the full rule ID, so smart-proxy was unable to retrieve content, we need to build it
		compositeRuleID, err := generators.GenerateCompositeRuleID(
			// for some unknown reason, there's a `.recommendation` suffix for each rule hit instead of the usual .report
			types.RuleFQDN(strings.TrimSuffix(recommendation.Component, types.WorkloadRecommendationSuffix)),
			types.ErrorKey(recommendation.Key),
		)
		if err != nil {
			log.Error().Err(err).Msg("error generating composite rule ID for rule")
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
