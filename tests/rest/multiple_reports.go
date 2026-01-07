/*
Copyright © 2020, 2021, 2022 Red Hat, Inc.

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
	"strings"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"
	"github.com/verdverm/frisby"

	"github.com/RedHatInsights/insights-results-aggregator/server"
	// "github.com/RedHatInsights/insights-results-aggregator/types"
)

// MultipleReportsResponse represents response from the server that contains
// results for multiple clusters together with overall status
type MultipleReportsResponse struct {
	Clusters    []string               `json:"clusters"`
	Errors      []string               `json:"errors"`
	Reports     map[string]interface{} `json:"reports"`
	GeneratedAt string                 `json:"generated_at"`
	Status      string                 `json:"status"`
}

// ClusterListInRequest represents request body containing list of clusters
type ClusterListInRequest struct {
	Clusters []string `json:"clusters"`
}

// constructURLForReportForOrgClusters function construct an URL to access the
// endpoint to retrieve results for given list of clusters
func constructURLForReportForOrgClusters(organizationID string, clusterIDs []string) string {
	cls := strings.Join(clusterIDs, ",")
	return httputils.MakeURLToEndpoint(apiURL, server.ReportForListOfClustersEndpoint, organizationID, cls)
}

// constructURLForReportForOrgClustersPostMethod function construct an URL to
// access the endpoint to retrieve results for given list of clusters using
// POST method
func constructURLForReportForOrgClustersPostMethod(organizationID string) string {
	return httputils.MakeURLToEndpoint(apiURL, server.ReportForListOfClustersPayloadEndpoint, organizationID)
}

// sendClusterListInPayload function sends the cluster list in request payload
// to server
func sendClusterListInPayload(f *frisby.Frisby, clusterList []string, addAuthHeader bool) {
	var payload = ClusterListInRequest{
		Clusters: clusterList,
	}
	// create payload
	f.SetJson(payload)

	// set auth header if required
	if addAuthHeader {
		setAuthHeader(f)
	}

	// and perform send
	f.Send()
}

// readMultipleReportsResponse reads and parses response body that should
// contains reports for multiple clusters
func readMultipleReportsResponse(f *frisby.Frisby) MultipleReportsResponse {
	response := MultipleReportsResponse{}

	// try to read response body
	text, err := f.Resp.Content()
	if err != nil {
		f.AddError(err.Error())
	} else {
		// try to deserialize response body
		err := json.Unmarshal(text, &response)
		if err != nil {
			f.AddError(err.Error())
		}
	}
	return response
}

// expectNumberOfClusters utility function checks if server response contains
// expected number of clusters
func expectNumberOfClusters(f *frisby.Frisby, response MultipleReportsResponse, expected int) {
	clusters := response.Clusters
	actual := len(clusters)
	if actual != expected {
		f.AddError(fmt.Sprintf("expected %d clusters in server response, but got %d instead", expected, actual))
	}
}

// expectNumberOfClusters utility function checks if server response contains
// expected number of errors
func expectNumberOfErrors(f *frisby.Frisby, response MultipleReportsResponse, expected int) {
	clusters := response.Errors
	actual := len(clusters)
	if actual != expected {
		f.AddError(fmt.Sprintf("expected %d errors in server response, but got %d instead", expected, actual))
	}
}

// expectNumberOfReports utility function checks if server response contains
// expected number of errors
func expectNumberOfReports(f *frisby.Frisby, response MultipleReportsResponse, expected int) {
	clusters := response.Reports
	actual := len(clusters)
	if actual != expected {
		f.AddError(fmt.Sprintf("expected %d reports in server response, but got %d instead", expected, actual))
	}
}

// expectOkStatus tests whether the response (JSON) contains status attribute set to 'ok'
func expectOkStatus(f *frisby.Frisby, response MultipleReportsResponse) {
	if response.Status != server.OkStatusPayload {
		f.AddError(fmt.Sprintf("Expected ok status, but got '%s' instead", response.Status))
	}
}

// expectClusterInResponse utility function checks if server response contains
// expected cluster name
func expectClusterInResponse(f *frisby.Frisby, response MultipleReportsResponse, clusterName string) {
	clusters := response.Clusters
	for _, cluster := range clusters {
		// cluster has been found
		if cluster == clusterName {
			return
		}
	}
	// cluster was not found
	f.AddError(fmt.Sprintf("Cluster %s can not be found in server response in the cluster list", clusterName))
}

// expectClusterInResponse utility function checks if server response contains
// expected report for specified cluster
func expectReportInResponse(f *frisby.Frisby, response MultipleReportsResponse, clusterName string) {
	reports := response.Reports
	for cluster := range reports {
		// cluster has been found
		if cluster == clusterName {
			return
		}
	}
	// report for cluster was not found
	f.AddError(fmt.Sprintf("Cluster %s can not be found in server response in reports map", clusterName))
}

// expectErrorClusterInResponse utility function checks if server response
// contains expected error
func expectErrorClusterInResponse(f *frisby.Frisby, response MultipleReportsResponse, clusterName string) {
	errors := response.Errors
	for _, cluster := range errors {
		// cluster has been found
		if cluster == clusterName {
			return
		}
	}
	// error for cluster was not found
	f.AddError(fmt.Sprintf("Cluster %s can not be found in server response in the errors list", clusterName))
}

// checkMultipleReportsForKnownOrganizationAnd1KnownCluster check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAnd1KnownCluster() {
	clusterList := []string{
		knownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClusters("1", clusterList)
	f := frisby.Create("Check the endpoint to return report for existing organization and one cluster ID").Get(url)
	setAuthHeader(f)
	f.Send()

	// check the response from server
	f.ExpectStatus(http.StatusOK)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	// check the payload returned from server
	response := readMultipleReportsResponse(f)
	expectNumberOfClusters(f, response, 1)
	expectNumberOfErrors(f, response, 0)
	expectNumberOfReports(f, response, 1)
	expectClusterInResponse(f, response, knownClusterForOrganization1)
	expectReportInResponse(f, response, knownClusterForOrganization1)
	expectOkStatus(f, response)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAnd2KnownClusters check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAnd2KnownClusters() {
	clusterList := []string{
		knownClusterForOrganization1,
		knownCluster2ForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClusters("1", clusterList)
	f := frisby.Create("Check the endpoint to return report for existing organization and two cluster IDs").Get(url)
	setAuthHeader(f)
	f.Send()

	// check the response from server
	f.ExpectStatus(http.StatusOK)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	// check the payload returned from server
	response := readMultipleReportsResponse(f)
	expectNumberOfClusters(f, response, 2)
	expectNumberOfErrors(f, response, 0)
	expectNumberOfReports(f, response, 2)
	expectClusterInResponse(f, response, knownClusterForOrganization1)
	expectClusterInResponse(f, response, knownCluster2ForOrganization1)
	expectReportInResponse(f, response, knownClusterForOrganization1)
	expectReportInResponse(f, response, knownCluster2ForOrganization1)
	expectOkStatus(f, response)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAnd3KnownClusters check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAnd3KnownClusters() {
	clusterList := []string{
		knownClusterForOrganization1,
		knownCluster2ForOrganization1,
		knownCluster3ForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClusters("1", clusterList)
	f := frisby.Create("Check the endpoint to return report for existing organization and three cluster IDs").Get(url)
	setAuthHeader(f)
	f.Send()

	// check the response from server
	f.ExpectStatus(http.StatusOK)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	// check the payload returned from server
	response := readMultipleReportsResponse(f)
	expectNumberOfClusters(f, response, 3)
	expectNumberOfErrors(f, response, 0)
	expectNumberOfReports(f, response, 3)
	expectClusterInResponse(f, response, knownClusterForOrganization1)
	expectClusterInResponse(f, response, knownCluster2ForOrganization1)
	expectClusterInResponse(f, response, knownCluster3ForOrganization1)
	expectReportInResponse(f, response, knownClusterForOrganization1)
	expectReportInResponse(f, response, knownCluster2ForOrganization1)
	expectReportInResponse(f, response, knownCluster3ForOrganization1)
	expectOkStatus(f, response)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAndUnknownCluster check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAndUnknownCluster() {
	clusterList := []string{
		unknownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClusters("1", clusterList)
	f := frisby.Create("Check the endpoint to return report for existing organization and one unknown cluster ID").Get(url)
	setAuthHeader(f)
	f.Send()

	// check the response from server
	f.ExpectStatus(http.StatusOK)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	// check the payload returned from server
	response := readMultipleReportsResponse(f)
	expectNumberOfClusters(f, response, 0)
	expectNumberOfErrors(f, response, 1)
	expectNumberOfReports(f, response, 0)
	expectErrorClusterInResponse(f, response, unknownClusterForOrganization1)
	expectOkStatus(f, response)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAndKnownAndUnknownCluster check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAndKnownAndUnknownCluster() {
	clusterList := []string{
		knownClusterForOrganization1,
		unknownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClusters("1", clusterList)
	f := frisby.Create("Check the endpoint to return report for existing organization and one known and one unknown cluste IDs").Get(url)
	setAuthHeader(f)
	f.Send()

	// check the response from server
	f.ExpectStatus(http.StatusOK)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	// check the payload returned from server
	response := readMultipleReportsResponse(f)
	expectNumberOfClusters(f, response, 1)
	expectNumberOfErrors(f, response, 1)
	expectNumberOfReports(f, response, 1)
	expectClusterInResponse(f, response, knownClusterForOrganization1)
	expectReportInResponse(f, response, knownClusterForOrganization1)
	expectErrorClusterInResponse(f, response, unknownClusterForOrganization1)
	expectOkStatus(f, response)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAnd2KnownAndUnknownCluster check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAnd2KnownAndUnknownCluster() {
	clusterList := []string{
		knownClusterForOrganization1,
		knownCluster2ForOrganization1,
		unknownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClusters("1", clusterList)
	f := frisby.Create("Check the endpoint to return report for existing organization and two knowns and one unknown cluster ID").Get(url)
	setAuthHeader(f)
	f.Send()

	// check the response from server
	f.ExpectStatus(http.StatusOK)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	// check the payload returned from server
	response := readMultipleReportsResponse(f)
	expectNumberOfClusters(f, response, 2)
	expectNumberOfErrors(f, response, 1)
	expectNumberOfReports(f, response, 2)
	expectClusterInResponse(f, response, knownClusterForOrganization1)
	expectClusterInResponse(f, response, knownCluster2ForOrganization1)
	expectReportInResponse(f, response, knownClusterForOrganization1)
	expectReportInResponse(f, response, knownCluster2ForOrganization1)
	expectErrorClusterInResponse(f, response, unknownClusterForOrganization1)
	expectOkStatus(f, response)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAnd3KnownAndUnknownCluster check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAnd3KnownAndUnknownCluster() {
	clusterList := []string{
		knownClusterForOrganization1,
		knownCluster2ForOrganization1,
		knownCluster3ForOrganization1,
		unknownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClusters("1", clusterList)
	f := frisby.Create("Check the endpoint to return report for existing organization and three knowns and one unknown cluster ID").Get(url)
	setAuthHeader(f)
	f.Send()

	// check the response from server
	f.ExpectStatus(http.StatusOK)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	// check the payload returned from server
	response := readMultipleReportsResponse(f)
	expectNumberOfClusters(f, response, 3)
	expectNumberOfErrors(f, response, 1)
	expectNumberOfReports(f, response, 3)
	expectClusterInResponse(f, response, knownClusterForOrganization1)
	expectClusterInResponse(f, response, knownCluster2ForOrganization1)
	expectClusterInResponse(f, response, knownCluster3ForOrganization1)
	expectReportInResponse(f, response, knownCluster2ForOrganization1)
	expectReportInResponse(f, response, knownCluster3ForOrganization1)
	expectReportInResponse(f, response, knownClusterForOrganization1)
	expectErrorClusterInResponse(f, response, unknownClusterForOrganization1)
	expectOkStatus(f, response)

	f.PrintReport()
}

// checkMultipleReportsForUnknownOrganizationAnd1KnownCluster check the endpoint that returns multiple results
func checkMultipleReportsForUnknownOrganizationAnd1KnownCluster() {
	clusterList := []string{
		knownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClusters(unknownOrganizationID, clusterList)
	f := frisby.Create("Check the endpoint to return report for unknown organization and known cluster ID").Get(url)
	setAuthHeader(f)
	f.Send()

	// check the response from server
	f.ExpectStatus(http.StatusBadRequest)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	f.PrintReport()
}

// checkMultipleReportsForUnknownOrganizationAndUnknownCluster check the endpoint that returns multiple results
func checkMultipleReportsForUnknownOrganizationAndUnknownCluster() {
	clusterList := []string{
		unknownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClusters(unknownOrganizationID, clusterList)
	f := frisby.Create("Check the endpoint to return report for unknown organization and unknown cluster ID").Get(url)
	setAuthHeader(f)
	f.Send()

	// check the response from server
	f.ExpectStatus(http.StatusOK)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	f.PrintReport()
}

// this is variant without authorization header
func checkMultipleReportsForKnownOrganizationAnd1KnownClusterUnauthorizedCase() {
	clusterList := []string{
		knownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClusters("1", clusterList)
	f := frisby.Create("Check the endpoint to return report for existing organization and one cluster ID without authorization").Get(url)
	f.Send()

	// check the response from server
	f.ExpectStatus(http.StatusUnauthorized)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAnd2KnownClusters check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAnd2KnownClustersUnauthorizedCase() {
	clusterList := []string{
		knownClusterForOrganization1,
		knownCluster2ForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClusters("1", clusterList)
	f := frisby.Create("Check the endpoint to return report for existing organization and two cluster IDs without authorization").Get(url)
	f.Send()

	// check the response from server
	f.ExpectStatus(http.StatusUnauthorized)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAnd3KnownClusters check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAnd3KnownClustersUnauthorizedCase() {
	clusterList := []string{
		knownClusterForOrganization1,
		knownCluster2ForOrganization1,
		knownCluster3ForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClusters("1", clusterList)
	f := frisby.Create("Check the endpoint to return report for existing organization and three cluster IDs without authorization").Get(url)
	f.Send()

	// check the response from server
	f.ExpectStatus(http.StatusUnauthorized)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAndUnknownCluster check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAndUnknownClusterUnauthorizedCase() {
	clusterList := []string{
		unknownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClusters("1", clusterList)
	f := frisby.Create("Check the endpoint to return report for existing organization and one unknown cluster ID without authorization").Get(url)
	f.Send()

	// check the response from server
	f.ExpectStatus(http.StatusUnauthorized)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAndKnownAndUnknownCluster check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAndKnownAndUnknownClusterUnauthorizedCase() {
	clusterList := []string{
		knownClusterForOrganization1,
		unknownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClusters("1", clusterList)
	f := frisby.Create("Check the endpoint to return report for existing organization and one known and one unknown cluste IDs without authorization").Get(url)
	f.Send()

	// check the response from server
	f.ExpectStatus(http.StatusUnauthorized)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAnd2KnownAndUnknownCluster check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAnd2KnownAndUnknownClusterUnauthorizedCase() {
	clusterList := []string{
		knownClusterForOrganization1,
		knownCluster2ForOrganization1,
		unknownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClusters("1", clusterList)
	f := frisby.Create("Check the endpoint to return report for existing organization and two knowns and one unknown cluster ID without authorization").Get(url)
	f.Send()

	// check the response from server
	f.ExpectStatus(http.StatusUnauthorized)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAnd3KnownAndUnknownCluster check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAnd3KnownAndUnknownClusterUnauthorizedCase() {
	clusterList := []string{
		knownClusterForOrganization1,
		knownCluster2ForOrganization1,
		knownCluster3ForOrganization1,
		unknownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClusters("1", clusterList)
	f := frisby.Create("Check the endpoint to return report for existing organization and three knowns and one unknown cluster ID without authorization").Get(url)
	f.Send()

	// check the response from server
	f.ExpectStatus(http.StatusUnauthorized)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	f.PrintReport()
}

// checkMultipleReportsForUnknownOrganizationAnd1KnownCluster check the endpoint that returns multiple results
func checkMultipleReportsForUnknownOrganizationAnd1KnownClusterUnauthorizedCase() {
	clusterList := []string{
		knownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClusters(unknownOrganizationID, clusterList)
	f := frisby.Create("Check the endpoint to return report for unknown organization and known cluster ID without authorization").Get(url)
	f.Send()

	// check the response from server
	f.ExpectStatus(http.StatusUnauthorized)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	f.PrintReport()
}

// checkMultipleReportsForUnknownOrganizationAndUnknownCluster check the endpoint that returns multiple results
func checkMultipleReportsForUnknownOrganizationAndUnknownClusterUnauthorizedCase() {
	clusterList := []string{
		unknownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClusters(unknownOrganizationID, clusterList)
	f := frisby.Create("Check the endpoint to return report for unknown organization and unknown cluster ID without authorization").Get(url)
	f.Send()

	// check the response from server
	f.ExpectStatus(http.StatusUnauthorized)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAnd1KnownClusterUsingPostMethod check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAnd1KnownClusterUsingPostMethod() {
	clusterList := []string{
		knownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClustersPostMethod("1")
	f := frisby.Create("Check the endpoint to return report for existing organization and one cluster ID (POST variant)").Post(url)
	sendClusterListInPayload(f, clusterList, true)

	// check the response from server
	f.ExpectStatus(http.StatusOK)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	// check the payload returned from server
	response := readMultipleReportsResponse(f)
	expectNumberOfClusters(f, response, 1)
	expectNumberOfErrors(f, response, 0)
	expectNumberOfReports(f, response, 1)
	expectClusterInResponse(f, response, knownClusterForOrganization1)
	expectReportInResponse(f, response, knownClusterForOrganization1)
	expectOkStatus(f, response)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAnd2KnownClustersUsingPostMethod check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAnd2KnownClustersUsingPostMethod() {
	clusterList := []string{
		knownClusterForOrganization1,
		knownCluster2ForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClustersPostMethod("1")
	f := frisby.Create("Check the endpoint to return report for existing organization and two cluster IDs (POST variant)").Post(url)
	sendClusterListInPayload(f, clusterList, true)

	// check the response from server
	f.ExpectStatus(http.StatusOK)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	// check the payload returned from server
	response := readMultipleReportsResponse(f)
	expectNumberOfClusters(f, response, 2)
	expectNumberOfErrors(f, response, 0)
	expectNumberOfReports(f, response, 2)
	expectClusterInResponse(f, response, knownClusterForOrganization1)
	expectClusterInResponse(f, response, knownCluster2ForOrganization1)
	expectReportInResponse(f, response, knownClusterForOrganization1)
	expectReportInResponse(f, response, knownCluster2ForOrganization1)
	expectOkStatus(f, response)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAnd3KnownClustersUsingPostMethod check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAnd3KnownClustersUsingPostMethod() {
	clusterList := []string{
		knownClusterForOrganization1,
		knownCluster2ForOrganization1,
		knownCluster3ForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClustersPostMethod("1")
	f := frisby.Create("Check the endpoint to return report for existing organization and three cluster IDs (POST variant)").Post(url)
	sendClusterListInPayload(f, clusterList, true)

	// check the response from server
	f.ExpectStatus(http.StatusOK)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	// check the payload returned from server
	response := readMultipleReportsResponse(f)
	expectNumberOfClusters(f, response, 3)
	expectNumberOfErrors(f, response, 0)
	expectNumberOfReports(f, response, 3)
	expectClusterInResponse(f, response, knownClusterForOrganization1)
	expectClusterInResponse(f, response, knownCluster2ForOrganization1)
	expectClusterInResponse(f, response, knownCluster3ForOrganization1)
	expectReportInResponse(f, response, knownClusterForOrganization1)
	expectReportInResponse(f, response, knownCluster2ForOrganization1)
	expectReportInResponse(f, response, knownCluster3ForOrganization1)
	expectOkStatus(f, response)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAndUnknownClusterUsingPostMethod check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAndUnknownClusterUsingPostMethod() {
	clusterList := []string{
		unknownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClustersPostMethod("1")
	f := frisby.Create("Check the endpoint to return report for existing organization and one unknown cluster ID (POST variant)").Post(url)
	sendClusterListInPayload(f, clusterList, true)

	// check the response from server
	f.ExpectStatus(http.StatusOK)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	// check the payload returned from server
	response := readMultipleReportsResponse(f)
	expectNumberOfClusters(f, response, 0)
	expectNumberOfErrors(f, response, 1)
	expectNumberOfReports(f, response, 0)
	expectErrorClusterInResponse(f, response, unknownClusterForOrganization1)
	expectOkStatus(f, response)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAndKnownAndUnknownClusterUsingPostMethod check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAndKnownAndUnknownClusterUsingPostMethod() {
	clusterList := []string{
		knownClusterForOrganization1,
		unknownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClustersPostMethod("1")
	f := frisby.Create("Check the endpoint to return report for existing organization and one known and one unknown cluster IDs (POST variant)").Post(url)
	sendClusterListInPayload(f, clusterList, true)

	// check the response from server
	f.ExpectStatus(http.StatusOK)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	// check the payload returned from server
	response := readMultipleReportsResponse(f)
	expectNumberOfClusters(f, response, 1)
	expectNumberOfErrors(f, response, 1)
	expectNumberOfReports(f, response, 1)
	expectClusterInResponse(f, response, knownClusterForOrganization1)
	expectReportInResponse(f, response, knownClusterForOrganization1)
	expectErrorClusterInResponse(f, response, unknownClusterForOrganization1)
	expectOkStatus(f, response)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAnd2KnownAndUnknownClusterUsingPostMethod check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAnd2KnownAndUnknownClusterUsingPostMethod() {
	clusterList := []string{
		knownClusterForOrganization1,
		knownCluster2ForOrganization1,
		unknownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClustersPostMethod("1")
	f := frisby.Create("Check the endpoint to return report for existing organization and two knowns and one unknown cluster ID (POST variant)").Post(url)
	sendClusterListInPayload(f, clusterList, true)

	// check the response from server
	f.ExpectStatus(http.StatusOK)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	// check the payload returned from server
	response := readMultipleReportsResponse(f)
	expectNumberOfClusters(f, response, 2)
	expectNumberOfErrors(f, response, 1)
	expectNumberOfReports(f, response, 2)
	expectClusterInResponse(f, response, knownClusterForOrganization1)
	expectClusterInResponse(f, response, knownCluster2ForOrganization1)
	expectReportInResponse(f, response, knownClusterForOrganization1)
	expectReportInResponse(f, response, knownCluster2ForOrganization1)
	expectErrorClusterInResponse(f, response, unknownClusterForOrganization1)
	expectOkStatus(f, response)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAnd3KnownAndUnknownClusterUsingPostMethod check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAnd3KnownAndUnknownClusterUsingPostMethod() {
	clusterList := []string{
		knownClusterForOrganization1,
		knownCluster2ForOrganization1,
		knownCluster3ForOrganization1,
		unknownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClustersPostMethod("1")
	f := frisby.Create("Check the endpoint to return report for existing organization and three knowns and one unknown cluster ID (POST variant)").Post(url)
	sendClusterListInPayload(f, clusterList, true)

	// check the response from server
	f.ExpectStatus(http.StatusOK)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	// check the payload returned from server
	response := readMultipleReportsResponse(f)
	expectNumberOfClusters(f, response, 3)
	expectNumberOfErrors(f, response, 1)
	expectNumberOfReports(f, response, 3)
	expectClusterInResponse(f, response, knownClusterForOrganization1)
	expectClusterInResponse(f, response, knownCluster2ForOrganization1)
	expectClusterInResponse(f, response, knownCluster3ForOrganization1)
	expectReportInResponse(f, response, knownCluster2ForOrganization1)
	expectReportInResponse(f, response, knownCluster3ForOrganization1)
	expectReportInResponse(f, response, knownClusterForOrganization1)
	expectErrorClusterInResponse(f, response, unknownClusterForOrganization1)
	expectOkStatus(f, response)

	f.PrintReport()
}

// checkMultipleReportsForUnknownOrganizationAnd1KnownClusterUsingPostMethod check the endpoint that returns multiple results
func checkMultipleReportsForUnknownOrganizationAnd1KnownClusterUsingPostMethod() {
	clusterList := []string{
		knownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClustersPostMethod(unknownClusterForOrganization1)
	f := frisby.Create("Check the endpoint to return report for unknown organization and known cluster ID (POST variant)").Post(url)
	sendClusterListInPayload(f, clusterList, true)

	// check the response from server
	f.ExpectStatus(http.StatusBadRequest)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	f.PrintReport()
}

// checkMultipleReportsForUnknownOrganizationAndUnknownClusterUsingPostMethod check the endpoint that returns multiple results
func checkMultipleReportsForUnknownOrganizationAndUnknownClusterUsingPostMethod() {
	clusterList := []string{
		unknownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClustersPostMethod(unknownClusterForOrganization1)
	f := frisby.Create("Check the endpoint to return report for unknown organization and unknown cluster ID (POST variant)").Post(url)
	sendClusterListInPayload(f, clusterList, true)

	// check the response from server
	f.ExpectStatus(http.StatusBadRequest)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAnd1KnownClusterUnauthorizedCaseUsingPostMethod check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAnd1KnownClusterUnauthorizedCaseUsingPostMethod() {
	clusterList := []string{
		knownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClustersPostMethod("1")
	f := frisby.Create("Check the endpoint to return report for existing organization and one cluster ID (POST variant)").Post(url)
	sendClusterListInPayload(f, clusterList, false)

	// check the response from server
	f.ExpectStatus(http.StatusUnauthorized)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAnd2KnownClustersUnauthorizedCaseUsingPostMethod check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAnd2KnownClustersUnauthorizedCaseUsingPostMethod() {
	clusterList := []string{
		knownClusterForOrganization1,
		knownCluster2ForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClustersPostMethod("1")
	f := frisby.Create("Check the endpoint to return report for existing organization and two cluster IDs (POST variant)").Post(url)
	sendClusterListInPayload(f, clusterList, false)

	// check the response from server
	f.ExpectStatus(http.StatusUnauthorized)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAnd3KnownClustersUnauthorizedCaseUsingPostMethod check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAnd3KnownClustersUnauthorizedCaseUsingPostMethod() {
	clusterList := []string{
		knownClusterForOrganization1,
		knownCluster2ForOrganization1,
		knownCluster3ForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClustersPostMethod("1")
	f := frisby.Create("Check the endpoint to return report for existing organization and three cluster IDs (POST variant)").Post(url)
	sendClusterListInPayload(f, clusterList, false)

	// check the response from server
	f.ExpectStatus(http.StatusUnauthorized)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAndUnknownClusterUnauthorizedCaseUsingPostMethod check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAndUnknownClusterUnauthorizedCaseUsingPostMethod() {
	clusterList := []string{
		unknownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClustersPostMethod("1")
	f := frisby.Create("Check the endpoint to return report for existing organization and one unknown cluster ID (POST variant)").Post(url)
	sendClusterListInPayload(f, clusterList, false)

	// check the response from server
	f.ExpectStatus(http.StatusUnauthorized)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAndKnownAndUnknownClusterUnauthorizedCaseUsingPostMethod check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAndKnownAndUnknownClusterUnauthorizedCaseUsingPostMethod() {
	clusterList := []string{
		knownClusterForOrganization1,
		unknownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClustersPostMethod("1")
	f := frisby.Create("Check the endpoint to return report for existing organization and one known and one unknown cluster IDs (POST variant)").Post(url)
	sendClusterListInPayload(f, clusterList, false)

	// check the response from server
	f.ExpectStatus(http.StatusUnauthorized)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAnd2KnownAndUnknownClusterUnauthorizedCaseUsingPostMethod check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAnd2KnownAndUnknownClusterUnauthorizedCaseUsingPostMethod() {
	clusterList := []string{
		knownClusterForOrganization1,
		knownCluster2ForOrganization1,
		unknownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClustersPostMethod("1")
	f := frisby.Create("Check the endpoint to return report for existing organization and two knowns and one unknown cluster ID (POST variant)").Post(url)
	sendClusterListInPayload(f, clusterList, false)

	// check the response from server
	f.ExpectStatus(http.StatusUnauthorized)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	f.PrintReport()
}

// checkMultipleReportsForKnownOrganizationAnd3KnownAndUnknownClusterUnauthorizedCaseUsingPostMethod check the endpoint that returns multiple results
func checkMultipleReportsForKnownOrganizationAnd3KnownAndUnknownClusterUnauthorizedCaseUsingPostMethod() {
	clusterList := []string{
		knownClusterForOrganization1,
		knownCluster2ForOrganization1,
		knownCluster3ForOrganization1,
		unknownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClustersPostMethod("1")
	f := frisby.Create("Check the endpoint to return report for existing organization and three knowns and one unknown cluster ID (POST variant)").Post(url)
	sendClusterListInPayload(f, clusterList, false)

	// check the response from server
	f.ExpectStatus(http.StatusUnauthorized)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	f.PrintReport()
}

// checkMultipleReportsForUnknownOrganizationAnd1KnownClusterUnauthorizedCaseUsingPostMethod check the endpoint that returns multiple results
func checkMultipleReportsForUnknownOrganizationAnd1KnownClusterUnauthorizedCaseUsingPostMethod() {
	clusterList := []string{
		knownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClustersPostMethod(unknownClusterForOrganization1)
	f := frisby.Create("Check the endpoint to return report for unknown organization and known cluster ID (POST variant)").Post(url)
	sendClusterListInPayload(f, clusterList, false)

	// check the response from server
	f.ExpectStatus(http.StatusUnauthorized)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	f.PrintReport()
}

// checkMultipleReportsForUnknownOrganizationAndUnknownClusterUnauthorizedCaseUsingPostMethod check the endpoint that returns multiple results
func checkMultipleReportsForUnknownOrganizationAndUnknownClusterUnauthorizedCaseUsingPostMethod() {
	clusterList := []string{
		unknownClusterForOrganization1,
	}

	// send request to the endpoint
	url := constructURLForReportForOrgClustersPostMethod(unknownClusterForOrganization1)
	f := frisby.Create("Check the endpoint to return report for unknown organization and unknown cluster ID (POST variant)").Post(url)
	sendClusterListInPayload(f, clusterList, false)

	// check the response from server
	f.ExpectStatus(http.StatusUnauthorized)
	f.ExpectHeader(contentTypeHeader, ContentTypeJSON)

	f.PrintReport()
}
