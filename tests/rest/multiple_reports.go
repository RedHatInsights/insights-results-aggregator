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

package tests

import (
	"encoding/json"
	"fmt"
	"strings"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	//"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/verdverm/frisby"

	"github.com/RedHatInsights/insights-results-aggregator/server"
	//"github.com/RedHatInsights/insights-results-aggregator/types"
)

// MultipleReportsResponse represents response from the server that contains
// results for multiple clusters together with overall status
type MultipleReportsResponse struct {
	Clusters    []string               `json:"clusters"`
	Errors      []string               `json:"errors"`
	Reports     map[string]interface{} `json:"reports"`
	GeneratedAt string                 `json:"generated_at"`
	Status      string                 `json:"status"`
}

// constructURLForReportForOrgClusters function construct an URL to access the
// endpoint to retrieve results for given list of clusters
func constructURLForReportForOrgClusters(organizationID string, clusterIDs []string) string {
	cls := strings.Join(clusterIDs, "'")
	return httputils.MakeURLToEndpoint(apiURL, server.ReportForListOfClustersEndpoint, organizationID, cls)
}

// readMultipleReportsResponse reads and parses response body that should
// contains reports for multiple clusters
func readMultipleReportsResponse(f *frisby.Frisby) MultipleReportsResponse {
	response := MultipleReportsResponse{}

	// try to read response body
	text, err := f.Resp.Content()
	if err != nil {
		f.AddError(err.Error())
	} else {
		// try to deserialize response body
		err := json.Unmarshal(text, &response)
		if err != nil {
			f.AddError(err.Error())
		}
	}
	return response
}

// expectNumberOfClusters utility function checks if server response contains
// expected number of clusters
func expectNumberOfClusters(f *frisby.Frisby, response MultipleReportsResponse, expected int) {
	clusters := response.Clusters
	actual := len(clusters)
	if actual != expected {
		f.AddError(fmt.Sprintf("expected %d clusters in server response, but got %d instead", expected, actual))
	}
}

// expectNumberOfClusters utility function checks if server response contains
// expected number of errors
func expectNumberOfErrors(f *frisby.Frisby, response MultipleReportsResponse, expected int) {
	clusters := response.Errors
	actual := len(clusters)
	if actual != expected {
		f.AddError(fmt.Sprintf("expected %d errors in server response, but got %d instead", expected, actual))
	}
}

// expectNumberOfReports utility function checks if server response contains
// expected number of errors
func expectNumberOfReports(f *frisby.Frisby, response MultipleReportsResponse, expected int) {
	clusters := response.Reports
	actual := len(clusters)
	if actual != expected {
		f.AddError(fmt.Sprintf("expected %d reports in server response, but got %d instead", expected, actual))
	}
}

// checkOkStatus tests whether the response (JSON) contains status attribute set to 'ok'
func expectOkStatus(f *frisby.Frisby, response MultipleReportsResponse) {
	if response.Status == "ok" {
		f.AddError(fmt.Sprintf("Expected error status, but got '%s' instead", response.Status))
	}
}
