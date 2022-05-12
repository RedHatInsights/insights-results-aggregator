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

// Package tests contains REST API tests for following endpoints:
//
// apiPrefix
//
// apiPrefix+"organizations"
//
// apiPrefix+"report/{organization}/{cluster}"
//
// apiPrefix+"clusters/{cluster}/rules/{rule_id}/like"
//
// apiPrefix+"clusters/{cluster}/rules/{rule_id}/dislike"
//
// apiPrefix+"clusters/{cluster}/rules/{rule_id}/reset_vote"
//
// apiPrefix+"organizations/{organization}/clusters"
package tests

// ServerTests run all tests for basic REST API endpoints
func ServerTests() {
	BasicTests()
	OrganizationsTests()
	ClustersTests()
	ClustersDetailTests()
	ReportsTests()
	ReportsMetadataTests()
	MultipleReportsTests()
	MultipleReportsTestsUsingPostMethod()
	VoteTests()
	DisableRuleTests()

	// tests for OpenAPI specification that is accessible via its endpoint as well
	// implementation of these tests is stored in openapi.go
	checkOpenAPISpecifications()

	// tests for metrics hat is accessible via its endpoint as well
	// implementation of these tests is stored in metrics.go
	checkPrometheusMetrics()

	// tests for /info endpoint
	checkInfoEndpoint()
}

// BasicTests implements basic tests for REST API apiPrefix
func BasicTests() {
	// implementation of these tests is stored in entrypoint.go
	checkRestAPIEntryPoint()
	checkNonExistentEntryPoint()
	checkWrongEntryPoint()
	checkWrongMethodsForEntryPoint()
}

// OrganizationsTests implements tests for REST API endpoints apiPrefix+"organizations"
func OrganizationsTests() {
	// implementation of these tests is stored in organizations.go
	checkOrganizationsEndpoint()
	checkOrganizationsEndpointWrongMethods()
}

// ClustersTests implements tests for REST API endpoints apiPrefix+"report/{organization}/{cluster}"
func ClustersTests() {
	// implementation of these tests is stored in org_clusters.go
	checkClustersEndpointForKnownOrganizations()
	checkClustersEndpointForUnknownOrganizations()
	checkClustersEndpointForImproperOrganizations()
	checkClustersEndpointWrongMethods()
	checkClustersEndpointSpecialOrganizationIds()
}

// ClustersDetailTests implements tests for REST API endpoints apiPrefix+"rules/{rule_selector}/organizations/{org_id}/users/{user_id}/clusters_detail"
func ClustersDetailTests() {
	// implementation of these tests is stored in clusters_detail.go
	checkClustersEndpointDetailGet()
	checkClustersDetailsEndpointWrongMethods()
	checkClustersDetailsEndpointForKnownOrganizationNoAuthHeaderCase()
}

// ReportsTests implements tests for REST API endpoints apiPrefix+"report/{organization}/{cluster}"
func ReportsTests() {
	// implementation of these tests is stored in reports.go
	checkReportEndpointForKnownOrganizationAndKnownCluster()
	checkReportEndpointForKnownOrganizationAndUnknownCluster()
	checkReportEndpointForUnknownOrganizationAndKnownCluster()
	checkReportEndpointForUnknownOrganizationAndUnknownCluster()
	checkReportEndpointForImproperOrganization()
	checkReportEndpointWrongMethods()
	// unauthorized access
	checkReportEndpointForKnownOrganizationAndKnownClusterUnauthorizedCase()
	checkReportEndpointForKnownOrganizationAndUnknownClusterUnauthorizedCase()
	checkReportEndpointForUnknownOrganizationAndKnownClusterUnauthorizedCase()
	checkReportEndpointForUnknownOrganizationAndUnknownClusterUnauthorizedCase()
	checkReportEndpointForImproperOrganizationUnauthorizedCase()
	// reproducers
	reproducerForIssue384()
}

// ReportsMetadataTests implements tests for REST API endpoints apiPrefix+"report/{organization}/{cluster}/info"
func ReportsMetadataTests() {
	// implementation of these tests is stored in reports_info.go
	checkReportInfoEndpointForKnownOrganizationAndKnownCluster()
	checkReportInfoEndpointForKnownOrganizationAndUnknownCluster()
	checkReportInfoEndpointForUnknownOrganizationAndKnownCluster()
	checkReportInfoEndpointForUnknownOrganizationAndUnknownCluster()
	checkReportInfoEndpointForImproperOrganization()
	checkReportInfoEndpointWrongMethods()
	checkReportInfoEndpointForKnownOrganizationAndKnownClusterUnauthorizedCase()
	checkReportInfoEndpointForKnownOrganizationAndUnknownClusterUnauthorizedCase()
	checkReportInfoEndpointForUnknownOrganizationAndKnownClusterUnauthorizedCase()
	checkReportInfoEndpointForUnknownOrganizationAndUnknownClusterUnauthorizedCase()
	checkReportInfoEndpointForImproperOrganizationUnauthorizedCase()
	// unauthorized access
}

// MultipleReportsTests function implements tests for REST API endpoints
// apiPrefix+"organizations/{organization}/clusters/{clusterList}/reports"
func MultipleReportsTests() {
	checkMultipleReportsForKnownOrganizationAnd1KnownCluster()
	checkMultipleReportsForKnownOrganizationAnd2KnownClusters()
	checkMultipleReportsForKnownOrganizationAnd3KnownClusters()
	checkMultipleReportsForKnownOrganizationAndUnknownCluster()
	checkMultipleReportsForKnownOrganizationAndKnownAndUnknownCluster()
	checkMultipleReportsForKnownOrganizationAnd2KnownAndUnknownCluster()
	checkMultipleReportsForKnownOrganizationAnd3KnownAndUnknownCluster()
	checkMultipleReportsForUnknownOrganizationAnd1KnownCluster()
	checkMultipleReportsForUnknownOrganizationAndUnknownCluster()
	// unauthorized access
	checkMultipleReportsForKnownOrganizationAnd1KnownClusterUnauthorizedCase()
	checkMultipleReportsForKnownOrganizationAnd2KnownClustersUnauthorizedCase()
	checkMultipleReportsForKnownOrganizationAnd3KnownClustersUnauthorizedCase()
	checkMultipleReportsForKnownOrganizationAndUnknownClusterUnauthorizedCase()
	checkMultipleReportsForKnownOrganizationAndKnownAndUnknownClusterUnauthorizedCase()
	checkMultipleReportsForKnownOrganizationAnd2KnownAndUnknownClusterUnauthorizedCase()
	checkMultipleReportsForKnownOrganizationAnd3KnownAndUnknownClusterUnauthorizedCase()
	checkMultipleReportsForUnknownOrganizationAnd1KnownClusterUnauthorizedCase()
	checkMultipleReportsForUnknownOrganizationAndUnknownClusterUnauthorizedCase()
}

// MultipleReportsTestsUsingPostMethod function implements tests for REST API
// endpoints apiPrefix+"organizations/{organization}/clusters/reports" in a
// variant that accepts cluster list in request payload
func MultipleReportsTestsUsingPostMethod() {
	checkMultipleReportsForKnownOrganizationAnd1KnownClusterUsingPostMethod()
	checkMultipleReportsForKnownOrganizationAnd2KnownClustersUsingPostMethod()
	checkMultipleReportsForKnownOrganizationAnd3KnownClustersUsingPostMethod()
	checkMultipleReportsForKnownOrganizationAndUnknownClusterUsingPostMethod()
	checkMultipleReportsForKnownOrganizationAndKnownAndUnknownClusterUsingPostMethod()
	checkMultipleReportsForKnownOrganizationAnd2KnownAndUnknownClusterUsingPostMethod()
	checkMultipleReportsForKnownOrganizationAnd3KnownAndUnknownClusterUsingPostMethod()
	checkMultipleReportsForUnknownOrganizationAnd1KnownClusterUsingPostMethod()
	checkMultipleReportsForUnknownOrganizationAndUnknownClusterUsingPostMethod()
	// unauthorized access
	checkMultipleReportsForKnownOrganizationAnd1KnownClusterUnauthorizedCaseUsingPostMethod()
	checkMultipleReportsForKnownOrganizationAnd2KnownClustersUnauthorizedCaseUsingPostMethod()
	checkMultipleReportsForKnownOrganizationAnd3KnownClustersUnauthorizedCaseUsingPostMethod()
	checkMultipleReportsForKnownOrganizationAndUnknownClusterUnauthorizedCaseUsingPostMethod()
	checkMultipleReportsForKnownOrganizationAndKnownAndUnknownClusterUnauthorizedCaseUsingPostMethod()
	checkMultipleReportsForKnownOrganizationAnd2KnownAndUnknownClusterUnauthorizedCaseUsingPostMethod()
	checkMultipleReportsForKnownOrganizationAnd3KnownAndUnknownClusterUnauthorizedCaseUsingPostMethod()
	checkMultipleReportsForUnknownOrganizationAnd1KnownClusterUnauthorizedCaseUsingPostMethod()
	checkMultipleReportsForUnknownOrganizationAndUnknownClusterUnauthorizedCaseUsingPostMethod()
}

// VoteTests implements tests for REST API endpoints for voting about rules
func VoteTests() {
	// implementation of these tests is stored in rule_vote.go
	checkLikeKnownRuleForKnownCluster()
	checkDislikeKnownRuleForKnownCluster()
	checkResetKnownRuleForKnownCluster()
	checkLikeKnownRuleForUnknownCluster()
	checkDislikeKnownRuleForUnknownCluster()
	checkResetKnownRuleForUnknownCluster()
	checkLikeKnownRuleForImproperCluster()
	checkDislikeKnownRuleForImproperCluster()
	checkResetKnownRuleForImproperCluster()
	reproducerForIssue385()
	checkGetUserVoteForKnownCluster()
	checkGetUserVoteForUnknownCluster()
	checkGetUserVoteForImproperCluster()
	checkGetUserVoteAfterVote()
	checkGetUserVoteAfterUnvote()
	checkGetUserVoteAfterDoubleVote()
	checkGetUserVoteAfterDoubleUnvote()
}

// DisableRuleTests implements tests for REST API endpoints for disabling etc.
// rules
func DisableRuleTests() {
	checkListOfDisabledRules()
	checkListOfDisabledRulesFeedback()
}
