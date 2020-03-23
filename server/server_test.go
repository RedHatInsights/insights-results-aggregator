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
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/RedHatInsights/insights-results-aggregator/storage"

	"github.com/RedHatInsights/insights-results-aggregator/tests/testdata"

	"github.com/RedHatInsights/insights-results-aggregator/types"

	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/stretchr/testify/assert"
)

var config = server.Configuration{
	Address:     ":8080",
	APIPrefix:   "/api/test/",
	APISpecFile: "openapi.json",
	Debug:       true,
	Auth:        false,
}

// TODO: remove these consts
const (
	testOrgID          = types.OrgID(1)
	testClusterName    = types.ClusterName("412701a1-c036-490a-9173-a3428c25b677")
	testRuleID         = types.RuleID("1")
	testUserID         = types.UserID("1")
	testBadClusterName = types.ClusterName("aaaa")
)

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}

// checkResponseBody checks if body is the same string as expected,
func checkResponseBody(t *testing.T, expected string, body io.ReadCloser) {
	result, err := ioutil.ReadAll(body)
	helpers.FailOnError(t, err)

	expected = strings.TrimSpace(expected)
	resultStr := strings.TrimSpace(string(result))

	assert.Equal(t, expected, resultStr)
}

func TestMakeURLToEndpoint(t *testing.T) {
	assert.Equal(
		t,
		"api/prefix/report/-55/cluster_id",
		server.MakeURLToEndpoint("api/prefix/", server.ReportEndpoint, -55, "cluster_id"),
	)
}

func TestListOfClustersForNonExistingOrganization(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ClustersForOrganizationEndpoint,
		EndpointArgs: []interface{}{1},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"clusters":[],"status":"ok"}`,
	})
}

func TestListOfClustersForOrganizationOK(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.LastCheckedAt,
	)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ClustersForOrganizationEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"clusters":["` + string(testdata.ClusterName) + `"],"status":"ok"}`,
	})
}

// TestListOfClustersForOrganizationDBError expects db error
// because the storage is closed before the query
func TestListOfClustersForOrganizationDBError(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	helpers.MustCloseStorage(t, mockStorage)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ClustersForOrganizationEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       `{"status": "sql: database is closed"}`,
	})
}

func TestListOfClustersForOrganizationNegativeID(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ClustersForOrganizationEndpoint,
		EndpointArgs: []interface{}{-1},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body: `{
			"status": "Error during parsing param organization with value -1. Error: unsigned integer expected"
		}`,
	})
}

func TestListOfClustersForOrganizationNonIntID(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.ClustersForOrganizationEndpoint,
		EndpointArgs: []interface{}{"non-int"},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body: `{
			"status": "Error during parsing param organization with value non-int. Error: unsigned integer expected"
		}`,
	})
}

func TestMainEndpoint(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:   http.MethodGet,
		Endpoint: server.MainEndpoint,
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"status": "ok"}`,
	})
}

func TestListOfOrganizationsEmpty(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:   http.MethodGet,
		Endpoint: server.OrganizationsEndpoint,
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"organizations":[],"status":"ok"}`,
	})
}

func TestListOfOrganizationsOK(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.WriteReportForCluster(1, "8083c377-8a05-4922-af8d-e7d0970c1f49", "{}", time.Now())
	helpers.FailOnError(t, err)

	err = mockStorage.WriteReportForCluster(5, "52ab955f-b769-444d-8170-4b676c5d3c85", "{}", time.Now())
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:   http.MethodGet,
		Endpoint: server.OrganizationsEndpoint,
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"organizations":[1, 5],"status":"ok"}`,
	})
}

func TestListOfOrganizationsDBError(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	helpers.MustCloseStorage(t, mockStorage)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:   http.MethodGet,
		Endpoint: server.OrganizationsEndpoint,
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       `{"status": "sql: database is closed"}`,
	})
}

func TestServerStart(t *testing.T) {
	helpers.RunTestWithTimeout(t, func(t *testing.T) {
		s := server.New(server.Configuration{
			// will use any free port
			Address:   ":0",
			APIPrefix: config.APIPrefix,
			Auth:      true,
			Debug:     true,
		}, nil)

		go func() {
			for {
				if s.Serv != nil {
					break
				}

				time.Sleep(500 * time.Millisecond)
			}

			// doing some request to be sure server started successfully
			req, err := http.NewRequest(http.MethodGet, config.APIPrefix, nil)
			helpers.FailOnError(t, err)

			response := helpers.ExecuteRequest(s, req, &config).Result()
			checkResponseCode(t, http.StatusForbidden, response.StatusCode)

			// stopping the server
			err = s.Stop(context.Background())
			helpers.FailOnError(t, err)
		}()

		err := s.Start()
		if err != nil && err != http.ErrServerClosed {
			t.Fatal(err)
		}
	}, 5*time.Second)
}

func TestServerStartError(t *testing.T) {
	testServer := server.New(server.Configuration{
		Address:   "localhost:99999",
		APIPrefix: "",
	}, nil)

	err := testServer.Start()
	assert.EqualError(t, err, "listen tcp: address 99999: invalid port")
}

func TestServeAPISpecFileOK(t *testing.T) {
	err := os.Chdir("../")
	helpers.FailOnError(t, err)

	fileData, err := ioutil.ReadFile(config.APISpecFile)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:   http.MethodGet,
		Endpoint: config.APISpecFile,
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       string(fileData),
	})
}

func TestServeAPISpecFileError(t *testing.T) {
	dirName, err := ioutil.TempDir("/tmp/", "")
	helpers.FailOnError(t, err)

	err = os.Chdir(dirName)
	helpers.FailOnError(t, err)

	err = os.Remove(dirName)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:   http.MethodGet,
		Endpoint: config.APISpecFile,
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       `{"status":"Error creating absolute path of OpenAPI spec file"}`,
	})
}

func TestRuleFeedbackVote(t *testing.T) {
	for _, endpoint := range []string{
		server.LikeRuleEndpoint, server.DislikeRuleEndpoint, server.ResetVoteOnRuleEndpoint,
	} {
		var expectedVote storage.UserVote

		switch endpoint {
		case server.LikeRuleEndpoint:
			expectedVote = storage.UserVoteLike
		case server.DislikeRuleEndpoint:
			expectedVote = storage.UserVoteDislike
		case server.ResetVoteOnRuleEndpoint:
			expectedVote = storage.UserVoteNone
		default:
			t.Fatal("not expected action")
		}

		func(endpoint string) {
			mockStorage := helpers.MustGetMockStorage(t, true)
			defer helpers.MustCloseStorage(t, mockStorage)

			helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
				Method:       http.MethodPut,
				Endpoint:     endpoint,
				EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID},
				UserID:       testUserID,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       `{"status": "ok"}`,
			})

			feedback, err := mockStorage.GetUserFeedbackOnRule(testdata.ClusterName, testdata.Rule1ID, testUserID)
			helpers.FailOnError(t, err)

			assert.Equal(t, testdata.ClusterName, feedback.ClusterID)
			assert.Equal(t, testdata.Rule1ID, feedback.RuleID)
			assert.Equal(t, testUserID, feedback.UserID)
			assert.Equal(t, "", feedback.Message)
			assert.Equal(t, expectedVote, feedback.UserVote)
		}(endpoint)
	}
}

func TestRuleFeedbackErrorBadClusterName(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.LikeRuleEndpoint,
		EndpointArgs: []interface{}{testBadClusterName, testRuleID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body:       `{"status": "invalid cluster name format: 'aaaa'"}`,
	})
}

func TestRuleFeedbackErrorClosedStorage(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	helpers.MustCloseStorage(t, mockStorage)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.LikeRuleEndpoint,
		EndpointArgs: []interface{}{testClusterName, testRuleID},
		UserID:       testUserID,
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       `{"status": "sql: database is closed"}`,
	})
}

func TestDeleteReportsByOrganization(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodDelete,
		Endpoint:     server.DeleteOrganizationsEndpoint,
		EndpointArgs: []interface{}{1},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"status": "ok"}`,
	})
}

func TestHTTPServer_deleteOrganizations_NonIntOrgID(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodDelete,
		Endpoint:     server.DeleteOrganizationsEndpoint,
		EndpointArgs: []interface{}{"non-int"},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body:       `{"status": "bad organizations param, integer array expected"}`,
	})
}

func TestDeleteReportsByClusterName(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodDelete,
		Endpoint:     server.DeleteClustersEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"status": "ok"}`,
	})
}
