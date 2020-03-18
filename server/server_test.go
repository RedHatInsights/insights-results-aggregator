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
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/RedHatInsights/insights-results-aggregator/types"

	"github.com/RedHatInsights/insights-results-aggregator/storage"

	"github.com/gorilla/mux"

	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/stretchr/testify/assert"
)

var config = server.Configuration{
	Address:     ":8080",
	APIPrefix:   "/api/test/",
	APISpecFile: "openapi.json",
	Debug:       true,
}

const (
	testOrgID          = types.OrgID(1)
	testClusterName    = types.ClusterName("412701a1-c036-490a-9173-a3428c25b677")
	testRuleID         = types.RuleID("1")
	testUserID         = types.UserID("1")
	testBadClusterName = types.ClusterName("aaaa")
)

func executeRequest(testServer *server.HTTPServer, req *http.Request) *httptest.ResponseRecorder {
	router := testServer.Initialize(config.Address)

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	return rr
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}

func checkResponseBody(t *testing.T, expected string, body io.ReadCloser) {
	result, err := ioutil.ReadAll(body)
	if err != nil {
		t.Fatal(err)
	}

	expected = strings.TrimSpace(expected)
	resultStr := strings.TrimSpace(string(result))

	assert.Equal(t, expected, resultStr)
}

func TestReadReportForClusterMissingOrgIdAndClusterName(t *testing.T) {
	testServer := server.New(config, nil)

	req, err := http.NewRequest("GET", config.APIPrefix+"report/", nil)
	if err != nil {
		t.Fatal(err)
	}

	response := executeRequest(testServer, req).Result()
	checkResponseCode(t, http.StatusNotFound, response.StatusCode)
}

func TestReadReportForClusterMissingClusterName(t *testing.T) {
	testServer := server.New(config, nil)

	req, err := http.NewRequest("GET", config.APIPrefix+"report/12345", nil)
	if err != nil {
		t.Fatal(err)
	}

	response := executeRequest(testServer, req).Result()
	checkResponseCode(t, http.StatusNotFound, response.StatusCode)
}

func TestReadReportForClusterNonIntOrgID(t *testing.T) {
	testServer := server.New(config, nil)

	req, err := http.NewRequest("GET", config.APIPrefix+"report/bad_org_id/cluster_name", nil)
	if err != nil {
		t.Fatal(err)
	}

	response := executeRequest(testServer, req).Result()
	checkResponseCode(t, http.StatusBadRequest, response.StatusCode)
}

func TestReadReportForClusterNegativeOrgID(t *testing.T) {
	testServer := server.New(config, nil)

	req, err := http.NewRequest("GET", config.APIPrefix+"report/-1/"+string(testClusterName), nil)
	if err != nil {
		t.Fatal(err)
	}

	response := executeRequest(testServer, req).Result()
	checkResponseCode(t, http.StatusBadRequest, response.StatusCode)
}

func TestReadReportForClusterBadClusterName(t *testing.T) {
	testServer := server.New(config, nil)

	req, err := http.NewRequest("GET", config.APIPrefix+"report/12345/"+string(testBadClusterName), nil)
	if err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatal(err)
	}

	testServer := server.New(config, mockStorage)

	req, err := http.NewRequest(
		"GET",
		config.APIPrefix+"report/"+fmt.Sprint(testOrgID)+"/"+string(testClusterName),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatal(err)
	}

	response := executeRequest(testServer, req).Result()
	checkResponseCode(t, http.StatusInternalServerError, response.StatusCode)
	checkResponseBody(t, `{"status":"sql: database is closed"}`, response.Body)
}

func TestListOfClustersForNonExistingOrganization(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	testServer := server.New(config, mockStorage)

	req, err := http.NewRequest("GET", config.APIPrefix+"organizations/1/clusters", nil)
	if err != nil {
		t.Fatal(err)
	}

	response := executeRequest(testServer, req).Result()
	checkResponseCode(t, http.StatusOK, response.StatusCode)
	checkResponseBody(t, `{"clusters":[],"status":"ok"}`, response.Body)
}

func TestListOfClustersForOrganizationOK(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.WriteReportForCluster(testOrgID, testClusterName, "{}", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	testServer := server.New(config, mockStorage)

	req, err := http.NewRequest("GET", config.APIPrefix+"organizations/1/clusters", nil)
	if err != nil {
		t.Fatal(err)
	}

	response := executeRequest(testServer, req).Result()
	checkResponseCode(t, http.StatusOK, response.StatusCode)
	checkResponseBody(t, `{"clusters":["`+string(testClusterName)+`"],"status":"ok"}`, response.Body)
}

// TestListOfClustersForOrganizationDBError expects db error
// because the storage is closed before the query
func TestListOfClustersForOrganizationDBError(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	helpers.MustCloseStorage(t, mockStorage)

	testServer := server.New(config, mockStorage)

	req, err := http.NewRequest("GET", config.APIPrefix+"organizations/1/clusters", nil)
	if err != nil {
		t.Fatal(err)
	}

	response := executeRequest(testServer, req).Result()
	checkResponseCode(t, http.StatusInternalServerError, response.StatusCode)
}

func TestListOfClustersForOrganizationNegativeID(t *testing.T) {
	testServer := server.New(config, nil)

	req, err := http.NewRequest("GET", config.APIPrefix+"organizations/-1/clusters", nil)
	if err != nil {
		t.Fatal(err)
	}

	response := executeRequest(testServer, req).Result()
	checkResponseCode(t, http.StatusBadRequest, response.StatusCode)
}

func TestListOfClustersForOrganizationNonIntID(t *testing.T) {
	testServer := server.New(config, nil)

	req, err := http.NewRequest("GET", config.APIPrefix+"organizations/nonint/clusters", nil)
	if err != nil {
		t.Fatal(err)
	}

	response := executeRequest(testServer, req).Result()
	checkResponseCode(t, http.StatusBadRequest, response.StatusCode)
}

func TestMainEndpoint(t *testing.T) {
	testServer := server.New(config, nil)

	req, err := http.NewRequest("GET", config.APIPrefix, nil)
	if err != nil {
		t.Fatal(err)
	}

	response := executeRequest(testServer, req).Result()
	checkResponseCode(t, http.StatusOK, response.StatusCode)
}

func TestListOfOrganizationsEmpty(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	testServer := server.New(config, mockStorage)

	req, err := http.NewRequest("GET", config.APIPrefix+"organizations", nil)
	if err != nil {
		t.Fatal(err)
	}

	response := executeRequest(testServer, req).Result()
	checkResponseCode(t, http.StatusOK, response.StatusCode)
	checkResponseBody(t, `{"organizations":[],"status":"ok"}`, response.Body)
}

func TestListOfOrganizationsOK(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.WriteReportForCluster(1, "8083c377-8a05-4922-af8d-e7d0970c1f49", "{}", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	err = mockStorage.WriteReportForCluster(5, "52ab955f-b769-444d-8170-4b676c5d3c85", "{}", time.Now())
	if err != nil {
		t.Fatal(err)
	}

	testServer := server.New(config, mockStorage)

	req, err := http.NewRequest("GET", config.APIPrefix+"organizations", nil)
	if err != nil {
		t.Fatal(err)
	}

	response := executeRequest(testServer, req).Result()
	checkResponseCode(t, http.StatusOK, response.StatusCode)
	checkResponseBody(t, `{"organizations":[1,5],"status":"ok"}`, response.Body)
}

func TestListOfOrganizationsDBError(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	helpers.MustCloseStorage(t, mockStorage)

	testServer := server.New(config, mockStorage)

	req, err := http.NewRequest("GET", config.APIPrefix+"organizations", nil)
	if err != nil {
		t.Fatal(err)
	}

	response := executeRequest(testServer, req).Result()
	checkResponseCode(t, http.StatusInternalServerError, response.StatusCode)
}

func TestServerStartError(t *testing.T) {
	testServer := server.New(server.Configuration{
		Address:   "localhost:99999",
		APIPrefix: "",
	}, nil)

	err := testServer.Start()
	if err == nil {
		t.Fatal(fmt.Errorf("should return an error"))
	}
}

func TestGetRouterIntParamMissing(t *testing.T) {
	request, err := http.NewRequest("GET", "organizations//clusters", nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = server.GetRouterPositiveIntParam(request, "test")
	if err == nil {
		t.Fatal("Param should be missing")
	}

	assert.Equal(t, "missing param test", err.Error())
}

func TestReadClusterNameMissing(t *testing.T) {
	request, err := http.NewRequest("GET", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = server.ReadClusterName(httptest.NewRecorder(), request)
	if err == nil {
		t.Fatal("Param should be missing")
	}

	assert.Equal(t, "missing param cluster", err.Error())
}

func TestReadOrganizationIDMissing(t *testing.T) {
	request, err := http.NewRequest("GET", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	_, err = server.ReadOrganizationID(httptest.NewRecorder(), request)
	if err == nil {
		t.Fatal("Param should be missing")
	}

	assert.Equal(t, "missing param organization", err.Error())
}

func mustGetRequestWithMuxVars(
	t *testing.T,
	method string,
	url string,
	body io.Reader,
	vars map[string]string,
) *http.Request {
	request, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatal(err)
	}

	request = mux.SetURLVars(request, vars)

	return request
}

func TestGetRouterIntParamNonIntError(t *testing.T) {
	request := mustGetRequestWithMuxVars(t, "GET", "", nil, map[string]string{
		"id": "non int",
	})

	_, err := server.GetRouterPositiveIntParam(request, "id")

	if err == nil {
		t.Fatalf("Error expected, got %v", err)
	}

	assert.Contains(t, err.Error(), "integer expected")
}

func TestGetRouterIntParamOK(t *testing.T) {
	request := mustGetRequestWithMuxVars(t, "GET", "", nil, map[string]string{
		"id": "99",
	})

	id, err := server.GetRouterPositiveIntParam(request, "id")

	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, uint64(99), id)
}

func TestGetRouterPositiveIntParamZeroError(t *testing.T) {
	request := mustGetRequestWithMuxVars(t, "GET", "", nil, map[string]string{
		"id": "0",
	})

	_, err := server.GetRouterPositiveIntParam(request, "id")

	if err == nil {
		t.Fatalf("Error expected, got %v", err)
	}

	assert.Contains(t, err.Error(), "positive value expected")
}

func TestServerStart(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		s := server.New(server.Configuration{
			// will use any free port
			Address:   ":0",
			APIPrefix: config.APIPrefix,
		}, nil)

		go func() {
			for {
				if s.Serv != nil {
					break
				}

				time.Sleep(500 * time.Millisecond)
			}

			// doing some request to be sure server started successfully
			req, err := http.NewRequest("GET", config.APIPrefix, nil)
			if err != nil {
				panic(err)
			}

			response := executeRequest(s, req).Result()
			checkResponseCode(t, http.StatusForbidden, response.StatusCode)

			// stopping the server
			err = s.Stop(context.Background())
			if err != nil {
				panic(err)
			}
		}()

		err := s.Start()
		if err != nil && err != http.ErrServerClosed {
			panic(err)
		}
	}, 5*time.Second)
}

func TestServeAPISpecFileOK(t *testing.T) {
	err := os.Chdir("../")
	if err != nil {
		t.Fatal(err)
	}

	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	testServer := server.New(config, mockStorage)

	req, err := http.NewRequest(
		"GET",
		config.APIPrefix+config.APISpecFile,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	response := executeRequest(testServer, req).Result()

	checkResponseCode(t, http.StatusOK, response.StatusCode)

	fileData, err := ioutil.ReadFile(config.APISpecFile)
	if err != nil {
		t.Fatal(err)
	}

	checkResponseBody(t, string(fileData), response.Body)
}

func TestServeAPISpecFileError(t *testing.T) {
	dirName, err := ioutil.TempDir("/tmp/", "")
	if err != nil {
		t.Fatal(err)
	}

	err = os.Chdir(dirName)
	if err != nil {
		t.Fatal(err)
	}

	err = os.Remove(dirName)
	if err != nil {
		t.Fatal(err)
	}

	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	testServer := server.New(config, mockStorage)

	req, err := http.NewRequest(
		"GET",
		config.APIPrefix+config.APISpecFile,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}

	response := executeRequest(testServer, req).Result()

	checkResponseCode(t, http.StatusInternalServerError, response.StatusCode)
}

func TestRuleFeedbackVote(t *testing.T) {
	for _, action := range []string{
		"like", "dislike", "reset_vote",
	} {
		var expectedVote storage.UserVote

		switch action {
		case "like":
			expectedVote = storage.UserVoteLike
		case "dislike":
			expectedVote = storage.UserVoteDislike
		case "reset_vote":
			expectedVote = storage.UserVoteNone
		default:
			t.Fatal("not expected action")
		}

		func(action string) {
			mockStorage := helpers.MustGetMockStorage(t, true)
			defer helpers.MustCloseStorage(t, mockStorage)

			testServer := server.New(config, mockStorage)

			url := fmt.Sprintf("%vclusters/%v/rules/%v/%v", config.APIPrefix, testClusterName, testRuleID, action)
			req, err := http.NewRequest("PUT", url, nil)
			helpers.FailOnError(t, err)

			// authorize user
			identity := server.Identity{
				AccountNumber: testUserID,
			}
			req = req.WithContext(context.WithValue(req.Context(), server.ContextKeyUser, identity))

			response := executeRequest(testServer, req).Result()

			checkResponseCode(t, http.StatusOK, response.StatusCode)
			checkResponseBody(t, `{"status":"ok"}`, response.Body)

			feedback, err := mockStorage.GetUserFeedbackOnRule(testClusterName, testRuleID, testUserID)
			helpers.FailOnError(t, err)

			assert.Equal(t, testClusterName, feedback.ClusterID)
			assert.Equal(t, testRuleID, feedback.RuleID)
			assert.Equal(t, testUserID, feedback.UserID)
			assert.Equal(t, "", feedback.Message)
			assert.Equal(t, expectedVote, feedback.UserVote)
		}(action)
	}
}

func TestRuleFeedbackErrorBadClusterName(t *testing.T) {
	testServer := server.New(config, nil)

	url := fmt.Sprintf("%vclusters/%v/rules/%v/like", config.APIPrefix, testBadClusterName, testRuleID)
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		t.Fatal(err)
	}

	response := executeRequest(testServer, req).Result()
	checkResponseCode(t, http.StatusInternalServerError, response.StatusCode)
}

func TestRuleFeedbackErrorClosedStorage(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	helpers.MustCloseStorage(t, mockStorage)

	testServer := server.New(config, mockStorage)

	url := fmt.Sprintf("%vclusters/%v/rules/%v/like", config.APIPrefix, testClusterName, testRuleID)
	req, err := http.NewRequest("PUT", url, nil)
	helpers.FailOnError(t, err)

	// authorize user
	identity := server.Identity{
		AccountNumber: testUserID,
	}
	req = req.WithContext(context.WithValue(req.Context(), server.ContextKeyUser, identity))

	response := executeRequest(testServer, req).Result()

	checkResponseCode(t, http.StatusInternalServerError, response.StatusCode)
}

func TestDeleteReportsByOrganization(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)
	testServer := server.New(config, mockStorage)

	req, err := http.NewRequest("DELETE", config.APIPrefix+"organizations/1", nil)
	helpers.FailOnError(t, err)

	resp := executeRequest(testServer, req)

	checkResponseCode(t, http.StatusOK, resp.Result().StatusCode)
}

func TestDeleteReportsByClusterName(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)
	testServer := server.New(config, mockStorage)

	req, err := http.NewRequest("DELETE", config.APIPrefix+"clusters/2d615e74-29f8-4bfb-8269-908f1c1b1bb4", nil)
	helpers.FailOnError(t, err)

	resp := executeRequest(testServer, req)

	checkResponseCode(t, http.StatusOK, resp.Result().StatusCode)
}
