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

// THis package contains tests for the following endpoint:
//
// apiPrefix+"/rules/{rule_selector}/organizations/{org_id}/users/{user_id}/clusters_detail"

package tests

import (
	"encoding/json"
	"fmt"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	server "github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/stretchr/testify/assert"
	"github.com/verdverm/frisby"
)

// ClusterInfoResponse represents the response containing the cluster information and status
type ClusterInfoResponse struct {
	Clusters []ClusterInfo `json:"clusters"`
	Status   string        `json:"status"`
}

// ClusterInfoMeta represents the cluster metadata information
type ClusterInfoMeta struct {
	Version string `json:"cluster_version"`
}

// ClusterInfo represents the cluster information
type ClusterInfo struct {
	Cluster string `json:"cluster"`
	// Name     string `json:"cluster_name"`
	// LastSeen string `json:"last_checked_at"`
	Meta ClusterInfoMeta `json:"meta"`
}

// constructURLForClustersDetail function constructs an URL to access
// the endpoint to retrieve results metadata for given cluster from selected
// organization and rule selector
func constructURLForClustersDetail(
	organizationID string, ruleSelector types.RuleSelector, userID types.UserID) string {
	return httputils.MakeURLToEndpoint(apiURL,
		server.RuleClusterDetailEndpoint, ruleSelector, organizationID, userID)
}

func checkResponseContent(f *frisby.Frisby, expectedResponse ClusterInfoResponse) {
	text, err := f.Resp.Content()
	if err != nil {
		f.AddError(err.Error())
	} else {
		response := ClusterInfoResponse{}
		err := json.Unmarshal(text, &response)
		if err != nil {
			f.AddError(err.Error())
		}

		if response.Status != expectedResponse.Status {
			f.AddError(fmt.Sprintf("Expecting 'status' to be set to %q but got %q", expectedResponse.Status, response.Status))
		}

		equal := assert.ObjectsAreEqualValues(expectedResponse.Clusters, response.Clusters)
		if !equal {
			f.AddError(fmt.Sprintf("Expecting clusters to be set to\n%v\nbut got\n%v", expectedResponse.Clusters, response.Clusters))
		}
	}
	f.PrintReport()
}

// checkClustersDetailEndpointGet check if the endpoint to return the cluster detail works as expected with GET method.
// For this test there is this data:
// | Organization | Cluster                              | Rule                                                      | Version            |
// |--------------|--------------------------------------|-----------------------------------------------------------|--------------------|
// | 1            | 11111111-1111-1111-1111-111111111111 | ccx_rules_ocp.external.rules.node_installer_degraded|ek1  | 1.0                |
// | 2            | 22222222-2222-2222-2222-222222222222 | ccx_rules_ocp.external.rules.node_installer_degraded|ek1  | (empty)            |
// | 3            | 33333333-3333-3333-3333-333333333333 | ccx_rules_ocp.external.rules.node_installer_degraded|ek1  | not in report_info |
func checkClustersEndpointDetailGet() {
	type testCase struct {
		name             string
		organizationID   string
		ruleSelector     types.RuleSelector
		expectStatus     int
		expectedResponse ClusterInfoResponse
	}

	table := []testCase{
		{
			name:           "Check the endpoint to return cluster information: known organization and rule found",
			expectStatus:   200,
			organizationID: "1",
			ruleSelector:   types.RuleSelector(testdata.Rule1CompositeID),
			expectedResponse: ClusterInfoResponse{
				Status: "ok",
				Clusters: []ClusterInfo{
					{
						Cluster: "11111111-1111-1111-1111-111111111111",
						Meta:    ClusterInfoMeta{Version: "1.0"},
					},
				},
			},
		},
		{
			name:           "Check the endpoint to return cluster information: known organization and rule didn't find any clusters",
			expectStatus:   404,
			organizationID: "1",
			ruleSelector:   types.RuleSelector(testdata.Rule2CompositeID),
			expectedResponse: ClusterInfoResponse{
				Status: fmt.Sprintf("Item with ID %s was not found in the storage", testdata.Rule2CompositeID),
			},
		},
		{
			name:           "Check the endpoint to return cluster information: known organization and version is empty",
			expectStatus:   200,
			organizationID: "2",
			ruleSelector:   types.RuleSelector(testdata.Rule1CompositeID),
			expectedResponse: ClusterInfoResponse{
				Status: "ok",
				Clusters: []ClusterInfo{
					{
						Cluster: "22222222-2222-2222-2222-222222222222",
					},
				},
			},
		},
		{
			name:           "Check the endpoint to return cluster information: known organization and there is no version in report_info",
			expectStatus:   200,
			organizationID: "3",
			ruleSelector:   types.RuleSelector(testdata.Rule1CompositeID),
			expectedResponse: ClusterInfoResponse{
				Status: "ok",
				Clusters: []ClusterInfo{
					{
						Cluster: "33333333-3333-3333-3333-333333333333",
					},
				},
			},
		},
		{
			name:           "Check the endpoint to return cluster information: unknown organization",
			expectStatus:   404,
			organizationID: "24",
			ruleSelector:   types.RuleSelector(testdata.Rule1CompositeID),
			expectedResponse: ClusterInfoResponse{
				Status: fmt.Sprintf("Item with ID %s was not found in the storage", testdata.Rule1CompositeID),
			},
		},
		{
			name:           "Check the endpoint to return cluster information: known organization invalid rule selector",
			expectStatus:   400,
			organizationID: "1",
			ruleSelector:   types.RuleSelector("not a rule"),
			expectedResponse: ClusterInfoResponse{
				Status: fmt.Sprintf("Error during parsing param 'rule_selector' with value '%s'. Error: 'Param rule_selector is not a valid rule selector (plugin_name|error_key)'", "not a rule"),
			},
		},
	}

	for _, tc := range table {
		url := constructURLForClustersDetail(tc.organizationID, tc.ruleSelector, testdata.UserID)
		f := frisby.Create(tc.name).Get(url)
		setAuthHeader(f)
		f.Send()
		f.ExpectStatus(tc.expectStatus)
		f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
		f.PrintReport()

		checkResponseContent(f, tc.expectedResponse)

		if tc.expectStatus != 200 {
			statusResponse := readStatusFromResponse(f)
			checkErrorStatusResponse(f, statusResponse)
		}
	}

}

// checkReportInfoEndpointWrongMethods check if the endpoint to return results
// responds correctly to other methods than HTTP GET
func checkClustersDetailsEndpointWrongMethods() {
	url := constructURLForClustersDetail("1", types.RuleSelector(testdata.Rule1CompositeID), testdata.UserID)
	checkGetEndpointByOtherMethods(url, false)
}

// checkClustersDetailsEndpointForKnownOrganizationNoAuthHeaderCase
// check if the endpoint to return report metadata works as expected.
// This test variant does not sent authorization header
func checkClustersDetailsEndpointForKnownOrganizationNoAuthHeaderCase() {
	url := constructURLForClustersDetail("1", types.RuleSelector(testdata.Rule1CompositeID), testdata.UserID)
	f := frisby.Create("Check the endpoint to return cluster information: no authorization header").Get(url)
	f.Send()
	f.ExpectStatus(401)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
	f.PrintReport()

	statusResponse := readStatusFromResponse(f)
	checkErrorStatusResponse(f, statusResponse)
}
