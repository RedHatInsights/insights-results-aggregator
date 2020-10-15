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
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/verdverm/frisby"
)

// ClustersResponse represents response containing list of clusters for given organization
type ClustersResponse struct {
	Clusters []string `json:"clusters"`
	Status   string   `json:"status"`
}

func constructURLForOrganizationsClusters(organization int) string {
	orgID := strconv.Itoa(organization)
	return apiURL + "organizations/" + orgID + "/clusters"
}

// readClustersFromResponse reads and parses information about clusters from response body
func readClustersFromResponse(f *frisby.Frisby) ClustersResponse {
	response := ClustersResponse{}
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

// checkEmptyListOfClusters tests whether list of clusters returned from the server is empty
func checkEmptyListOfClusters(f *frisby.Frisby, response ClustersResponse) {
	if len(response.Clusters) != 0 {
		f.AddError("List of clusters needs to be empty for unknown organization")
	}
}

// checkNonEmptyListOfClusters tests whether list of clusters returned from the server is not empty and that the list are correct
func checkNonEmptyListOfClusters(f *frisby.Frisby, response ClustersResponse) {
	if len(response.Clusters) == 0 {
		f.AddError("Clusters are not read")
	}
	for _, cluster := range response.Clusters {
		_, err := uuid.Parse(cluster)
		if err != nil {
			f.AddError(fmt.Sprintf("cluster name is not a UUID: %s", cluster))
		}
	}
}

// checkClustersEndpointForKnownOrganizations check if the end point to return list of clusters responds correctly to HTTP GET command
func checkClustersEndpointForKnownOrganizations() {
	for _, knownOrganization := range knownOrganizations {
		url := constructURLForOrganizationsClusters(knownOrganization)
		f := frisby.Create("Check the end point to return list of clusters for known organization by HTTP GET method").Get(url)
		setAuthHeaderForOrganization(f, knownOrganization)
		f.Send()
		f.ExpectStatus(200)
		f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
		response := readClustersFromResponse(f)
		checkOkStatusResponse(f, response)
		checkNonEmptyListOfClusters(f, response)
		f.PrintReport()
	}
}

// checkClustersEndpointForUnknownOrganizations check if the end point to return list of clusters responds correctly to HTTP GET command
func checkClustersEndpointForUnknownOrganizations() {
	for _, unknownOrganization := range unknownOrganizations {
		url := constructURLForOrganizationsClusters(unknownOrganization)
		f := frisby.Create("Check the end point to return list of clusters for unknown organization by HTTP GET method").Get(url)
		setAuthHeaderForOrganization(f, unknownOrganization)
		f.Send()
		f.ExpectStatus(200)
		f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
		response := readClustersFromResponse(f)
		checkOkStatusResponse(f, response)
		checkEmptyListOfClusters(f, response)
		f.PrintReport()
	}
}

// checkClustersEndpointForImproperOrganizations check if the end point to return list of clusters responds correctly to HTTP GET command
func checkClustersEndpointForImproperOrganizations() {
	for _, improperOrganization := range improperOrganizations {
		url := constructURLForOrganizationsClusters(improperOrganization)
		f := frisby.Create("Check the end point to return list of clusters by HTTP GET method").Get(url)
		setAuthHeaderForOrganization(f, improperOrganization)
		f.Send()
		if improperOrganization == 0 {
			f.ExpectStatus(http.StatusBadRequest)
		} else {
			// negative values are not unmarshalled properly
			f.ExpectStatus(http.StatusUnauthorized)
		}
		f.PrintReport()
	}
}

// checkClustersEndpointWrongMethods check if the end point to return list of arganizations responds correctly to other methods than HTTP GET
func checkClustersEndpointWrongMethods() {
	for _, knownOrganization := range knownOrganizations {
		orgID := strconv.Itoa(knownOrganization)
		url := apiURL + "organizations/" + orgID + "/clusters"
		checkGetEndpointByOtherMethods(url, false)
	}
}

// checkClustersEndpointSpecialOrganizationIds is an implementation of several reproducers
func checkClustersEndpointSpecialOrganizationIds() {
	var orgIDs []int = []int{
		2147483647, // maxint32
		2147483648, // reproducer for https://github.com/RedHatInsights/insights-results-aggregator/issues/383
	}
	for _, orgID := range orgIDs {
		url := constructURLForOrganizationsClusters(orgID)
		f := frisby.Create("Check the end point to return list of clusters for organization with special ID by HTTP GET method").Get(url)
		setAuthHeaderForOrganization(f, orgID)
		f.Send()
		f.ExpectStatus(200)
		f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
		response := readClustersFromResponse(f)
		checkOkStatusResponse(f, response)
		checkEmptyListOfClusters(f, response)
		f.PrintReport()
	}
}
