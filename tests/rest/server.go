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

package tests

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	"github.com/verdverm/frisby"
)

const (
	apiURL            = "http://localhost:8080/api/v1/"
	contentTypeHeader = "Content-Type"
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
	f.ExpectHeader(contentTypeHeader, "application/json; charset=utf-8")
	f.PrintReport()
}

// checkNonExistentEntryPoint check whether non-existing endpoints are handled properly (HTTP code 404 etc.)
func checkNonExistentEntryPoint() {
	f := frisby.Create("Check the non-existent entry point to REST API").Get(apiURL + "foobar")
	f.Send()
	f.ExpectStatus(404)
	f.ExpectHeader(contentTypeHeader, "text/plain; charset=utf-8")
	f.PrintReport()
}

// checkWrongEntryPoint check whether wrongly scecified URLs are handled correctly
func checkWrongEntryPoint() {
	postfixes := [...]string{"..", "../", "...", "..?", "..?foobar"}
	for _, postfix := range postfixes {
		f := frisby.Create("Check the wrong entry point to REST API with postfix '" + postfix + "'").Get(apiURL + postfix)
		f.Send()
		f.ExpectStatus(404)
		f.ExpectHeader(contentTypeHeader, "text/plain; charset=utf-8")
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
	f.ExpectHeader(contentTypeHeader, "application/json; charset=utf-8")
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
		f := frisby.Create("Check the end point to return list of clusters by HTTP GET method").Get(url)
		f.Send()
		f.ExpectStatus(200)
		f.ExpectHeader(contentTypeHeader, "application/json; charset=utf-8")
		f.PrintReport()
	}
}

// checkClustersEndpointForUnknownOrganizations check if the end point to return list of clusters responds correctly to HTTP GET command
func checkClustersEndpointForUnknownOrganizations() {
	for _, unknownOrganization := range unknownOrganizations {
		url := constructURLForOrganizationsClusters(unknownOrganization)
		f := frisby.Create("Check the end point to return list of clusters by HTTP GET method").Get(url)
		f.Send()
		f.ExpectStatus(200)
		f.ExpectHeader(contentTypeHeader, "application/json; charset=utf-8")
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

// checkOpenAPISpecifications checks whether OpenAPI endpoint is handled correctly
func checkOpenAPISpecifications() {
	f := frisby.Create("Check the wrong entry point to REST API").Get(apiURL + "openapi.json")
	f.Send()
	f.ExpectStatus(200)
	f.ExpectHeader(contentTypeHeader, "application/json")
	f.PrintReport()
}

// ServerTests run all tests for basic REST API endpoints
func ServerTests() {
	checkRestAPIEntryPoint()
	checkNonExistentEntryPoint()
	checkWrongEntryPoint()
	checkWrongMethodsForEntryPoint()
	checkOrganizationsEndpoint()
	checkOrganizationsEndpointWrongMethods()
	checkClustersEndpointForKnownOrganizations()
	checkClustersEndpointForUnknownOrganizations()
	checkClustersEndpointForImproperOrganizations()
	checkClustersEndpointWrongMethods()
	checkOpenAPISpecifications()
}
