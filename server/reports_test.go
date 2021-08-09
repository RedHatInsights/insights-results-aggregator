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

package server_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"

	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// TestValidateClusterID checks the helper function validateClusterID
func TestValidateClusterID(t *testing.T) {
	err1 := server.ValidateClusterID("")
	assert.EqualError(t, err1, "invalid cluster ID: ''. Error: invalid UUID length: 0")

	err2 := server.ValidateClusterID("foobar")
	assert.EqualError(t, err2, "invalid cluster ID: 'foobar'. Error: invalid UUID length: 6")

	err3 := server.ValidateClusterID("34c3ecc5-624a-49a5-bab8-4fdc5e51a26Z")
	assert.EqualError(t, err3, "invalid cluster ID: '34c3ecc5-624a-49a5-bab8-4fdc5e51a26Z'. Error: invalid UUID format")

	err4 := server.ValidateClusterID("34c3ecc5-624a-49a5-bab8-4fdc5e51a266")
	assert.Nil(t, err4)
}

// TestConstructClusterNames checks the helper function constructClusterNames
func TestConstructClusterNames(t *testing.T) {
	const cn1 = "cn1"
	const cn2 = "cn2"

	cnames0 := []string{}
	res0 := server.ConstructClusterNames(cnames0)
	assert.Equal(t, len(res0), 0)

	cnames1 := []string{cn1}
	res1 := server.ConstructClusterNames(cnames1)
	assert.Equal(t, len(res1), 1)
	assert.Equal(t, res1[0], types.ClusterName(cn1))

	cnames2 := []string{cn1, cn2}
	res2 := server.ConstructClusterNames(cnames2)
	assert.Equal(t, len(res2), 2)
	assert.Equal(t, res2[0], types.ClusterName(cn1))
	assert.Equal(t, res2[1], types.ClusterName(cn2))
}

// TestFillInGeneratedReportsEmptySet check the function fillInGeneratedReports
func TestFillInGeneratedReportsEmptySet(t *testing.T) {
	clusterNames := server.ConstructClusterNames([]string{})
	reports := make(map[types.ClusterName]types.ClusterReport)
	generatedReport := server.FillInGeneratedReports(clusterNames, reports)

	// no errors, no clusters, and no reports are expected
	assert.Equal(t, len(generatedReport.Errors), 0)
	assert.Equal(t, len(generatedReport.ClusterList), 0)
	assert.Equal(t, len(generatedReport.Reports), 0)
}

// TestFillInGeneratedReportsNoResults check the function fillInGeneratedReports
func TestFillInGeneratedReportsNoResults(t *testing.T) {
	clusterNames := []types.ClusterName{testdata.ClusterName}
	reports := make(map[types.ClusterName]types.ClusterReport)
	generatedReport := server.FillInGeneratedReports(clusterNames, reports)

	// one cluster with error is expected
	assert.Equal(t, len(generatedReport.Errors), 1)
	assert.Equal(t, len(generatedReport.ClusterList), 0)
	assert.Equal(t, len(generatedReport.Reports), 0)
}

// TestFillInGeneratedReportsOneResult check the function fillInGeneratedReports
func TestFillInGeneratedReportsOneResult(t *testing.T) {
	clusterNames := []types.ClusterName{testdata.ClusterName}
	reports := make(map[types.ClusterName]types.ClusterReport)
	reports[testdata.ClusterName] = "{}"
	generatedReport := server.FillInGeneratedReports(clusterNames, reports)

	// one cluster without error is expected
	assert.Equal(t, len(generatedReport.Errors), 0)
	assert.Equal(t, len(generatedReport.ClusterList), 1)
	assert.Equal(t, len(generatedReport.Reports), 1)
}

// TestFillInGeneratedReportsImproperJSON check the function fillInGeneratedReports
func TestFillInGeneratedReportsImproperJSON(t *testing.T) {
	clusterNames := []types.ClusterName{testdata.ClusterName}
	reports := make(map[types.ClusterName]types.ClusterReport)
	reports[testdata.ClusterName] = "this{}is{}not{}json"
	generatedReport := server.FillInGeneratedReports(clusterNames, reports)

	// one cluster without error is expected
	assert.Equal(t, len(generatedReport.Errors), 1)
	assert.Equal(t, len(generatedReport.ClusterList), 0)
	assert.Equal(t, len(generatedReport.Reports), 0)
}

// TestReadReportsForClustersNegativeOrgID check if wrong organization ID is
// handled by ReportForListOfClustersEndpoint handler.
func TestReadReportsForClustersNegativeOrgID(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, nil, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportForListOfClustersEndpoint,
		EndpointArgs: []interface{}{-1, testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body: `{
			"status":"Error during parsing param 'org_id' with value '-1'. Error: 'unsigned integer expected'"
		}`,
	})
}

// TestReadReportsForClustersUnknownCluster check if unknown cluster ID is
// handled by ReportForListOfClustersEndpoint handler.
func TestReadReportsForClustersUnknownCluster(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, nil, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportForListOfClustersEndpoint,
		EndpointArgs: []interface{}{1, "not a real cluster"},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body: `{
			"status":"invalid cluster ID: 'not a real cluster'. Error: invalid UUID length: 18"
		}`,
	})
}

// TestReadReportsForClustersKnownCluster check if unknown cluster ID is
// handled by ReportForListOfClustersEndpoint handler.
func TestReadReportsForClustersKnownCluster(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, nil, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportForListOfClustersEndpoint,
		EndpointArgs: []interface{}{1, testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body: `{
			"clusters": null,"errors": ["84f7eedc-0dd8-49cd-9d4d-f6646df3a5bc"],"reports": {},"generated_at": "","status": "ok"
		}`,
	})
}

// TestReadReportsForClustersTwoClusters check if unknown cluster ID is
// handled by ReportForListOfClustersEndpoint handler.
func TestReadReportsForClustersTwoClusters(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, nil, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportForListOfClustersEndpoint,
		EndpointArgs: []interface{}{1, testdata.ClusterName + "," + testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body: `{
			"clusters": null,"errors": ["84f7eedc-0dd8-49cd-9d4d-f6646df3a5bc","84f7eedc-0dd8-49cd-9d4d-f6646df3a5bc"],"reports": {},"generated_at": "","status": "ok"
		}`,
	})
}

// TestReadReportsForClustersPayloadNegativeOrgID check if wrong organization
// ID is handled by ReportForListOfClustersPayloadEndpoint handler.
func TestReadReportsForClustersPayloadNegativeOrgID(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, nil, &helpers.APIRequest{
		Method:       http.MethodPost,
		Endpoint:     server.ReportForListOfClustersPayloadEndpoint,
		EndpointArgs: []interface{}{-1},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body: `{
			"status":"Error during parsing param 'org_id' with value '-1'. Error: 'unsigned integer expected'"
		}`,
	})
}

// TestReadReportsForClustersPayloadPositiveOrgID check if wrong organization
// ID is handled by ReportForListOfClustersPayloadEndpoint handler.
func TestReadReportsForClustersPayloadPositiveOrgID(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, nil, &helpers.APIRequest{
		Method:       http.MethodPost,
		Endpoint:     server.ReportForListOfClustersPayloadEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body: `{
			"status":"client didn't provide request body"
		}`,
	})
}
