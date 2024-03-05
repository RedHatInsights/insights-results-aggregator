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
	"testing"
	"time"

	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
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
	namespaceAID         = "NAMESPACE-UID-A"
	namespaceBID         = "NAMESPACE-UID-B"
)

// TestProcessSingleDVONamespace_MustProcessEscapedString tests the behavior of ProcessSingleDVONamespace with the current
// escaped JSON string, the whole string is also wrapped in quotation marks
func TestProcessSingleDVONamespace_MustProcessEscapedString(t *testing.T) {
	testServer := server.New(helpers.DefaultServerConfig, nil, nil)

	now := types.Timestamp(time.Now().UTC().Format(time.RFC3339))

	dvoReport := types.DVOReport{
		OrgID:           "1",
		NamespaceID:     namespaceBID,
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
	assert.Equal(t, namespaceBID, processedWorkload.Namespace.UUID)
	assert.Equal(t, 1, processedWorkload.Metadata.Objects)
	assert.Equal(t, 1, processedWorkload.Metadata.Recommendations)

	samples, err := json.Marshal(processedWorkload.Recommendations[0].TemplateData["samples"])
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(string(samples)), 1)
}

// TestProcessSingleDVONamespace_MustProcessCorrectString tests the behavior of the ProcessSingleDVONamespace with a
// correct string (no double escapes, leading, trailing quotes). This test demonstrates that we can fix the encoding
// without affecting the API response at all, as the function simply doesn't strip anything from the strings.
func TestProcessSingleDVONamespace_MustProcessCorrectString(t *testing.T) {
	testServer := server.New(helpers.DefaultServerConfig, nil, nil)

	now := types.Timestamp(time.Now().UTC().Format(time.RFC3339))

	dvoReport := types.DVOReport{
		OrgID:           "1",
		NamespaceID:     namespaceBID,
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
	assert.Equal(t, namespaceBID, processedWorkload.Namespace.UUID)
	assert.Equal(t, 1, processedWorkload.Metadata.Objects)
	assert.Equal(t, 1, processedWorkload.Metadata.Recommendations)

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
	testServer := server.New(helpers.DefaultServerConfig, nil, nil)

	now := types.Timestamp(time.Now().UTC().Format(time.RFC3339))

	dvoReport := types.DVOReport{
		OrgID:           "1",
		NamespaceID:     namespaceAID,
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
			UID:  "193a2099-0000-1111-916a-d570c9aac158",
			Kind: "Pod",
		},
		{
			UID:  "193a2099-1234-5678-916a-d570c9aac158",
			Kind: "DaemonSet",
		},
	}
	assert.ElementsMatch(t, expectedObjects, processedWorkload.Recommendations[0].Objects)
	assert.Equal(t, namespaceAID, processedWorkload.Namespace.UUID)

	// check correct metadata as well
	assert.Equal(t, 2, processedWorkload.Metadata.Objects)
	assert.Equal(t, 1, processedWorkload.Metadata.Recommendations)

	samples, err := json.Marshal(processedWorkload.Recommendations[0].TemplateData["samples"])
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(string(samples)), 1)
}
