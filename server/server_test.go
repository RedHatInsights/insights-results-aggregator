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

package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/RedHatInsights/insights-results-aggregator/server"
)

var config = server.Configuration{
	Address:   ":8080",
	APIPrefix: "/api/test/",
	Debug:     true,
}

func executeRequest(server *server.HTTPServer, req *http.Request) *httptest.ResponseRecorder {
	router := server.Initialize(config.Address)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	return rr
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}

func TestReadReportForClusterMissingOrgIdAndClusterName(t *testing.T) {
	server := server.New(config, nil)
	req, err := http.NewRequest("GET", config.APIPrefix+"report/", nil)
	if err != nil {
		t.Fatal(err)
	}

	response := executeRequest(server, req).Result()
	checkResponseCode(t, http.StatusNotFound, response.StatusCode)
}

func TestReadReportForClusterMissingClusterName(t *testing.T) {
	server := server.New(config, nil)
	req, err := http.NewRequest("GET", config.APIPrefix+"report/12345", nil)
	if err != nil {
		t.Fatal(err)
	}

	response := executeRequest(server, req).Result()
	checkResponseCode(t, http.StatusNotFound, response.StatusCode)
}

func TestReadReportForClusterBadOrgID(t *testing.T) {
	server := server.New(config, nil)
	req, err := http.NewRequest("GET", config.APIPrefix+"report/bad_org_id", nil)
	if err != nil {
		t.Fatal(err)
	}

	response := executeRequest(server, req).Result()
	// It responses with NotFound because it cannot be stored in the DB
	checkResponseCode(t, http.StatusNotFound, response.StatusCode)
}

func TestReadReportForClusterBadClusterName(t *testing.T) {
	server := server.New(config, nil)
	req, err := http.NewRequest("GET", config.APIPrefix+"report/12345/aaaa", nil)
	if err != nil {
		t.Fatal(err)
	}

	response := executeRequest(server, req).Result()
	checkResponseCode(t, http.StatusInternalServerError, response.StatusCode)
}
