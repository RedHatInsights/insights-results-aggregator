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

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/verdverm/frisby"
)

// common constants used by REST API tests
const (
	apiURL              = "http://localhost:8080/api/v1/"
	contentTypeHeader   = "Content-Type"
	contentLengthHeader = "Content-Length"

	authHeaderName = "x-rh-identity"

	// ContentTypeJSON represents MIME type for JSON format
	ContentTypeJSON = "application/json; charset=utf-8"

	// ContentTypeText represents MIME type for plain text format
	ContentTypeText = "text/plain; charset=utf-8"

	knownClusterForOrganization1   = "00000000-0000-0000-0000-000000000000"
	unknownClusterForOrganization1 = "00000000-0000-0000-0000-000000000001"
)

// StatusOnlyResponse represents response containing just a status
type StatusOnlyResponse struct {
	Status string `json:"status"`
}

// list of known organizations that are stored in test database
var knownOrganizations = []int{1, 2, 3, 4}

// list of unknown organizations that are not stored in test database
var unknownOrganizations = []int{5, 6, 7, 8}

// list of improper organization IDs
var improperOrganizations = []int{-1000, -1, 0}

// setAuthHeaderForOrganization set authorization header to request
func setAuthHeaderForOrganization(f *frisby.Frisby, orgID int) {
	plainHeader := fmt.Sprintf("{\"identity\": {\"internal\": {\"org_id\": \"%d\"}}}", orgID)
	encodedHeader := base64.StdEncoding.EncodeToString([]byte(plainHeader))
	f.SetHeader(authHeaderName, encodedHeader)
}

// setAuthHeader set authorization header to request for organization 1
func setAuthHeader(f *frisby.Frisby) {
	setAuthHeaderForOrganization(f, 1)
}

// readStatusFromResponse reads and parses status from response body
func readStatusFromResponse(f *frisby.Frisby) StatusOnlyResponse {
	response := StatusOnlyResponse{}
	text, err := f.Resp.Content()
	if err != nil {
		f.AddError(err.Error())
	} else {
		err := json.Unmarshal(text, &response)
		if err != nil {
			f.AddError(err.Error())
		}
	}
	return response
}

// sendAndExpectStatus sends the request to the server and checks whether expected HTTP code (status) is returned
func sendAndExpectStatus(f *frisby.Frisby, expectedStatus int) {
	f.Send()
	f.ExpectStatus(expectedStatus)
	f.PrintReport()
}

// checkGetEndpointByOtherMethods checks whether a 'GET' endpoint respond correctly if other HTTP methods are used
func checkGetEndpointByOtherMethods(endpoint string, includingOptions bool) {
	f := frisby.Create("Check the end point " + endpoint + " with wrong method: POST").Post(endpoint)
	sendAndExpectStatus(f, 405)

	f = frisby.Create("Check the entry point " + endpoint + " with wrong method: PUT").Put(endpoint)
	sendAndExpectStatus(f, 405)

	f = frisby.Create("Check the entry point " + endpoint + " with wrong method: DELETE").Delete(endpoint)
	sendAndExpectStatus(f, 405)

	f = frisby.Create("Check the entry point " + endpoint + " with wrong method: PATCH").Patch(endpoint)
	sendAndExpectStatus(f, 405)

	f = frisby.Create("Check the entry point " + endpoint + " with wrong method: HEAD").Head(endpoint)
	sendAndExpectStatus(f, 405)

	// some endpoints accepts OPTIONS method together with GET one, so this check is fully optional
	if includingOptions {
		f = frisby.Create("Check the entry point " + endpoint + " with wrong method: OPTIONS").Options(endpoint)
		sendAndExpectStatus(f, 405)
	}
}

// ServerTests run all tests for basic REST API endpoints
func ServerTests() {
	// basic tests for REST API apiPrefix
	checkRestAPIEntryPoint()
	checkNonExistentEntryPoint()
	checkWrongEntryPoint()
	checkWrongMethodsForEntryPoint()

	// tests for REST API endpoints apiPrefix+"organizations"
	checkOrganizationsEndpoint()
	checkOrganizationsEndpointWrongMethods()

	// tests for REST API endpoints apiPrefix+"report/{organization}/{cluster}"
	checkClustersEndpointForKnownOrganizations()
	checkClustersEndpointForUnknownOrganizations()
	checkClustersEndpointForImproperOrganizations()
	checkClustersEndpointWrongMethods()
	checkClustersEndpointSpecialOrganizationIds()

	// tests for REST API endpoints apiPrefix+"report/{organization}/{cluster}"
	checkReportEndpointForKnownOrganizationAndKnownCluster()
	checkReportEndpointForKnownOrganizationAndUnknownCluster()
	checkReportEndpointForUnknownOrganizationAndKnownCluster()
	checkReportEndpointForUnknownOrganizationAndUnknownCluster()
	checkReportEndpointForImproperOrganization()
	checkReportEndpointWrongMethods()
	reproducerForIssue384()

	// tests for REST API endpoints for voting about rules
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

	// tests for OpenAPI specification that is accessible via its endpoint as well
	checkOpenAPISpecifications()

	// tests for metrics hat is accessible via its endpoint as well
	checkPrometheusMetrics()
}
