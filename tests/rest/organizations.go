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

	"github.com/verdverm/frisby"
)

// OrganizationsResponse represents response containing list of organizations
type OrganizationsResponse struct {
	Organizations []int  `json:"organizations"`
	Status        string `json:"status"`
}

// readOrganizationsFromResponse reads and parses information about organization from response body
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
