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
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/verdverm/frisby"
)

const (
	apiURL            = "http://localhost:8080/api/v1/"
	contentTypeHeader = "Content-Type"

	// ContentTypeJSON represents MIME type for JSON format
	ContentTypeJSON = "application/json; charset=utf-8"

	// ContentTypeText represents MIME type for plain text format
	ContentTypeText = "text/plain; charset=utf-8"

	knownClusterForOrganization1   = "00000000-0000-0000-0000-000000000000"
	unknownClusterForOrganization1 = "00000000-0000-0000-0000-000000000001"
)

// list of known organizations that are stored in test database
var knownOrganizations []int = []int{1, 2, 3, 4}

// list of unknown organizations that are not stored in test database
var unknownOrganizations []int = []int{5, 6, 7, 8}

// list of improper organization IDs
var improperOrganizations []int = []int{-1000, -1, 0}

// checkRestAPIEntryPoint check if the entry point (usually /api/v1/) responds correctly to HTTP GET command
func checkRestAPIEntryPoint() {
	f := frisby.Create("Check the entry point to REST API using HTTP GET method").Get(apiURL)
	f.Send()
	f.ExpectStatus(200)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
	f.PrintReport()
}

// checkNonExistentEntryPoint check whether non-existing endpoints are handled properly (HTTP code 404 etc.)
func checkNonExistentEntryPoint() {
	f := frisby.Create("Check the non-existent entry point to REST API").Get(apiURL + "foobar")
	f.Send()
	f.ExpectStatus(404)
	f.ExpectHeader(contentTypeHeader, ContentTypeText)
	f.PrintReport()
}

// checkWrongEntryPoint check whether wrongly scecified URLs are handled correctly
func checkWrongEntryPoint() {
	postfixes := [...]string{"..", "../", "...", "..?", "..?foobar"}
	for _, postfix := range postfixes {
		f := frisby.Create("Check the wrong entry point to REST API with postfix '" + postfix + "'").Get(apiURL + postfix)
		f.Send()
		f.ExpectStatus(404)
		f.ExpectHeader(contentTypeHeader, ContentTypeText)
		f.PrintReport()
	}
}

// sendAndExpectStatus sends the request to the server and checks whether expected HTTP code (status) is returned
func sendAndExpectStatus(f *frisby.Frisby, expectedStatus int) {
	f.Send()
	f.ExpectStatus(405)
	f.PrintReport()
}

// checkGetEndpointByOtherMethods checks whether a 'GET' endpoint respond correctly if other HTTP methods are used
func checkGetEndpointByOtherMethods(endpoint string) {
	f := frisby.Create("Check the end point " + endpoint + " with wrong method: POST").Post(apiURL)
	sendAndExpectStatus(f, 405)

	f = frisby.Create("Check the entry point " + endpoint + " with wrong method: PUT").Put(apiURL)
	sendAndExpectStatus(f, 405)

	f = frisby.Create("Check the entry point " + endpoint + " with wrong method: DELETE").Delete(apiURL)
	sendAndExpectStatus(f, 405)

	f = frisby.Create("Check the entry point " + endpoint + " with wrong method: PATCH").Patch(apiURL)
	sendAndExpectStatus(f, 405)

	f = frisby.Create("Check the entry point " + endpoint + " with wrong method: OPTIONS").Options(apiURL)
	sendAndExpectStatus(f, 405)

	f = frisby.Create("Check the entry point " + endpoint + " with wrong method: HEAD").Head(apiURL)
	sendAndExpectStatus(f, 405)
}

// check whether other HTTP methods are rejected correctly for the REST API entry point
func checkWrongMethodsForEntryPoint() {
	checkGetEndpointByOtherMethods(apiURL)
}

// OrganizationsResponse represents response containing list of organizations
type OrganizationsResponse struct {
	Organizations []int  `json:"organizations"`
	Status        string `json:"status"`
}

func readOrganizationsFromResponse(f *frisby.Frisby) OrganizationsResponse {
	response := OrganizationsResponse{}
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

// checkOrganizationsEndpoint check if the end point to return list of organizations responds correctly to HTTP GET command
func checkOrganizationsEndpoint() {
	f := frisby.Create("Check the end point to return list of organizations by HTTP GET method").Get(apiURL + "organizations")
	f.Send()
	f.ExpectStatus(200)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
	organizationsResponse := readOrganizationsFromResponse(f)
	if organizationsResponse.Status != "ok" {
		f.AddError(fmt.Sprintf("Expected status is 'ok', but got '%s' instead", organizationsResponse.Status))
	}
	expectedOrglist := []int{1, 2, 3, 4}
	if !reflect.DeepEqual(organizationsResponse.Organizations, expectedOrglist) {
		f.AddError(fmt.Sprintf("Expected the following organizations %v, but got %v instead", expectedOrglist, organizationsResponse.Organizations))
	}
	f.PrintReport()
}

// checkOrganizationsEndpointWrongMethods check if the end point to return list of arganizations responds correctly to other methods than HTTP GET
func checkOrganizationsEndpointWrongMethods() {
	checkGetEndpointByOtherMethods(apiURL + "organizations")
}

func constructURLForOrganizationsClusters(organization int) string {
	orgID := strconv.Itoa(organization)
	return apiURL + "organizations/" + orgID + "/clusters"
}

// checkClustersEndpointForKnownOrganizations check if the end point to return list of clusters responds correctly to HTTP GET command
func checkClustersEndpointForKnownOrganizations() {
	for _, knownOrganization := range knownOrganizations {
		url := constructURLForOrganizationsClusters(knownOrganization)
		f := frisby.Create("Check the end point to return list of clusters for known organization by HTTP GET method").Get(url)
		f.Send()
		f.ExpectStatus(200)
		f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
		f.PrintReport()
	}
}

// checkClustersEndpointForUnknownOrganizations check if the end point to return list of clusters responds correctly to HTTP GET command
func checkClustersEndpointForUnknownOrganizations() {
	for _, unknownOrganization := range unknownOrganizations {
		url := constructURLForOrganizationsClusters(unknownOrganization)
		f := frisby.Create("Check the end point to return list of clusters for unknown organization by HTTP GET method").Get(url)
		f.Send()
		f.ExpectStatus(200)
		f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
		f.PrintReport()
	}
}

// checkClustersEndpointForImproperOrganizations check if the end point to return list of clusters responds correctly to HTTP GET command
func checkClustersEndpointForImproperOrganizations() {
	for _, improperOrganization := range improperOrganizations {
		url := constructURLForOrganizationsClusters(improperOrganization)
		f := frisby.Create("Check the end point to return list of clusters by HTTP GET method").Get(url)
		f.Send()
		f.ExpectStatus(400)
		f.PrintReport()
	}
}

// checkClustersEndpointWrongMethods check if the end point to return list of arganizations responds correctly to other methods than HTTP GET
func checkClustersEndpointWrongMethods() {
	for _, knownOrganization := range knownOrganizations {
		orgID := strconv.Itoa(knownOrganization)
		url := apiURL + "organizations/" + orgID + "/clusters"
		checkGetEndpointByOtherMethods(url)
	}
}

func constructURLForResultForOrgCluster(organizationID string, clusterID string) string {
	return apiURL + "report/" + organizationID + "/" + clusterID
}

// checkReportEndpointForKnownOrganizationAndKnownCluster check if the endpoint to return report works as expected
func checkReportEndpointForKnownOrganizationAndKnownCluster() {
	url := constructURLForResultForOrgCluster("1", knownClusterForOrganization1)
	f := frisby.Create("Check the end point to return report for existing organization and cluster ID").Get(url)
	f.Send()
	f.ExpectStatus(200)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
	f.PrintReport()
}

// checkReportEndpointForKnownOrganizationAndUnknownCluster check if the endpoint to return report works as expected
func checkReportEndpointForKnownOrganizationAndUnknownCluster() {
	url := constructURLForResultForOrgCluster("1", unknownClusterForOrganization1)
	f := frisby.Create("Check the end point to return report for existing organization and non-existing cluster ID").Get(url)
	f.Send()
	f.ExpectStatus(404)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
	f.PrintReport()
}

// checkReportEndpointForUnknownOrganizationAndKnownCluster check if the endpoint to return report works as expected
func checkReportEndpointForUnknownOrganizationAndKnownCluster() {
	url := constructURLForResultForOrgCluster("100000", knownClusterForOrganization1)
	f := frisby.Create("Check the end point to return report for non-existing organization and non-existing cluster ID").Get(url)
	f.Send()
	f.ExpectStatus(404)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
	f.PrintReport()
}

// checkReportEndpointForUnknownOrganizationAndUnknownCluster check if the endpoint to return report works as expected
func checkReportEndpointForUnknownOrganizationAndUnknownCluster() {
	url := constructURLForResultForOrgCluster("100000", unknownClusterForOrganization1)
	f := frisby.Create("Check the end point to return report for non-existing organization and non-existing cluster ID").Get(url)
	f.Send()
	f.ExpectStatus(404)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
	f.PrintReport()
}

// checkReportEndpointForImproperOrganization check if the endpoint to return report works as expected
func checkReportEndpointForImproperOrganization() {
	url := constructURLForResultForOrgCluster("foobar", knownClusterForOrganization1)
	f := frisby.Create("Check the end point to return report for improper organization").Get(url)
	f.Send()
	f.ExpectStatus(400)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
	f.PrintReport()
}

// checkReportEndpointWrongMethods check if the end point to return results responds correctly to other methods than HTTP GET
func checkReportEndpointWrongMethods() {
	url := constructURLForResultForOrgCluster("1", knownClusterForOrganization1)
	checkGetEndpointByOtherMethods(url)
}

// checkOpenAPISpecifications checks whether OpenAPI endpoint is handled correctly
func checkOpenAPISpecifications() {
	f := frisby.Create("Check the OpenAPI endpoint").Get(apiURL + "openapi.json")
	f.Send()
	f.ExpectStatus(200)
	f.ExpectHeader(contentTypeHeader, "application/json")
	f.PrintReport()
}

// checkPrometheusMetrics checks the Prometheus metrics API endpoint
func checkPrometheusMetrics() {
	f := frisby.Create("Check the Prometheus metrics API endpoint").Get(apiURL + "metrics")
	f.Send()
	f.ExpectStatus(200)
	// the content type header set by metrics handler is a bit complicated
	// but it must start with "text/plain" in any case
	f.Expect(func(F *frisby.Frisby) (bool, string) {
		header := F.Resp.Header.Get(contentTypeHeader)
		if strings.HasPrefix(header, "text/plain") {
			return true, "ok"
		}
		return false, fmt.Sprintf("Expected Header %q to be %q, but got %q", contentTypeHeader, "text/plain", header)
	})
	f.PrintReport()
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

	// tests for REST API endpoints apiPrefix+"report/{organization}/{cluster}"
	checkReportEndpointForKnownOrganizationAndKnownCluster()
	checkReportEndpointForKnownOrganizationAndUnknownCluster()
	checkReportEndpointForUnknownOrganizationAndKnownCluster()
	checkReportEndpointForUnknownOrganizationAndUnknownCluster()
	checkReportEndpointForImproperOrganization()
	checkReportEndpointWrongMethods()

	// tests for OpenAPI specification that is accessible via its endpoint as well
	checkOpenAPISpecifications()

	// tests for metrics hat is accessible via its endpoint as well
	checkPrometheusMetrics()
}
