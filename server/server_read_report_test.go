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
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/server"
	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/tests/testdata"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

func TestReadReportForClusterNonIntOrgID(t *testing.T) {
	ira_helpers.AssertAPIRequest(t, nil, &config, &ira_helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{"non-int", testdata.ClusterName},
	}, &ira_helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body: `{
			"status": "Error during parsing param 'organization' with value 'non-int'. Error: 'unsigned integer expected'"
		}`,
	})
}

func TestReadReportForClusterNegativeOrgID(t *testing.T) {
	ira_helpers.AssertAPIRequest(t, nil, &config, &ira_helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{-1, testdata.ClusterName},
	}, &ira_helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body: `{
			"status": "Error during parsing param 'organization' with value '-1'. Error: 'unsigned integer expected'"
		}`,
	})
}

func TestReadReportForClusterBadClusterName(t *testing.T) {
	ira_helpers.AssertAPIRequest(t, nil, &config, &ira_helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.BadClusterName},
	}, &ira_helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body:       `{"status": "Error during parsing param 'cluster' with value 'aaaa'. Error: 'invalid UUID length: 4'"}`,
	})
}

func TestReadNonExistingReport(t *testing.T) {
	ira_helpers.AssertAPIRequest(t, nil, &config, &ira_helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
	}, &ira_helpers.APIResponse{
		StatusCode: http.StatusNotFound,
		Body: fmt.Sprintf(
			`{"status":"Item with ID %v/%v was not found in the storage"}`, testdata.OrgID, testdata.ClusterName,
		),
	})
}

func TestHttpServer_readReportForCluster_NoRules(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report0Rules, testdata.LastCheckedAt, testdata.KafkaOffset,
	)
	helpers.FailOnError(t, err)

	ira_helpers.AssertAPIRequest(t, mockStorage, &config, &ira_helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
	}, &ira_helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body: `{
			"status":"ok",
			"report": {
				"meta": {
					"count": -1,
					"last_checked_at": "` + testdata.LastCheckedAt.Format(time.RFC3339) + `"
				},
				"reports":[]
			}
		}`,
	})
}

func TestReadReportDBError(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	closer()

	ira_helpers.AssertAPIRequest(t, mockStorage, &config, &ira_helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
	}, &ira_helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       `{"status":"Internal Server Error"}`,
	})
}

func TestHttpServer_readReportForCluster_getContentForRule_BadReport(t *testing.T) {
	const badReport = "not-json"

	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, badReport, testdata.LastCheckedAt, testdata.KafkaOffset,
	)
	helpers.FailOnError(t, err)

	ira_helpers.AssertAPIRequest(t, mockStorage, &config, &ira_helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
	}, &ira_helpers.APIResponse{
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
	if err != nil {
		log.Error().Msg("Error unmarshalling expected value")
	}

	helpers.FailOnError(t, err)
	err = helpers.JSONUnmarshalStrict([]byte(got), &gotResponse)
	if err != nil {
		log.Error().Msg("Error unmarshalling got value")
	}
	helpers.FailOnError(t, err)

	assert.NotEmpty(
		t,
		expectedResponse.Status,
		"status is empty(probably json is completely wrong and unmarshal didn't do anything useful)",
	)
	assert.Equal(t, expectedResponse.Status, gotResponse.Status)
	assert.Equal(t, expectedResponse.Report.Meta, gotResponse.Report.Meta)
	// ignore the order
	assert.ElementsMatch(t, expectedResponse.Report.Report, gotResponse.Report.Report)
}

func TestReadReport(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.Report3Rules,
		testdata.LastCheckedAt,
		testdata.KafkaOffset,
	)
	helpers.FailOnError(t, err)

	ira_helpers.AssertAPIRequest(t, mockStorage, &config, &ira_helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
	}, &ira_helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report3RulesExpectedResponse,
		BodyChecker: assertReportResponsesEqual,
	})
}

// TestReadReportDisableRule reads a report, disables the first rule, fetches again,
// expecting the rule to be last and disabled, re-enables it and expects regular
// response with Rule1 first again
func TestReadReportDisableRule(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.Report2Rules,
		testdata.LastCheckedAt,
		testdata.KafkaOffset,
	)
	helpers.FailOnError(t, err)

	err = mockStorage.LoadRuleContent(testdata.RuleContent3Rules)
	helpers.FailOnError(t, err)

	ira_helpers.AssertAPIRequest(t, mockStorage, &config, &ira_helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
		UserID:       testdata.UserID,
	}, &ira_helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report2RulesEnabledRulesExpectedResponse,
		BodyChecker: assertReportResponsesEqual,
	})

	ira_helpers.AssertAPIRequest(t, mockStorage, &config, &ira_helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.DisableRuleForClusterEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID},
		UserID:       testdata.UserID,
	}, &ira_helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"status": "ok"}`,
	})

	ira_helpers.AssertAPIRequest(t, mockStorage, &config, &ira_helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
		UserID:       testdata.UserID,
	}, &ira_helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report2RulesDisabledRule1ExpectedResponse,
		BodyChecker: assertReportResponsesEqual,
	})

	ira_helpers.AssertAPIRequest(t, mockStorage, &config, &ira_helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.EnableRuleForClusterEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID},
		UserID:       testdata.UserID,
	}, &ira_helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"status": "ok"}`,
	})

	ira_helpers.AssertAPIRequest(t, mockStorage, &config, &ira_helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
		UserID:       testdata.UserID,
	}, &ira_helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report2RulesEnabledRulesExpectedResponse,
		BodyChecker: assertReportResponsesEqual,
	})
}

// TestReadReportDisableRuleMultipleUsers is a reproducer for bug issues 811 and 814
func TestReadReportDisableRuleMultipleUsers(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.Report2Rules,
		testdata.LastCheckedAt,
		testdata.KafkaOffset,
	)
	helpers.FailOnError(t, err)

	err = mockStorage.LoadRuleContent(testdata.RuleContent3Rules)
	helpers.FailOnError(t, err)

	// user 1 check no disabled rules in response
	ira_helpers.AssertAPIRequest(t, mockStorage, &config, &ira_helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
		UserID:       testdata.UserID,
	}, &ira_helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report2RulesEnabledRulesExpectedResponse,
		BodyChecker: assertReportResponsesEqual,
	})

	// user 2 disables rule1
	ira_helpers.AssertAPIRequest(t, mockStorage, &config, &ira_helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.DisableRuleForClusterEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID},
		UserID:       testdata.User2ID,
	}, &ira_helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"status": "ok"}`,
	})

	// user 2 is affected
	ira_helpers.AssertAPIRequest(t, mockStorage, &config, &ira_helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
		UserID:       testdata.User2ID,
	}, &ira_helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report2RulesDisabledRule1ExpectedResponse,
		BodyChecker: assertReportResponsesEqual,
	})

	// user 1 is not affected
	ira_helpers.AssertAPIRequest(t, mockStorage, &config, &ira_helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
		UserID:       testdata.UserID,
	}, &ira_helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report2RulesEnabledRulesExpectedResponse,
		BodyChecker: assertReportResponsesEqual,
	})

	// user 2 re-enables rule
	ira_helpers.AssertAPIRequest(t, mockStorage, &config, &ira_helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.EnableRuleForClusterEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID},
		UserID:       testdata.User2ID,
	}, &ira_helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"status": "ok"}`,
	})

	// user 2 sees no rules disabled
	ira_helpers.AssertAPIRequest(t, mockStorage, &config, &ira_helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
		UserID:       testdata.User2ID,
	}, &ira_helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report2RulesEnabledRulesExpectedResponse,
		BodyChecker: assertReportResponsesEqual,
	})

	// user 1 disables rule1
	ira_helpers.AssertAPIRequest(t, mockStorage, &config, &ira_helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.DisableRuleForClusterEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID},
		UserID:       testdata.UserID,
	}, &ira_helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"status": "ok"}`,
	})

	// user 1 disables rule2
	ira_helpers.AssertAPIRequest(t, mockStorage, &config, &ira_helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.DisableRuleForClusterEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule2ID},
		UserID:       testdata.UserID,
	}, &ira_helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"status": "ok"}`,
	})

	// user 1 is affected
	ira_helpers.AssertAPIRequest(t, mockStorage, &config, &ira_helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
		UserID:       testdata.UserID,
	}, &ira_helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report2RulesDisabledExpectedResponse,
		BodyChecker: assertReportResponsesEqual,
	})

	// user 2 is not affected
	ira_helpers.AssertAPIRequest(t, mockStorage, &config, &ira_helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
		UserID:       testdata.User2ID,
	}, &ira_helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report2RulesEnabledRulesExpectedResponse,
		BodyChecker: assertReportResponsesEqual,
	})

}
