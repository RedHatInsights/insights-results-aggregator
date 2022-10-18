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
	"sort"
	"testing"
	"time"

	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	ctypes "github.com/RedHatInsights/insights-results-types"
	"github.com/stretchr/testify/assert"

	utils "github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

func TestReadReportForClusterNonIntOrgID(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, nil, &helpers.APIRequest{
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
	helpers.AssertAPIRequest(t, nil, nil, &helpers.APIRequest{
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
	helpers.AssertAPIRequest(t, nil, nil, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.BadClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body:       `{"status": "Error during parsing param 'cluster' with value 'aaaa'. Error: 'invalid UUID length: 4'"}`,
	})
}

func TestReadNonExistingReport(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, nil, &helpers.APIRequest{
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
		testdata.OrgID,
		testdata.ClusterName,
		testdata.Report0Rules,
		testdata.ReportEmptyRulesParsed,
		testdata.LastCheckedAt,
		testdata.LastCheckedAt,
		time.Now(),
		testdata.KafkaOffset,
	)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, nil, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body: `{
			"status":"ok",
			"report": {
				"meta": {
					"cluster_name":"",
					"count": -1,
					"last_checked_at": "` + testdata.LastCheckedAt.Format(time.RFC3339) + `",
					"gathered_at": "` + testdata.LastCheckedAt.Format(time.RFC3339) + `"
				},
				"reports":[]
			}
		}`,
	})
}

func TestReadReportDBError(t *testing.T) {
	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	closer()

	helpers.AssertAPIRequest(t, mockStorage, nil, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       `{"status":"Internal Server Error"}`,
	})
}

func TestReadReport(t *testing.T) {

	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.Report3Rules,
		testdata.Report3RulesParsed,
		testdata.LastCheckedAt,
		testdata.LastCheckedAt,
		time.Now(),
		testdata.KafkaOffset,
	)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, nil, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       testdata.Report3RulesExpectedResponse,
		BodyChecker: func(t testing.TB, expected, got []byte) {
			helpers.AssertReportResponsesEqualCustomElementsChecker(
				t, expected, got,
				func(
					t testing.TB,
					expectedRules []ctypes.RuleOnReport,
					gotRules []ctypes.RuleOnReport,
				) {
					assert.Equal(t, len(expectedRules), len(gotRules))
					expectedRules = utils.SortReports(expectedRules)
					gotRules = utils.SortReports(gotRules)
					for i, v := range expectedRules {
						utils.CompareReportResponses(t, v, gotRules[i], time.Time{})
					}
				},
			)
		},
	})
}

func TestReadRuleReport(t *testing.T) {
	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.Report3Rules,
		testdata.Report3RulesParsed,
		testdata.LastCheckedAt,
		testdata.LastCheckedAt,
		time.Now(),
		testdata.KafkaOffset,
	)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, nil, &helpers.APIRequest{
		Method:   http.MethodGet,
		Endpoint: server.RuleEndpoint,
		EndpointArgs: []interface{}{
			testdata.OrgID,
			testdata.ClusterName,
			testdata.UserID,
			fmt.Sprintf("%v|%v", testdata.Rule1ID, testdata.ErrorKey1),
		},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body: fmt.Sprintf(`{
			"report": %v,
			"status": "ok"
		}`, helpers.ToJSONString(testdata.RuleOnReport1)),
		BodyChecker: helpers.AssertRuleResponsesEqual,
	})
}

// TestReadReportDisableRule reads a report, disables the first rule, fetches again,
// expecting the rule to be last and disabled, re-enables it and expects regular
// response with Rule1 first again
func TestReadReportDisableRule(t *testing.T) {

	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	defer closer()

	now := time.Now()
	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.Report2Rules,
		testdata.Report2RulesParsed,
		testdata.LastCheckedAt,
		testdata.LastCheckedAt,
		now,
		testdata.KafkaOffset,
	)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, nil, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       testdata.Report2RulesEnabledRulesExpectedResponse,
		BodyChecker: func(t testing.TB, expected, got []byte) {
			helpers.AssertReportResponsesEqualCustomElementsChecker(
				t, expected, got,
				func(
					t testing.TB,
					expectedRules []ctypes.RuleOnReport,
					gotRules []ctypes.RuleOnReport,
				) {
					assert.Equal(t, len(expectedRules), len(gotRules))
					for i, v := range expectedRules {
						utils.CompareReportResponses(t, v, gotRules[i], now)
					}
				},
			)
		},
	})

	helpers.AssertAPIRequest(t, mockStorage, nil, &helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.DisableRuleForClusterEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"status": "ok"}`,
	})

	helpers.AssertAPIRequest(t, mockStorage, nil, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       testdata.Report2RulesDisabledRule1ExpectedResponse,
		BodyChecker: func(t testing.TB, expected, got []byte) {
			helpers.AssertReportResponsesEqualCustomElementsChecker(
				t, expected, got,
				func(
					t testing.TB,
					expectedRules []ctypes.RuleOnReport,
					gotRules []ctypes.RuleOnReport,
				) {
					assert.Equal(t, len(expectedRules), len(gotRules))
					expectedRules = utils.SortReports(expectedRules)
					gotRules = utils.SortReports(gotRules)
					for i, v := range expectedRules {
						utils.CompareReportResponses(t, v, gotRules[i], time.Time{})
					}
				},
			)
		},
	})

	helpers.AssertAPIRequest(t, mockStorage, nil, &helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.EnableRuleForClusterEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"status": "ok"}`,
	})

	helpers.AssertAPIRequest(t, mockStorage, nil, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       testdata.Report2RulesEnabledRulesExpectedResponse,
		BodyChecker: func(t testing.TB, expected, got []byte) {
			helpers.AssertReportResponsesEqualCustomElementsChecker(
				t, expected, got,
				func(
					t testing.TB,
					expectedRules []ctypes.RuleOnReport,
					gotRules []ctypes.RuleOnReport,
				) {
					assert.Equal(t, len(expectedRules), len(gotRules))
					for i, v := range expectedRules {
						utils.CompareReportResponses(t, v, gotRules[i], now)
					}
				},
			)
		},
	})
}

func TestReadReport_RuleDisableFeedback(t *testing.T) {

	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	defer closer()

	now := time.Now()
	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.Report2Rules,
		testdata.Report2RulesParsed,
		testdata.LastCheckedAt,
		testdata.LastCheckedAt,
		now,
		testdata.KafkaOffset,
	)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, nil, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       testdata.Report2RulesEnabledRulesExpectedResponse,
		BodyChecker: func(t testing.TB, expected, got []byte) {
			helpers.AssertReportResponsesEqualCustomElementsChecker(
				t, expected, got,
				func(
					t testing.TB,
					expectedRules []ctypes.RuleOnReport,
					gotRules []ctypes.RuleOnReport,
				) {
					assert.Equal(t, len(expectedRules), len(gotRules))
					for i, v := range expectedRules {
						utils.CompareReportResponses(t, v, gotRules[i], now)
					}
				},
			)
		},
	})

	helpers.AssertAPIRequest(t, mockStorage, nil, &helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.DisableRuleForClusterEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"status": "ok"}`,
	})

	helpers.AssertAPIRequest(t, mockStorage, nil, &helpers.APIRequest{
		Method:       http.MethodPost,
		Endpoint:     server.DisableRuleFeedbackEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.ErrorKey1, testdata.OrgID, testdata.UserID},
		Body:         `{"message": "test"}`,
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"status": "ok", "message": "test"}`,
	})

	helpers.AssertAPIRequest(t, mockStorage, nil, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       testdata.Report2RulesDisabledRule1WithFeedbackExpectedResponse,
		BodyChecker: func(t testing.TB, expected, got []byte) {
			helpers.AssertReportResponsesEqualCustomElementsChecker(
				t, expected, got,
				func(
					t testing.TB,
					expectedRules []ctypes.RuleOnReport,
					gotRules []ctypes.RuleOnReport,
				) {
					assert.Equal(t, len(expectedRules), len(gotRules))

					sort.Slice(expectedRules, func(i, j int) bool {
						return expectedRules[i].Module > expectedRules[j].Module
					})
					sort.Slice(gotRules, func(i, j int) bool {
						return gotRules[i].Module > gotRules[j].Module
					})

					for i := 0; i < len(expectedRules); i++ {
						expectedRule := &expectedRules[i]
						gotRule := &gotRules[i]
						assert.Equal(t, expectedRule.Module, gotRule.Module)
						assert.Equal(t, expectedRule.Disabled, gotRule.Disabled)
						assert.Equal(t, expectedRule.DisableFeedback, gotRule.DisableFeedback)
						assert.Equal(t, expectedRule.TemplateData, gotRule.TemplateData)
						assert.Equal(t, expectedRule.ErrorKey, gotRule.ErrorKey)
						assert.Equal(t, expectedRule.UserVote, gotRule.UserVote)
					}
				},
			)
		},
	})
}
