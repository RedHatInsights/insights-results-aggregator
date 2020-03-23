// Copyright 2020 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/tests/testdata"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

func TestReadReportForClusterMissingOrgIdAndClusterName(t *testing.T) {
	testServer := server.New(config, nil)

	req, err := http.NewRequest("GET", config.APIPrefix+"report/", nil)
	helpers.FailOnError(t, err)

	response := executeRequest(testServer, req).Result()
	checkResponseCode(t, http.StatusNotFound, response.StatusCode)
}

func TestReadReportForClusterMissingClusterName(t *testing.T) {
	testServer := server.New(config, nil)

	req, err := http.NewRequest("GET", config.APIPrefix+"report/12345", nil)
	helpers.FailOnError(t, err)

	response := executeRequest(testServer, req).Result()
	checkResponseCode(t, http.StatusNotFound, response.StatusCode)
}

func TestReadReportForClusterNonIntOrgID(t *testing.T) {
	testServer := server.New(config, nil)

	req, err := http.NewRequest("GET", config.APIPrefix+"report/bad_org_id/cluster_name", nil)
	helpers.FailOnError(t, err)

	response := executeRequest(testServer, req).Result()
	checkResponseCode(t, http.StatusBadRequest, response.StatusCode)
}

func TestReadReportForClusterNegativeOrgID(t *testing.T) {
	testServer := server.New(config, nil)

	req, err := http.NewRequest("GET", config.APIPrefix+"report/-1/"+string(testClusterName), nil)
	helpers.FailOnError(t, err)

	response := executeRequest(testServer, req).Result()
	checkResponseCode(t, http.StatusBadRequest, response.StatusCode)
}

func TestReadReportForClusterBadClusterName(t *testing.T) {
	testServer := server.New(config, nil)

	req, err := http.NewRequest("GET", config.APIPrefix+"report/12345/"+string(testBadClusterName), nil)
	helpers.FailOnError(t, err)

	response := executeRequest(testServer, req).Result()
	checkResponseCode(t, http.StatusInternalServerError, response.StatusCode)
}

func TestReadNonExistingReport(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	testServer := server.New(config, mockStorage)

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%v%v/%v", config.APIPrefix, "report/1", testClusterName),
		nil,
	)
	helpers.FailOnError(t, err)

	response := executeRequest(testServer, req).Result()
	checkResponseCode(t, http.StatusNotFound, response.StatusCode)
	checkResponseBody(
		t,
		fmt.Sprintf(`{"status":"Item with ID 1/%v was not found in the storage"}`, testClusterName),
		response.Body,
	)
}

func TestReadExistingReport(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.WriteReportForCluster(testOrgID, testClusterName, "{}", time.Now())
	helpers.FailOnError(t, err)

	testServer := server.New(config, mockStorage)

	req, err := http.NewRequest(
		"GET",
		config.APIPrefix+"report/"+fmt.Sprint(testOrgID)+"/"+string(testClusterName),
		nil,
	)
	helpers.FailOnError(t, err)

	response := executeRequest(testServer, req).Result()
	checkResponseCode(t, http.StatusOK, response.StatusCode)
}

func TestReadReportDBError(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	helpers.MustCloseStorage(t, mockStorage)

	testServer := server.New(config, mockStorage)

	req, err := http.NewRequest(
		"GET",
		config.APIPrefix+"report/1/2d615e74-29f8-4bfb-8269-908f1c1b1bb4",
		nil,
	)
	helpers.FailOnError(t, err)

	response := executeRequest(testServer, req).Result()
	checkResponseCode(t, http.StatusInternalServerError, response.StatusCode)
	checkResponseBody(t, `{"status":"sql: database is closed"}`, response.Body)
}

func assertReportResponsesEqual(t *testing.T, expected, got string) {
	var expectedResponse, gotResponse struct {
		Status string               `json:"status"`
		Report types.ReportResponse `json:"report"`
	}

	err := helpers.JSONUnmarshalStrict([]byte(expected), &expectedResponse)
	helpers.FailOnError(t, err)
	err = helpers.JSONUnmarshalStrict([]byte(got), &gotResponse)
	helpers.FailOnError(t, err)

	assert.NotEmpty(
		t,
		expectedResponse.Status,
		"status is empty(probably json is completely wrong and unmarshal didn't do anything useful)",
	)
	assert.Equal(t, expectedResponse.Status, gotResponse.Status)
	assert.Equal(t, expectedResponse.Report.Meta, gotResponse.Report.Meta)
	// ignore the order
	assert.ElementsMatch(t, expectedResponse.Report.Rules, gotResponse.Report.Rules)
}

func TestReadReportWithContent(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)
	dbStorage := mockStorage.(*storage.DBStorage)

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.Report3Rules,
		testdata.LastCheckedAt,
	)
	helpers.FailOnError(t, err)

	// write some rule content into the DB
	err = dbStorage.LoadRuleContent(testdata.RuleContent3Rules)
	helpers.FailOnError(t, err)

	testServer := server.New(config, mockStorage)

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf(
			"%v%v/%v/%v",
			config.APIPrefix, "report", testdata.OrgID, testdata.ClusterName,
		),
		nil,
	)
	helpers.FailOnError(t, err)

	response := executeRequest(testServer, req).Result()
	checkResponseCode(t, http.StatusOK, response.StatusCode)

	body, err := ioutil.ReadAll(response.Body)
	helpers.FailOnError(t, err)

	assertReportResponsesEqual(t, testdata.Report3RulesExpectedResponse, string(body))
}
