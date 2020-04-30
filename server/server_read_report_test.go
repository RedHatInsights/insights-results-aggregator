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
	"database/sql"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/tests/testdata"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

func TestReadReportForClusterNonIntOrgID(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{"non-int", testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body: `{
			"status": "Error during parsing param 'organization' with value 'non-int'. Error: 'unsigned integer expected'"
		}`,
	})
}

func TestReadReportForClusterNegativeOrgID(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{-1, testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body: `{
			"status": "Error during parsing param 'organization' with value '-1'. Error: 'unsigned integer expected'"
		}`,
	})
}

func TestReadReportForClusterBadClusterName(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.BadClusterName},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body:       `{"status": "Error during parsing param 'cluster' with value 'aaaa'. Error: 'invalid UUID length: 4'"}`,
	})
}

func TestReadNonExistingReport(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode: http.StatusNotFound,
		Body: fmt.Sprintf(
			`{"status":"Item with ID %v/%v was not found in the storage"}`, testdata.OrgID, testdata.ClusterName,
		),
	})
}

func TestHttpServer_readReportForCluster_NoContent(t *testing.T) {
	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.LastCheckedAt,
	)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body: `{
			"status":"ok",
			"report": {
				"meta": {
					"count": 0,
					"last_checked_at": "` + testdata.LastCheckedAt.Format(time.RFC3339) + `"
				},
				"data":[]
			}
		}`,
	})
}

func TestHttpServer_readReportForCluster_NoRules(t *testing.T) {
	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report0Rules, testdata.LastCheckedAt,
	)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body: `{
			"status":"ok",
			"report": {
				"meta": {
					"count": -1,
					"last_checked_at": "` + testdata.LastCheckedAt.Format(time.RFC3339) + `"
				},
				"data":[]
			}
		}`,
	})
}

func TestReadReportDBError(t *testing.T) {
	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	closer()

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       `{"status":"Internal Server Error"}`,
	})
}

func TestHttpServer_readReportForCluster_getContentForRule_DBError(t *testing.T) {
	connection, err := sql.Open("sqlite3", ":memory:")
	helpers.FailOnError(t, err)

	mockStorage := storage.NewFromConnection(connection, storage.DBDriverSQLite3)
	defer helpers.MustCloseStorage(t, mockStorage)

	err = mockStorage.Init()
	helpers.FailOnError(t, err)

	// remove table to cause db error
	_, err = connection.Exec("DROP TABLE rule_error_key;")
	helpers.FailOnError(t, err)

	err = mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.LastCheckedAt,
	)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       `{ "status":"Internal Server Error" }`,
	})
}

func TestHttpServer_readReportForCluster_getContentForRule_BadReport(t *testing.T) {
	const badReport = "not-json"

	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, badReport, testdata.LastCheckedAt,
	)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body:       `{ "status": "invalid character 'o' in literal null (expecting 'u')" }`,
	})
}

func assertReportResponsesEqual(t testing.TB, expected, got string) {
	var expectedResponse, gotResponse struct {
		Status string               `json:"status"`
		Report types.ReportResponse `json:"report"`
	}

	err := helpers.JSONUnmarshalStrict([]byte(expected), &expectedResponse)
	helpers.FailOnError(t, err)
	err = helpers.JSONUnmarshalStrict([]byte(got), &gotResponse)
	helpers.FailOnError(t, err)

	assert.NotEmpty(
		t,
		expectedResponse.Status,
		"status is empty(probably json is completely wrong and unmarshal didn't do anything useful)",
	)
	assert.Equal(t, expectedResponse.Status, gotResponse.Status)
	assert.Equal(t, expectedResponse.Report.Meta, gotResponse.Report.Meta)
	// ignore the order
	assert.ElementsMatch(t, expectedResponse.Report.Rules, gotResponse.Report.Rules)
}

func TestReadReportWithContent(t *testing.T) {
	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.Report3Rules,
		testdata.LastCheckedAt,
	)
	helpers.FailOnError(t, err)

	// write some rule content into the DB
	err = mockStorage.LoadRuleContent(testdata.RuleContent3Rules)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report3RulesExpectedResponse,
		BodyChecker: assertReportResponsesEqual,
	})
}

// TestReadReportDisableRule reads a report, disables the first rule, fetches again,
// expecting the rule to be last and disabled, re-enables it and expects regular
// response with Rule1 first again
func TestReadReportDisableRule(t *testing.T) {
	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.Report2Rules,
		testdata.LastCheckedAt,
	)
	helpers.FailOnError(t, err)

	err = mockStorage.LoadRuleContent(testdata.RuleContent3Rules)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report2RulesEnabledRule1ExpectedResponse,
		BodyChecker: assertReportResponsesEqual,
	})

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.DisableRuleForClusterEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID},
		UserID:       testdata.UserID,
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"status": "ok"}`,
	})

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report2RulesDisabledRule1ExpectedResponse,
		BodyChecker: assertReportResponsesEqual,
	})

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.EnableRuleForClusterEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID},
		UserID:       testdata.UserID,
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"status": "ok"}`,
	})

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report2RulesEnabledRule1ExpectedResponse,
		BodyChecker: assertReportResponsesEqual,
	})
}
