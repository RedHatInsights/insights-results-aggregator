/*
Copyright © 2020, 2021, 2022, 2023, 2024  Red Hat, Inc.

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

package server_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"

	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/RedHatInsights/insights-results-aggregator/server"
	ira_data "github.com/RedHatInsights/insights-results-aggregator/tests/data"
	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/stretchr/testify/assert"
)

const (
	/*
		test reports have following structure:
		{
			...
			"workload_recommendations": [
				{
					"response_id": "unset_requirements|DVO_UNSET_REQUIREMENTS",
					...
					"workloads": [
						{
						"namespace": "namespace-name-A",
						...
						"uid": "193a2099-1234-5678-916a-d570c9aac158"
						},
						{
						"namespace": "namespace-name-A",
						...
						"uid": "193a2099-0000-1111-916a-d570c9aac158"
						}
					]
				},
				{
					"response_id": "excluded_pod|EXCLUDED_POD",
					...
					"workloads": [
						{
						"namespace": "namespace-name-B",
						...
						"uid": "12345678-1234-5678-916a-d570c9aac158"
						}
					]
				}
			]
		}

	*/
	currentDvoReportFromDB = `"{\"system\":{\"metadata\":{},\"hostname\":null},\"fingerprints\":[],\"version\":1,\"analysis_metadata\":{},\"workload_recommendations\":[{\"response_id\":\"unset_requirements|DVO_UNSET_REQUIREMENTS\",\"component\":\"ccx_rules_ocp.external.dvo.unset_requirements.recommendation\",\"key\":\"DVO_UNSET_REQUIREMENTS\",\"details\":{\"check_name\":\"\",\"check_url\":\"\",\"samples\":[{\"namespace_uid\":\"NAMESPACE-UID-A\",\"kind\":\"DaemonSet\",\"uid\":\"193a2099-1234-5678-916a-d570c9aac158\"}]},\"tags\":[],\"links\":{\"jira\":[\"https://issues.redhat.com/browse/AN_ISSUE\"],\"product_documentation\":[]},\"workloads\":[{\"namespace\":\"namespace-name-A\",\"namespace_uid\":\"NAMESPACE-UID-A\",\"kind\":\"DaemonSet\",\"name\":\"test-name-0099\",\"uid\":\"193a2099-1234-5678-916a-d570c9aac158\"},{\"namespace\":\"namespace-name-A\",\"namespace_uid\":\"NAMESPACE-UID-A\",\"kind\":\"Pod\",\"name\":\"test-name-0001\",\"uid\":\"193a2099-0000-1111-916a-d570c9aac158\"}]},{\"response_id\":\"excluded_pod|EXCLUDED_POD\",\"component\":\"ccx_rules_ocp.external.dvo.excluded_pod.recommendation\",\"key\":\"EXCLUDED_POD\",\"details\":{\"check_name\":\"\",\"check_url\":\"\",\"samples\":[{\"namespace_uid\":\"NAMESPACE-UID-B\",\"kind\":\"DaemonSet\",\"uid\":\"12345678-1234-5678-916a-d570c9aac158\"}]},\"tags\":[],\"links\":{\"jira\":[\"https://issues.redhat.com/browse/AN_ISSUE\"],\"product_documentation\":[]},\"workloads\":[{\"namespace\":\"namespace-name-B\",\"namespace_uid\":\"NAMESPACE-UID-B\",\"kind\":\"DaemonSet\",\"name\":\"test-name-1234\",\"uid\":\"12345678-1234-5678-916a-d570c9aac158\"}]}]}"`
	// fixedDvoReportFromDB is what the string inside the report column should look like (after we fix the encoding issues)
	fixedDvoReportFromDB = `{"system":{"metadata":{},"hostname":null},"fingerprints":[],"version":1,"analysis_metadata":{},"workload_recommendations":[{"response_id":"unset_requirements|DVO_UNSET_REQUIREMENTS","component":"ccx_rules_ocp.external.dvo.unset_requirements.recommendation","key":"DVO_UNSET_REQUIREMENTS","details":{"check_name":"","check_url":"","samples":[{"namespace_uid":"NAMESPACE-UID-A","kind":"DaemonSet","uid":"193a2099-1234-5678-916a-d570c9aac158"}]},"tags":[],"links":{"jira":["https://issues.redhat.com/browse/AN_ISSUE"],"product_documentation":[]},"workloads":[{"namespace":"namespace-name-A","namespace_uid":"NAMESPACE-UID-A","kind":"DaemonSet","name":"test-name-0099","uid":"193a2099-1234-5678-916a-d570c9aac158"},{"namespace":"namespace-name-A","namespace_uid":"NAMESPACE-UID-A","kind":"Pod","name":"test-name-0001","uid":"193a2099-0000-1111-916a-d570c9aac158"}]},{"response_id":"excluded_pod|EXCLUDED_POD","component":"ccx_rules_ocp.external.dvo.excluded_pod.recommendation","key":"EXCLUDED_POD","details":{"check_name":"","check_url":"","samples":[{"namespace_uid":"NAMESPACE-UID-B","kind":"DaemonSet","uid":"12345678-1234-5678-916a-d570c9aac158"}]},"tags":[],"links":{"jira":["https://issues.redhat.com/browse/AN_ISSUE"],"product_documentation":[]},"workloads":[{"namespace":"namespace-name-B","namespace_uid":"NAMESPACE-UID-B","kind":"DaemonSet","name":"test-name-1234","uid":"12345678-1234-5678-916a-d570c9aac158"}]}]}`
	objectBUID           = `12345678-1234-5678-916a-d570c9aac158`
)

var (
	now = time.Now().UTC()
)

func workloadInResponseChecker(t testing.TB, expected, got []byte) {
	type Response struct {
		Status    string `json:"string"`
		Workloads []types.WorkloadsForNamespace
	}

	var expectedResp Response
	var gotResp Response

	if err := json.Unmarshal(expected, &expectedResp); err != nil {
		err = fmt.Errorf(`"expected" is not JSON. value = "%v", err = "%v"`, expected, err)
		helpers.FailOnError(t, err)
	}

	if err := json.Unmarshal(got, &gotResp); err != nil {
		err = fmt.Errorf(`"got" is not JSON. value = "%v", err = "%v"`, string(got), err)
		helpers.FailOnError(t, err)
	}

	assert.ElementsMatch(t, expectedResp.Workloads, gotResp.Workloads)
}

// TestProcessSingleDVONamespace_MustProcessEscapedString tests the behavior of ProcessSingleDVONamespace with the current
// escaped JSON string, the whole string is also wrapped in quotation marks
func TestProcessSingleDVONamespace_MustProcessEscapedString(t *testing.T) {
	testServer := server.New(ira_helpers.DefaultServerConfig, nil, nil)

	now := types.Timestamp(time.Now().UTC().Format(time.RFC3339))

	dvoReport := types.DVOReport{
		OrgID:           "1",
		NamespaceID:     ira_data.NamespaceBUID,
		NamespaceName:   "namespace-name-B",
		ClusterID:       string(testdata.ClusterName),
		Recommendations: 1,
		Report:          currentDvoReportFromDB,
		Objects:         1,
		ReportedAt:      now,
		LastCheckedAt:   now,
	}

	processedWorkload := testServer.ProcessSingleDVONamespace(dvoReport)

	assert.Equal(t, 1, len(processedWorkload.Recommendations))
	assert.Equal(t, 1, len(processedWorkload.Recommendations[0].Objects))
	assert.Equal(t, objectBUID, processedWorkload.Recommendations[0].Objects[0].UID)
	assert.Equal(t, ira_data.NamespaceBUID, processedWorkload.Namespace.UUID)
	assert.Equal(t, uint(1), processedWorkload.Metadata.Objects)
	assert.Equal(t, uint(1), processedWorkload.Metadata.Recommendations)

	samples, err := json.Marshal(processedWorkload.Recommendations[0].TemplateData["samples"])
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(string(samples)), 1)
}

// TestProcessSingleDVONamespace_MustProcessCorrectString tests the behavior of the ProcessSingleDVONamespace with a
// correct string (no double escapes, leading, trailing quotes). This test demonstrates that we can fix the encoding
// without affecting the API response at all, as the function simply doesn't strip anything from the strings.
func TestProcessSingleDVONamespace_MustProcessCorrectString(t *testing.T) {
	testServer := server.New(ira_helpers.DefaultServerConfig, nil, nil)

	now := types.Timestamp(time.Now().UTC().Format(time.RFC3339))

	dvoReport := types.DVOReport{
		OrgID:           "1",
		NamespaceID:     ira_data.NamespaceBUID,
		NamespaceName:   "namespace-name-B",
		ClusterID:       "193a2099-1234-5678-916a-d570c9aac158",
		Recommendations: 1,
		Report:          fixedDvoReportFromDB,
		Objects:         1,
		ReportedAt:      now,
		LastCheckedAt:   now,
	}

	processedWorkload := testServer.ProcessSingleDVONamespace(dvoReport)

	assert.Equal(t, 1, len(processedWorkload.Recommendations))
	assert.Equal(t, 1, len(processedWorkload.Recommendations[0].Objects))
	assert.Equal(t, objectBUID, processedWorkload.Recommendations[0].Objects[0].UID)
	assert.Equal(t, ira_data.NamespaceBUID, processedWorkload.Namespace.UUID)
	assert.Equal(t, uint(1), processedWorkload.Metadata.Objects)
	assert.Equal(t, uint(1), processedWorkload.Metadata.Recommendations)

	samples, err := json.Marshal(processedWorkload.Recommendations[0].TemplateData["samples"])
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(string(samples)), 1)
}

// TestProcessSingleDVONamespace_MustFilterZeroObjects_CCXDEV_12589_Reproducer
// tests the behavior of the ProcessSingleDVONamespace
// to filter out recommendations that have 0 objects/workloads for that particular namespace.
// Since we process the report every time, we need to filter out objects+namespaces that don't match
// the requested one. And since we can end up with 0 objects for that rule_ID + namespace after the filtering,
// we mustn't show this recommendation in the API, as it has no rule hits in reality.
func TestProcessSingleDVONamespace_MustFilterZeroObjects_CCXDEV_12589_Reproducer(t *testing.T) {
	testServer := server.New(ira_helpers.DefaultServerConfig, nil, nil)

	now := types.Timestamp(time.Now().UTC().Format(time.RFC3339))

	dvoReport := types.DVOReport{
		OrgID:           "1",
		NamespaceID:     ira_data.NamespaceAUID,
		NamespaceName:   "namespace-name-A",
		ClusterID:       string(testdata.ClusterName),
		Recommendations: 1,
		Report:          fixedDvoReportFromDB,
		Objects:         2,
		ReportedAt:      now,
		LastCheckedAt:   now,
	}

	processedWorkload := testServer.ProcessSingleDVONamespace(dvoReport)

	assert.Equal(t, 1, len(processedWorkload.Recommendations)) // <-- this would be 2 without CCXDEV-12589 fix
	assert.Equal(t, 2, len(processedWorkload.Recommendations[0].Objects))
	expectedObjects := []server.DVOObject{
		{
			UID:         "193a2099-0000-1111-916a-d570c9aac158",
			Kind:        "Pod",
			DisplayName: "test-name-0001",
		},
		{
			UID:         "193a2099-1234-5678-916a-d570c9aac158",
			Kind:        "DaemonSet",
			DisplayName: "test-name-0099",
		},
	}
	assert.ElementsMatch(t, expectedObjects, processedWorkload.Recommendations[0].Objects)
	assert.Equal(t, ira_data.NamespaceAUID, processedWorkload.Namespace.UUID)

	// check correct metadata as well
	assert.Equal(t, uint(2), processedWorkload.Metadata.Objects)
	assert.Equal(t, uint(1), processedWorkload.Metadata.Recommendations)

	samples, err := json.Marshal(processedWorkload.Recommendations[0].TemplateData["samples"])
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(string(samples)), 1)
}

func TestGetWorkloadsOK(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		types.ClusterReport(ira_data.ValidReport),
		ira_data.ValidDVORecommendation,
		now,
		now,
		now,
		testdata.RequestID1,
	)
	helpers.FailOnError(t, err)

	workload := types.WorkloadsForNamespace{
		Cluster: types.Cluster{
			UUID: string(testdata.ClusterName),
		},
		Namespace: types.Namespace{
			UUID: ira_data.NamespaceAUID,
			Name: "namespace-name-A",
		},
		Metadata: types.DVOMetadata{
			Recommendations: 1,
			Objects:         1,
			ReportedAt:      now.UTC().Format(time.RFC3339),
			LastCheckedAt:   now.UTC().Format(time.RFC3339),
		},
		RecommendationsHitCount: types.RuleHitsCount{
			"ccx_rules_ocp.external.dvo.an_issue_pod|DVO_AN_ISSUE": 1,
		},
	}

	ira_helpers.AssertAPIRequestDVO(t, mockStorage, nil, &helpers.APIRequest{
		Method:       http.MethodPost,
		Endpoint:     server.DVOWorkloadRecommendations,
		EndpointArgs: []interface{}{testdata.OrgID},
		Body:         fmt.Sprintf(`["%v"]`, testdata.ClusterName),
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"status":"ok","workloads":` + helpers.ToJSONString([]types.WorkloadsForNamespace{workload}) + `}`,
	})
}

func TestGetWorkloads_NoData(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)
	defer closer()

	ira_helpers.AssertAPIRequestDVO(t, mockStorage, nil, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.DVOWorkloadRecommendations,
		EndpointArgs: []interface{}{testdata.OrgID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
	})
}

func TestGetWorkloadsOK_TwoNamespacesGet(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		types.ClusterReport(ira_data.ValidReport),
		ira_data.TwoNamespacesRecommendation,
		now,
		now,
		now,
		testdata.RequestID1,
	)
	helpers.FailOnError(t, err)

	workloads := []types.WorkloadsForNamespace{
		{
			Cluster: types.Cluster{
				UUID: string(testdata.ClusterName),
			},
			Namespace: types.Namespace{
				UUID: ira_data.NamespaceAUID,
				Name: "namespace-name-A",
			},
			Metadata: types.DVOMetadata{
				Recommendations: 1,
				Objects:         1,
				ReportedAt:      now.UTC().Format(time.RFC3339),
				LastCheckedAt:   now.UTC().Format(time.RFC3339),
			},
			RecommendationsHitCount: types.RuleHitsCount{
				"ccx_rules_ocp.external.dvo.an_issue_pod|DVO_AN_ISSUE": 1,
			},
		},
		{
			Cluster: types.Cluster{
				UUID: string(testdata.ClusterName),
			},
			Namespace: types.Namespace{
				UUID: ira_data.NamespaceBUID,
				Name: "namespace-name-B",
			},
			Metadata: types.DVOMetadata{
				Recommendations: 1,
				Objects:         1,
				ReportedAt:      now.UTC().Format(time.RFC3339),
				LastCheckedAt:   now.UTC().Format(time.RFC3339),
			},
			RecommendationsHitCount: types.RuleHitsCount{
				"ccx_rules_ocp.external.dvo.an_issue_pod|DVO_AN_ISSUE": 1,
			},
		},
	}

	// testing that GET functionality is kept (no filtering)
	ira_helpers.AssertAPIRequestDVO(t, mockStorage, nil, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.DVOWorkloadRecommendations,
		EndpointArgs: []interface{}{testdata.OrgID},
	}, &helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        `{"status":"ok","workloads":` + helpers.ToJSONString(workloads) + `}`,
		BodyChecker: workloadInResponseChecker,
	})
}

func TestGetWorkloadsOK_TwoNamespaces(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		types.ClusterReport(ira_data.ValidReport),
		ira_data.TwoNamespacesRecommendation,
		now,
		now,
		now,
		testdata.RequestID1,
	)
	helpers.FailOnError(t, err)

	workloads := []types.WorkloadsForNamespace{
		{
			Cluster: types.Cluster{
				UUID: string(testdata.ClusterName),
			},
			Namespace: types.Namespace{
				UUID: ira_data.NamespaceAUID,
				Name: "namespace-name-A",
			},
			Metadata: types.DVOMetadata{
				Recommendations: 1,
				Objects:         1,
				ReportedAt:      now.UTC().Format(time.RFC3339),
				LastCheckedAt:   now.UTC().Format(time.RFC3339),
			},
			RecommendationsHitCount: types.RuleHitsCount{
				"ccx_rules_ocp.external.dvo.an_issue_pod|DVO_AN_ISSUE": 1,
			},
		},
		{
			Cluster: types.Cluster{
				UUID: string(testdata.ClusterName),
			},
			Namespace: types.Namespace{
				UUID: ira_data.NamespaceBUID,
				Name: "namespace-name-B",
			},
			Metadata: types.DVOMetadata{
				Recommendations: 1,
				Objects:         1,
				ReportedAt:      now.UTC().Format(time.RFC3339),
				LastCheckedAt:   now.UTC().Format(time.RFC3339),
			},
			RecommendationsHitCount: types.RuleHitsCount{
				"ccx_rules_ocp.external.dvo.an_issue_pod|DVO_AN_ISSUE": 1,
			},
		},
	}

	ira_helpers.AssertAPIRequestDVO(t, mockStorage, nil, &helpers.APIRequest{
		Method:       http.MethodPost,
		Endpoint:     server.DVOWorkloadRecommendations,
		EndpointArgs: []interface{}{testdata.OrgID},
		Body:         fmt.Sprintf(`["%v"]`, testdata.ClusterName),
	}, &helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        `{"status":"ok","workloads":` + helpers.ToJSONString(workloads) + `}`,
		BodyChecker: workloadInResponseChecker,
	})
}

// TestGetWorkloadsOK_ActiveClusterFilter tests scenario where cluster list filtering is enabled by making
// a POST request, but no clusters are provided. Therefore, all clusters/results must be filtered out.
func TestGetWorkloadsOK_ActiveClusterFilter(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		types.ClusterReport(ira_data.ValidReport),
		ira_data.TwoNamespacesRecommendation,
		now,
		now,
		now,
		testdata.RequestID1,
	)
	helpers.FailOnError(t, err)

	ira_helpers.AssertAPIRequestDVO(t, mockStorage, nil, &helpers.APIRequest{
		Method:       http.MethodPost,
		Endpoint:     server.DVOWorkloadRecommendations,
		EndpointArgs: []interface{}{testdata.OrgID},
		Body:         `[]`, // <-- no cluster list
	}, &helpers.APIResponse{
		StatusCode:  http.StatusOK,
		Body:        `{"status":"ok","workloads":[]}`, // <-- cluster filtered
		BodyChecker: workloadInResponseChecker,
	})
}
