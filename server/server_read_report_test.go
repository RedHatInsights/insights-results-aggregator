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

	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"

	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

func TestReadReportForClusterNonIntOrgID(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{"non-int", testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body: `{
			"status": "Error during parsing param 'org_id' with value 'non-int'. Error: 'unsigned integer expected'"
		}`,
	})
}

func TestReadReportForClusterNegativeOrgID(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{-1, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body: `{
			"status":"Error during parsing param 'org_id' with value '-1'. Error: 'unsigned integer expected'"
		}`,
	})
}

func TestReadReportForClusterBadClusterName(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.BadClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body:       `{"status": "Error during parsing param 'cluster' with value 'aaaa'. Error: 'invalid UUID length: 4'"}`,
	})
}

func TestReadNonExistingReport(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusNotFound,
		Body: fmt.Sprintf(
			`{"status":"Item with ID %v/%v was not found in the storage"}`, testdata.OrgID, testdata.ClusterName,
		),
	})
}

func TestHttpServer_readReportForCluster_NoRules(t *testing.T) {
	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report0Rules, testdata.LastCheckedAt, testdata.KafkaOffset,
	)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
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
	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	closer()

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       `{"status":"Internal Server Error"}`,
	})
}

func TestHttpServer_readReportForCluster_getContentForRule_BadReport(t *testing.T) {
	const badReport = "not-json"

	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, badReport, testdata.LastCheckedAt, testdata.KafkaOffset,
	)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body:       `{ "status": "invalid character 'o' in literal null (expecting 'u')" }`,
	})
}

func TestReadReport(t *testing.T) {
	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.Report3Rules,
		testdata.LastCheckedAt,
		testdata.KafkaOffset,
	)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report3RulesExpectedResponse,
		BodyChecker: helpers.AssertReportResponsesEqual,
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
		testdata.KafkaOffset,
	)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report2RulesEnabledRulesExpectedResponse,
		BodyChecker: helpers.AssertReportResponsesEqual,
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
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report2RulesDisabledRule1ExpectedResponse,
		BodyChecker: helpers.AssertReportResponsesEqual,
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
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report2RulesEnabledRulesExpectedResponse,
		BodyChecker: helpers.AssertReportResponsesEqual,
	})
}

// TestReadReportDisableRuleMultipleUsers is a reproducer for bug issues 811 and 814
func TestReadReportDisableRuleMultipleUsers(t *testing.T) {
	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.Report2Rules,
		testdata.LastCheckedAt,
		testdata.KafkaOffset,
	)
	helpers.FailOnError(t, err)

	// user 1 check no disabled rules in response
	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report2RulesEnabledRulesExpectedResponse,
		BodyChecker: helpers.AssertReportResponsesEqual,
	})

	// user 2 disables rule1
	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.DisableRuleForClusterEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID},
		UserID:       testdata.User2ID,
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"status": "ok"}`,
	})

	// user 2 is affected
	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.User2ID},
	}, &helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report2RulesDisabledRule1ExpectedResponse,
		BodyChecker: helpers.AssertReportResponsesEqual,
	})

	// user 1 is not affected
	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report2RulesEnabledRulesExpectedResponse,
		BodyChecker: helpers.AssertReportResponsesEqual,
	})

	// user 2 re-enables rule
	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.EnableRuleForClusterEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID},
		UserID:       testdata.User2ID,
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"status": "ok"}`,
	})

	// user 2 sees no rules disabled
	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.User2ID},
	}, &helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report2RulesEnabledRulesExpectedResponse,
		BodyChecker: helpers.AssertReportResponsesEqual,
	})

	// user 1 disables rule1
	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.DisableRuleForClusterEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID},
		UserID:       testdata.UserID,
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"status": "ok"}`,
	})

	// user 1 disables rule2
	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.DisableRuleForClusterEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule2ID},
		UserID:       testdata.UserID,
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"status": "ok"}`,
	})

	// user 1 is affected
	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report2RulesDisabledExpectedResponse,
		BodyChecker: helpers.AssertReportResponsesEqual,
	})

	// user 2 is not affected
	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.User2ID},
	}, &helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        testdata.Report2RulesEnabledRulesExpectedResponse,
		BodyChecker: helpers.AssertReportResponsesEqual,
	})
}
