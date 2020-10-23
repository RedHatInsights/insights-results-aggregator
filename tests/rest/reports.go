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
	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/verdverm/frisby"

	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// constructURLForReportForOrgCluster function constructs an URL to access the
// endpoint to retrieve results for given cluster from selected organization
func constructURLForReportForOrgCluster(organizationID string, clusterID string, userID types.UserID) string {
	return httputils.MakeURLToEndpoint(apiURL, server.ReportEndpoint, organizationID, clusterID, userID)
}

// checkReportEndpointForKnownOrganizationAndKnownCluster check if the endpoint to return report works as expected
func checkReportEndpointForKnownOrganizationAndKnownCluster() {
	url := constructURLForReportForOrgCluster("1", knownClusterForOrganization1, testdata.UserID)
	f := frisby.Create("Check the endpoint to return report for existing organization and cluster ID").Get(url)
	setAuthHeader(f)
	f.Send()
	f.ExpectStatus(200)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
	f.PrintReport()
}

// checkReportEndpointForKnownOrganizationAndUnknownCluster check if the endpoint to return report works as expected
func checkReportEndpointForKnownOrganizationAndUnknownCluster() {
	url := constructURLForReportForOrgCluster("1", unknownClusterForOrganization1, testdata.UserID)
	f := frisby.Create("Check the endpoint to return report for existing organization and non-existing cluster ID").Get(url)
	setAuthHeader(f)
	f.Send()
	f.ExpectStatus(404)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
	f.PrintReport()
}

// checkReportEndpointForUnknownOrganizationAndKnownCluster check if the endpoint to return report works as expected
func checkReportEndpointForUnknownOrganizationAndKnownCluster() {
	url := constructURLForReportForOrgCluster(unknownOrganizationID, knownClusterForOrganization1, testdata.UserID)
	f := frisby.Create("Check the endpoint to return report for non-existing organization and non-existing cluster ID").Get(url)
	setAuthHeaderForOrganization(f, 100000)
	f.Send()
	f.ExpectStatus(404)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
	f.PrintReport()
}

// checkReportEndpointForUnknownOrganizationAndUnknownCluster check if the endpoint to return report works as expected
func checkReportEndpointForUnknownOrganizationAndUnknownCluster() {
	url := constructURLForReportForOrgCluster(unknownOrganizationID, unknownClusterForOrganization1, testdata.UserID)
	f := frisby.Create("Check the endpoint to return report for non-existing organization and non-existing cluster ID").Get(url)
	setAuthHeaderForOrganization(f, 100000)
	f.Send()
	f.ExpectStatus(404)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
	f.PrintReport()
}

// reproducerForIssue384 checks whether the issue https://github.com/RedHatInsights/insights-results-aggregator/issues/384 has been fixed
func reproducerForIssue384() {
	url := constructURLForReportForOrgCluster("000000000000000000000000000000000000", "1", testdata.UserID)
	f := frisby.Create("Reproducer for issue #384 (https://github.com/RedHatInsights/insights-results-aggregator/issues/384)").Get(url)
	setAuthHeader(f)
	f.Send()
	f.ExpectStatus(400)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)
	f.PrintReport()
}

// checkReportEndpointForImproperOrganization check if the endpoint to return report works as expected
func checkReportEndpointForImproperOrganization() {
	url := constructURLForReportForOrgCluster("foobar", knownClusterForOrganization1, testdata.UserID)
	f := frisby.Create("Check the endpoint to return report for improper organization").Get(url)
	setAuthHeader(f)
	f.Send()
	f.ExpectStatus(400)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	statusResponse := readStatusFromResponse(f)
	checkErrorStatusResponse(f, statusResponse)

	f.PrintReport()
}

// checkReportEndpointWrongMethods check if the endpoint to return results responds correctly to other methods than HTTP GET
func checkReportEndpointWrongMethods() {
	url := constructURLForReportForOrgCluster("1", knownClusterForOrganization1, testdata.UserID)
	checkGetEndpointByOtherMethods(url, false)
}

// checkReportEndpointForKnownOrganizationAndKnownClusterUnauthorizedCase check if the endpoint to return report works as expected
// This test variant does not sent authorization header
func checkReportEndpointForKnownOrganizationAndKnownClusterUnauthorizedCase() {
	url := constructURLForReportForOrgCluster("1", knownClusterForOrganization1, testdata.UserID)
	f := frisby.Create("Check the endpoint to return report for existing organization and cluster ID w/o authorization token").Get(url)
	f.Send()
	f.ExpectStatus(401)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	statusResponse := readStatusFromResponse(f)
	checkErrorStatusResponse(f, statusResponse)

	f.PrintReport()
}

// checkReportEndpointForKnownOrganizationAndUnknownClusterUnauthorizedCase check if the endpoint to return report works as expected
// This test variant does not sent authorization header
func checkReportEndpointForKnownOrganizationAndUnknownClusterUnauthorizedCase() {
	url := constructURLForReportForOrgCluster("1", unknownClusterForOrganization1, testdata.UserID)
	f := frisby.Create("Check the endpoint to return report for existing organization and non-existing cluster ID w/o authorization token").Get(url)
	f.Send()
	f.ExpectStatus(401)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	statusResponse := readStatusFromResponse(f)
	checkErrorStatusResponse(f, statusResponse)

	f.PrintReport()
}

// checkReportEndpointForUnknownOrganizationAndKnownClusterUnauthorizedCase check if the endpoint to return report works as expected
// This test variant does not sent authorization header
func checkReportEndpointForUnknownOrganizationAndKnownClusterUnauthorizedCase() {
	url := constructURLForReportForOrgCluster(unknownOrganizationID, knownClusterForOrganization1, testdata.UserID)
	f := frisby.Create("Check the endpoint to return report for non-existing organization and non-existing cluster ID w/o authorization token").Get(url)
	f.Send()
	f.ExpectStatus(401)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	statusResponse := readStatusFromResponse(f)
	checkErrorStatusResponse(f, statusResponse)

	f.PrintReport()
}

// checkReportEndpointForUnknownOrganizationAndUnknownClusterUnauthorizedCase check if the endpoint to return report works as expected
// This test variant does not sent authorization header
func checkReportEndpointForUnknownOrganizationAndUnknownClusterUnauthorizedCase() {
	url := constructURLForReportForOrgCluster(unknownOrganizationID, unknownClusterForOrganization1, testdata.UserID)
	f := frisby.Create("Check the endpoint to return report for non-existing organization and non-existing cluster ID w/o authorization token").Get(url)
	f.Send()
	f.ExpectStatus(401)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	statusResponse := readStatusFromResponse(f)
	checkErrorStatusResponse(f, statusResponse)

	f.PrintReport()
}

// checkReportEndpointForImproperOrganizationUnauthorizedCase check if the endpoint to return report works as expected
// This test variant does not sent authorization header
func checkReportEndpointForImproperOrganizationUnauthorizedCase() {
	url := constructURLForReportForOrgCluster("foobar", knownClusterForOrganization1, testdata.UserID)
	f := frisby.Create("Check the endpoint to return report for improper organization w/o authorization token").Get(url)
	f.Send()
	f.ExpectStatus(401)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	statusResponse := readStatusFromResponse(f)
	checkErrorStatusResponse(f, statusResponse)

	f.PrintReport()
}
