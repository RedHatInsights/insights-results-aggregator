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
	"testing"
	"time"

	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/stretchr/testify/assert"
)

const (
	currentDvoReportFromDB = `\"{\\\"system\\\":{\\\"metadata\\\":{},\\\"hostname\\\":null},\\\"fingerprints\\\":[],\\\"version\\\":1,\\\"analysis_metadata\\\":{},\\\"workload_recommendations\\\":[{\\\"response_id\\\":\\\"an_issue|DVO_AN_ISSUE\\\",\\\"component\\\":\\\"ccx_rules_ocp.external.dvo.an_issue_pod.recommendation\\\",\\\"key\\\":\\\"DVO_AN_ISSUE\\\",\\\"details\\\":{\\\"check_name\\\":\\\"\\\",\\\"check_url\\\":\\\"\\\",\\\"samples\\\":[{\\\"namespace_uid\\\":\\\"NAMESPACE-UID-A\\\",\\\"kind\\\":\\\"DaemonSet\\\",\\\"uid\\\":\\\"193a2099-1234-5678-916a-d570c9aac158\\\"}]},\\\"tags\\\":[],\\\"links\\\":{\\\"jira\\\":[\\\"https://issues.redhat.com/browse/AN_ISSUE\\\"],\\\"product_documentation\\\":[]},\\\"workloads\\\":[{\\\"namespace\\\":\\\"namespace-name-A\\\",\\\"namespace_uid\\\":\\\"NAMESPACE-UID-A\\\",\\\"kind\\\":\\\"DaemonSet\\\",\\\"name\\\":\\\"test-name-0099\\\",\\\"uid\\\":\\\"UID-0099\\\"}]}]}\"`
	// fixedDvoReportFromDB is what the string inside the report column should look like (after we fix the encoding issues)
	fixedDvoReportFromDB = `{"system":{"metadata":{},"hostname":null},"fingerprints":[],"version":1,"analysis_metadata":{},"workload_recommendations":[{"response_id":"an_issue|DVO_AN_ISSUE","component":"ccx_rules_ocp.external.dvo.an_issue_pod.recommendation","key":"DVO_AN_ISSUE","details":{"check_name":"","check_url":"","samples":[{"namespace_uid":"NAMESPACE-UID-A","kind":"DaemonSet","uid":"193a2099-1234-5678-916a-d570c9aac158"}]},"tags":[],"links":{"jira":["https://issues.redhat.com/browse/AN_ISSUE"],"product_documentation":[]},"workloads":[{"namespace":"namespace-name-A","namespace_uid":"NAMESPACE-UID-A","kind":"DaemonSet","name":"test-name-0099","uid":"UID-0099"}]}]}`
	objectUID            = `UID-0099`
	namespaceID          = "NAMESPACE-UID-A"
)

// TestProcessSingleDVONamespace_MustProcessEscapedString tests the behavior of ProcessSingleDVONamespace with the current
// double-escaped JSON string, the whole string is also wrapped in quotation marks, which are only single-escaped
func TestProcessSingleDVONamespace_MustProcessEscapedString(t *testing.T) {
	testServer := server.New(helpers.DefaultServerConfig, nil, nil)

	now := types.Timestamp(time.Now().UTC().Format(time.RFC3339))

	dvoReport := types.DVOReport{
		OrgID:           "1",
		NamespaceID:     namespaceID,
		NamespaceName:   "namespace-name-A",
		ClusterID:       "193a2099-1234-5678-916a-d570c9aac158",
		Recommendations: 1,
		Report:          currentDvoReportFromDB,
		Objects:         1,
		ReportedAt:      now,
		LastCheckedAt:   now,
	}

	processedWorkload := testServer.ProcessSingleDVONamespace(dvoReport)

	assert.Equal(t, 1, len(processedWorkload.Recommendations))
	assert.Equal(t, 1, len(processedWorkload.Recommendations[0].Objects))
	assert.Equal(t, objectUID, processedWorkload.Recommendations[0].Objects[0].UID)
	assert.Equal(t, namespaceID, processedWorkload.Namespace.UUID)
	assert.Equal(t, 1, processedWorkload.Metadata.Objects)
	assert.Equal(t, 1, processedWorkload.Metadata.Recommendations)
}

// TestProcessSingleDVONamespace_MustProcessCorrectString tests the behavior of the ProcessSingleDVONamespace with a
// correct string (no double escapes, leading, trailing quotes). This test demonstrates that we can fix the encoding
// without affecting the API response at all, as the function simply doesn't strip anything from the strings.
func TestProcessSingleDVONamespace_MustProcessCorrectString(t *testing.T) {
	testServer := server.New(helpers.DefaultServerConfig, nil, nil)

	now := types.Timestamp(time.Now().UTC().Format(time.RFC3339))

	dvoReport := types.DVOReport{
		OrgID:           "1",
		NamespaceID:     namespaceID,
		NamespaceName:   "namespace-name-A",
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
	assert.Equal(t, objectUID, processedWorkload.Recommendations[0].Objects[0].UID)
	assert.Equal(t, namespaceID, processedWorkload.Namespace.UUID)
	assert.Equal(t, 1, processedWorkload.Metadata.Objects)
	assert.Equal(t, 1, processedWorkload.Metadata.Recommendations)
}
