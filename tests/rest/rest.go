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
	// basic tests for REST API apiPrefix
	// implementation of these tests is stored in entrypoint.go
	checkRestAPIEntryPoint()
	checkNonExistentEntryPoint()
	checkWrongEntryPoint()
	checkWrongMethodsForEntryPoint()

	// tests for REST API endpoints apiPrefix+"organizations"
	// implementation of these tests is stored in organizations.go
	checkOrganizationsEndpoint()
	checkOrganizationsEndpointWrongMethods()

	// tests for REST API endpoints apiPrefix+"report/{organization}/{cluster}"
	// implementation of these tests is stored in org_clusters.go
	checkClustersEndpointForKnownOrganizations()
	checkClustersEndpointForUnknownOrganizations()
	checkClustersEndpointForImproperOrganizations()
	checkClustersEndpointWrongMethods()
	checkClustersEndpointSpecialOrganizationIds()

	// tests for REST API endpoints apiPrefix+"report/{organization}/{cluster}"
	// implementation of these tests is stored in reports.go
	checkReportEndpointForKnownOrganizationAndKnownCluster()
	checkReportEndpointForKnownOrganizationAndUnknownCluster()
	checkReportEndpointForUnknownOrganizationAndKnownCluster()
	checkReportEndpointForUnknownOrganizationAndUnknownCluster()
	checkReportEndpointForImproperOrganization()
	checkReportEndpointWrongMethods()
	reproducerForIssue384()

	// tests for REST API endpoints for voting about rules
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
	checkLikeUnknownRuleForKnownCluster()
	checkDislikeUnknownRuleForKnownCluster()
	checkResetUnknownRuleForKnownCluster()
	checkLikeUnknownRuleForUnknownCluster()
	checkDislikeUnknownRuleForUnknownCluster()
	checkResetUnknownRuleForUnknownCluster()
	checkLikeUnknownRuleForImproperCluster()
	checkDislikeUnknownRuleForImproperCluster()
	checkResetUnknownRuleForImproperCluster()
	reproducerForIssue385()
	checkGetUserVoteForKnownCluster()
	checkGetUserVoteForUnknownCluster()
	checkGetUserVoteForImproperCluster()
	checkGetUserVoteForUnknownRuleAndKnownCluster()
	checkGetUserVoteForUnknownRuleAndUnknownCluster()
	checkGetUserVoteForUnknownRuleAndImproperCluster()
	checkGetUserVoteAfterVote()
	checkGetUserVoteAfterUnvote()
	checkGetUserVoteAfterDoubleVote()
	checkGetUserVoteAfterDoubleUnvote()

	// tests for OpenAPI specification that is accessible via its endpoint as well
	// implementation of these tests is stored in openapi.go
	checkOpenAPISpecifications()

	// tests for metrics hat is accessible via its endpoint as well
	// implementation of these tests is stored in metrics.go
	checkPrometheusMetrics()
}
