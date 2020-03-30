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
	"strconv"

	"github.com/verdverm/frisby"
)

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

// checkClustersEndpointSpecialOrganizationIds is an implementation of several reproducers
func checkClustersEndpointSpecialOrganizationIds() {
	const MaxInt = int(^uint(0) >> 1)

	var orgIDs []int = []int{
		2147483647, // maxint32
		2147483648, // reproducer for https://github.com/RedHatInsights/insights-results-aggregator/issues/383
		MaxInt,
	}
	for _, orgID := range orgIDs {
		url := constructURLForOrganizationsClusters(orgID)
		f := frisby.Create("Check the end point to return list of clusters for organization with special ID by HTTP GET method").Get(url)
		f.Send()
		f.ExpectStatus(200)
		f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
		f.PrintReport()
	}
}
