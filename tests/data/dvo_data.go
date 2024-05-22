/*
Copyright © 2024 Red Hat, Inc.

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

package testdata

import (
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

var (
	// NamespaceAUID represents namespace A
	NamespaceAUID = "NAMESPACE-UID-A"
	// NamespaceBUID represents namespace A
	NamespaceBUID = "NAMESPACE-UID-B"
	// NamespaceAWorkload workload for namespace A
	NamespaceAWorkload = types.DVOWorkload{
		Namespace:    "namespace-name-A",
		NamespaceUID: NamespaceAUID,
		Kind:         "DaemonSet",
		Name:         "test-name-0099",
		UID:          "UID-0099",
	}
	// NamespaceAWorkload2 another workload for namespace A
	NamespaceAWorkload2 = types.DVOWorkload{
		Namespace:    "namespace-name-A",
		NamespaceUID: NamespaceAUID,
		Kind:         "Pod",
		Name:         "test-name-0001",
		UID:          "UID-0001",
	}
	// NamespaceBWorkload workload for namespace B
	NamespaceBWorkload = types.DVOWorkload{
		Namespace:    "namespace-name-B",
		NamespaceUID: NamespaceBUID,
		Kind:         "NotDaemonSet",
		Name:         "test-name-1199",
		UID:          "UID-1199",
	}
	// ValidDVORecommendation to be inserted into DB
	ValidDVORecommendation = []types.WorkloadRecommendation{
		{
			ResponseID: "an_issue|DVO_AN_ISSUE",
			Component:  "ccx_rules_ocp.external.dvo.an_issue_pod.recommendation",
			Key:        "DVO_AN_ISSUE",
			Links: types.DVOLinks{
				Jira:                 []string{"https://issues.redhat.com/browse/AN_ISSUE"},
				ProductDocumentation: []string{},
			},
			Details: map[string]interface{}{
				"check_name": "",
				"check_url":  "",
				"samples": []interface{}{
					map[string]interface{}{
						"namespace_uid": NamespaceAUID, "kind": "DaemonSet", "uid": "193a2099-1234-5678-916a-d570c9aac158",
					},
				},
			},
			Tags:      []string{},
			Workloads: []types.DVOWorkload{NamespaceAWorkload},
		},
	}
	// ValidReport is a full report inserted into `report` column
	ValidReport = `{"system":{"metadata":{},"hostname":null},"fingerprints":[],"version":1,"analysis_metadata":{},"workload_recommendations":[{"response_id":"an_issue|DVO_AN_ISSUE","component":"ccx_rules_ocp.external.dvo.an_issue_pod.recommendation","key":"DVO_AN_ISSUE","details":{"check_name":"","check_url":"","samples":[{"namespace_uid":"NAMESPACE-UID-A","kind":"DaemonSet","uid":"193a2099-1234-5678-916a-d570c9aac158"}]},"tags":[],"links":{"jira":["https://issues.redhat.com/browse/AN_ISSUE"],"product_documentation":[]},"workloads":[{"namespace":"namespace-name-A","namespace_uid":"NAMESPACE-UID-A","kind":"DaemonSet","name":"test-name-0099","uid":"UID-0099"}]}]}`
	// ValidReport2Rules2Namespaces is a full report inserted into `report` column with 2 rules and 2 namespaces
	ValidReport2Rules2Namespaces = `{"system":{"metadata":{},"hostname":null},"fingerprints":[],"version":1,"analysis_metadata":{},"workload_recommendations":[{"response_id":"unset_requirements|DVO_UNSET_REQUIREMENTS","component":"ccx_rules_ocp.external.dvo.unset_requirements.recommendation","key":"DVO_UNSET_REQUIREMENTS","details":{"check_name":"","check_url":"","samples":[{"namespace_uid":"NAMESPACE-UID-A","kind":"DaemonSet","uid":"193a2099-1234-5678-916a-d570c9aac158"}]},"tags":[],"links":{"jira":["https://issues.redhat.com/browse/AN_ISSUE"],"product_documentation":[]},"workloads":[{"namespace":"namespace-name-A","namespace_uid":"NAMESPACE-UID-A","kind":"DaemonSet","name":"test-name-0099","uid":"193a2099-1234-5678-916a-d570c9aac158"},{"namespace":"namespace-name-B","namespace_uid":"NAMESPACE-UID-B","kind":"DaemonSet","name":"test-name-1234","uid":"12345678-1234-5678-916a-d570c9aac158"}]},{"response_id":"excluded_pod|EXCLUDED_POD","component":"ccx_rules_ocp.external.dvo.excluded_pod.recommendation","key":"EXCLUDED_POD","details":{"check_name":"","check_url":"","samples":[{"namespace_uid":"NAMESPACE-UID-B","kind":"DaemonSet","uid":"12345678-1234-5678-916a-d570c9aac158"}]},"tags":[],"links":{"jira":["https://issues.redhat.com/browse/AN_ISSUE"],"product_documentation":[]},"workloads":[{"namespace":"namespace-name-B","namespace_uid":"NAMESPACE-UID-B","kind":"DaemonSet","name":"test-name-1234","uid":"12345678-1234-5678-916a-d570c9aac158"}]}]}`
	// TwoNamespacesRecommendation to be inserted into DB with 2 namespaces
	TwoNamespacesRecommendation = []types.WorkloadRecommendation{
		{
			ResponseID: "an_issue|DVO_AN_ISSUE",
			Component:  "ccx_rules_ocp.external.dvo.an_issue_pod.recommendation",
			Key:        "DVO_AN_ISSUE",
			Links: types.DVOLinks{
				Jira:                 []string{"https://issues.redhat.com/browse/AN_ISSUE"},
				ProductDocumentation: []string{},
			},
			Details: map[string]interface{}{
				"check_name": "",
				"check_url":  "",
				"samples": []interface{}{
					map[string]interface{}{
						"namespace_uid": NamespaceAUID, "kind": "DaemonSet", "uid": "193a2099-1234-5678-916a-d570c9aac158",
					},
				},
			},
			Tags:      []string{},
			Workloads: []types.DVOWorkload{NamespaceAWorkload, NamespaceBWorkload},
		},
	}
	// Recommendation1TwoNamespaces with 2 namespaces 1 rule
	Recommendation1TwoNamespaces = types.WorkloadRecommendation{
		ResponseID: "an_issue|DVO_AN_ISSUE",
		Component:  "ccx_rules_ocp.external.dvo.an_issue_pod.recommendation",
		Key:        "DVO_AN_ISSUE",
		Links: types.DVOLinks{
			Jira:                 []string{"https://issues.redhat.com/browse/AN_ISSUE"},
			ProductDocumentation: []string{},
		},
		Details: map[string]interface{}{
			"check_name": "",
			"check_url":  "",
		},
		Tags:      []string{},
		Workloads: []types.DVOWorkload{NamespaceAWorkload, NamespaceBWorkload},
	}
	// Recommendation2OneNamespace with 1 namespace 2 rules
	Recommendation2OneNamespace = types.WorkloadRecommendation{
		ResponseID: "unset_requirements|DVO_UNSET_REQUIREMENTS",
		Component:  "ccx_rules_ocp.external.dvo.unset_requirements.recommendation",
		Key:        "DVO_UNSET_REQUIREMENTS",
		Links: types.DVOLinks{
			Jira:                 []string{"https://issues.redhat.com/browse/AN_ISSUE"},
			ProductDocumentation: []string{},
		},
		Details: map[string]interface{}{
			"check_name": "",
			"check_url":  "",
		},
		Tags:      []string{},
		Workloads: []types.DVOWorkload{NamespaceAWorkload, NamespaceAWorkload2},
	}
	// Recommendation3OneNamespace with 1 namespace 3 rules
	Recommendation3OneNamespace = types.WorkloadRecommendation{
		ResponseID: "bad_requirements|BAD_REQUIREMENTS",
		Component:  "ccx_rules_ocp.external.dvo.bad_requirements.recommendation",
		Key:        "BAD_REQUIREMENTS",
		Links: types.DVOLinks{
			Jira:                 []string{"https://issues.redhat.com/browse/AN_ISSUE"},
			ProductDocumentation: []string{},
		},
		Details: map[string]interface{}{
			"check_name": "",
			"check_url":  "",
		},
		Tags:      []string{},
		Workloads: []types.DVOWorkload{NamespaceAWorkload, NamespaceAWorkload2},
	}
)
