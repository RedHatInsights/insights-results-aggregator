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
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/tests/testdata"
)

var config = server.Configuration{
	Address:     ":8080",
	APIPrefix:   "/api/test/",
	APISpecFile: "openapi.json",
	Debug:       true,
	Auth:        false,
	UseHTTPS:    false,
	EnableCORS:  true,
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}

func TestMakeURLToEndpoint(t *testing.T) {
	assert.Equal(
		t,
		"api/prefix/report/-55/cluster_id",
		server.MakeURLToEndpoint("api/prefix/", server.ReportEndpoint, -55, "cluster_id"),
	)
}

func TestAddCORSHeaders(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.WriteReportForCluster(testdata.OrgID, testdata.ClusterName, "{}", time.Now())
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodOptions,
		Endpoint:     server.ReportEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID, testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Headers: map[string]string{
			"Access-Control-Allow-Origin":      "*",
			"Access-Control-Allow-Methods":     "POST, GET, OPTIONS, PUT, DELETE",
			"Access-Control-Allow-Headers":     "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization",
			"Access-Control-Allow-Credentials": "true",
		},
	})
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
		Body:       `{"status": "Internal Server Error"}`,
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
			"status": "Error during parsing param 'organization' with value '-1'. Error: 'unsigned integer expected'"
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
			"status": "Error during parsing param 'organization' with value 'non-int'. Error: 'unsigned integer expected'"
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
		Body:       `{"status": "Internal Server Error"}`,
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
		Body:       `{"status": "Internal Server Error"}`,
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

			err := mockStorage.WriteReportForCluster(
				testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.LastCheckedAt,
			)
			helpers.FailOnError(t, err)

			err = mockStorage.LoadRuleContent(testdata.RuleContent3Rules)
			helpers.FailOnError(t, err)

			helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
				Method:       http.MethodPut,
				Endpoint:     endpoint,
				EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID},
				UserID:       testdata.UserID,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       `{"status": "ok"}`,
			})

			feedback, err := mockStorage.GetUserFeedbackOnRule(testdata.ClusterName, testdata.Rule1ID, testdata.UserID)
			helpers.FailOnError(t, err)

			assert.Equal(t, testdata.ClusterName, feedback.ClusterID)
			assert.Equal(t, testdata.Rule1ID, feedback.RuleID)
			assert.Equal(t, testdata.UserID, feedback.UserID)
			assert.Equal(t, "", feedback.Message)
			assert.Equal(t, expectedVote, feedback.UserVote)
		}(endpoint)
	}
}

func TestRuleFeedbackVote_CheckIfRuleExists_DBError(t *testing.T) {
	const errStr = "Internal Server Error"

	mockStorage, expects := helpers.MustGetMockStorageWithExpects(t)
	defer helpers.MustCloseMockStorageWithExpects(t, mockStorage, expects)

	expects.ExpectQuery("SELECT .* FROM report").
		WillReturnRows(
			sqlmock.NewRows([]string{"report", "last_checked_at"}).AddRow("1", time.Now()),
		)

	expects.ExpectQuery("SELECT .* FROM rule").
		WillReturnError(fmt.Errorf(errStr))

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.LikeRuleEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID},
		UserID:       testdata.UserID,
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       `{"status": "` + errStr + `"}`,
	})
}

func TestRuleFeedbackVote_DBError(t *testing.T) {
	const errStr = "Internal Server Error"

	mockStorage, expects := helpers.MustGetMockStorageWithExpects(t)
	defer helpers.MustCloseMockStorageWithExpects(t, mockStorage, expects)

	expects.ExpectQuery("SELECT .* FROM report").
		WillReturnRows(
			sqlmock.NewRows([]string{"report", "last_checked_at"}).AddRow("1", time.Now()),
		)

	expects.ExpectQuery("SELECT .* FROM rule").
		WillReturnRows(
			sqlmock.NewRows(
				[]string{
					"module",
					"name",
					"summary",
					"reason",
					"resolution",
					"more_info",
				},
			).AddRow(
				testdata.Rule1ID,
				testdata.Rule1Name,
				testdata.Rule1Summary,
				testdata.Rule1Reason,
				testdata.Rule1Resolution,
				testdata.Rule1MoreInfo,
			),
		)

	expects.ExpectPrepare("INSERT INTO").
		WillReturnError(fmt.Errorf(errStr))

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.LikeRuleEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID},
		UserID:       testdata.UserID,
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       `{"status": "` + errStr + `"}`,
	})
}

func TestHTTPServer_UserFeedback_ClusterDoesNotExistError(t *testing.T) {
	for _, endpoint := range []string{
		server.LikeRuleEndpoint, server.DislikeRuleEndpoint, server.ResetVoteOnRuleEndpoint,
	} {
		helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
			Method:       http.MethodPut,
			Endpoint:     endpoint,
			EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID},
			UserID:       testdata.UserID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusNotFound,
			Body: fmt.Sprintf(
				`{"status": "Item with ID %v was not found in the storage"}`,
				testdata.ClusterName,
			),
		})
	}
}

func TestHTTPServer_UserFeedback_RuleDoesNotExistError(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.LastCheckedAt,
	)
	helpers.FailOnError(t, err)

	for _, endpoint := range []string{
		server.LikeRuleEndpoint, server.DislikeRuleEndpoint, server.ResetVoteOnRuleEndpoint,
	} {
		helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
			Method:       http.MethodPut,
			Endpoint:     endpoint,
			EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID},
			UserID:       testdata.UserID,
		}, &helpers.APIResponse{
			StatusCode: http.StatusNotFound,
			Body: fmt.Sprintf(
				`{"status": "Item with ID %v was not found in the storage"}`,
				testdata.Rule1ID,
			),
		})
	}
}

func TestRuleFeedbackErrorBadClusterName(t *testing.T) {
	buf := new(bytes.Buffer)
	log.Logger = zerolog.New(buf)

	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.LikeRuleEndpoint,
		EndpointArgs: []interface{}{testdata.BadClusterName, testdata.Rule1ID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body:       `{"status": "Error during parsing param 'cluster' with value 'aaaa'. Error: 'invalid UUID length: 4'"}`,
	})

	assert.Contains(t, buf.String(), "invalid cluster name: 'aaaa'. Error: invalid UUID length: 4")
}

func TestRuleFeedbackErrorBadRuleID(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.LikeRuleEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.BadRuleID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body: `{
			"status": "Error during parsing param 'rule_id' with value 'rule id with spaces'. Error: 'invalid rule ID, it must contain only from latin characters, number, underscores or dots'"
		}`,
	})
}

func TestHTTPServer_GetVoteOnRule_BadRuleID(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.GetVoteOnRuleEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.BadRuleID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body: `{
			"status": "Error during parsing param 'rule_id' with value 'rule id with spaces'. Error: 'invalid rule ID, it must contain only from latin characters, number, underscores or dots'"
		}`,
	})
}

func TestHTTPServer_GetVoteOnRule_DBError(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.LastCheckedAt,
	)
	helpers.FailOnError(t, err)

	err = mockStorage.LoadRuleContent(testdata.RuleContent3Rules)
	helpers.FailOnError(t, err)

	connection := mockStorage.(*storage.DBStorage).GetConnection()

	_, err = connection.Exec(`DROP TABLE cluster_rule_user_feedback;`)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.GetVoteOnRuleEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       `{"status": "Internal Server Error"}`,
	})
}

func TestRuleFeedbackErrorBadUserID(t *testing.T) {
	testServer := server.New(config, nil)

	url := server.MakeURLToEndpoint(config.APIPrefix, server.LikeRuleEndpoint, testdata.ClusterName, testdata.Rule1ID)

	req, err := http.NewRequest(http.MethodPut, url, nil)
	helpers.FailOnError(t, err)

	// put wrong identity
	identity := "wrong type"
	req = req.WithContext(context.WithValue(req.Context(), server.ContextKeyUser, identity))

	response := helpers.ExecuteRequest(testServer, req, &config).Result()

	assert.Equal(t, http.StatusInternalServerError, response.StatusCode, "Expected different status code")
	helpers.CheckResponseBodyJSON(t, `{"status": "Internal Server Error"}`, response.Body)
}

func TestRuleFeedbackErrorClosedStorage(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	helpers.MustCloseStorage(t, mockStorage)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.LikeRuleEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID},
		UserID:       testdata.UserID,
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       `{"status": "Internal Server Error"}`,
	})
}

func TestHTTPServer_GetVoteOnRule(t *testing.T) {
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

			err := mockStorage.WriteReportForCluster(
				testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.LastCheckedAt,
			)
			helpers.FailOnError(t, err)

			err = mockStorage.LoadRuleContent(testdata.RuleContent3Rules)
			helpers.FailOnError(t, err)

			helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
				Method:       http.MethodPut,
				Endpoint:     endpoint,
				EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID},
				UserID:       testdata.UserID,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       `{"status": "ok"}`,
			})

			helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.GetVoteOnRuleEndpoint,
				EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID},
				UserID:       testdata.UserID,
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       fmt.Sprintf(`{"status": "ok", "vote":%v}`, expectedVote),
			})
		}(endpoint)
	}
}

func TestHTTPServer_deleteOrganizationsOK(t *testing.T) {
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
		Body:       `{"status": "Error during parsing param 'organizations' with value 'non-int'. Error: 'integer array expected'"}`,
	})
}

func TestHTTPServer_deleteOrganizations_DBError(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	helpers.MustCloseStorage(t, mockStorage)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodDelete,
		Endpoint:     server.DeleteOrganizationsEndpoint,
		EndpointArgs: []interface{}{testdata.OrgID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       `{"status": "Internal Server Error"}`,
	})
}

func TestHTTPServer_deleteClusters(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodDelete,
		Endpoint:     server.DeleteClustersEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body:       `{"status": "ok"}`,
	})
}

func TestHTTPServer_deleteClusters_DBError(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	helpers.MustCloseStorage(t, mockStorage)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodDelete,
		Endpoint:     server.DeleteClustersEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName},
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       `{"status": "Internal Server Error"}`,
	})
}

func TestHTTPServer_deleteClusters_BadClusterName(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodDelete,
		Endpoint:     server.DeleteClustersEndpoint,
		EndpointArgs: []interface{}{testdata.BadClusterName},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body:       `{"status": "Error during parsing param 'cluster' with value 'aaaa'. Error: 'invalid UUID length: 4'"}`,
	})
}

func TestHTTPServer_CreateRule(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodPost,
		Endpoint:     server.RuleEndpoint,
		EndpointArgs: []interface{}{testdata.Rule1ID},
		Body: fmt.Sprintf(`{
			"module": "%v",
			"name": "%v",
			"summary": "%v",
			"reason": "%v",
			"resolution": "%v",
			"more_info": "%v"
		}`,
			testdata.Rule1ID,
			testdata.Rule1Name,
			testdata.Rule1Summary,
			testdata.Rule1Reason,
			testdata.Rule1Resolution,
			testdata.Rule1MoreInfo,
		),
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body: `{
			"status": "ok",
			"rule": ` + fmt.Sprintf(`{
				"module": "%v",
				"name": "%v",
				"summary": "%v",
				"reason": "%v",
				"resolution": "%v",
				"more_info": "%v"
			}`,
			testdata.Rule1ID,
			testdata.Rule1Name,
			testdata.Rule1Summary,
			testdata.Rule1Reason,
			testdata.Rule1Resolution,
			testdata.Rule1MoreInfo,
		) + `
		}`,
	})
}

func TestHTTPServer_CreateRule_BadRuleID(t *testing.T) {
	const errMessage = "Error during parsing param 'rule_id' with value 'rule id with spaces'." +
		" Error: 'invalid rule ID, it must contain only from latin characters, number, underscores or dots'"

	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodPost,
		Endpoint:     server.RuleEndpoint,
		EndpointArgs: []interface{}{testdata.BadRuleID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body:       `{"status": "` + errMessage + `"}`,
	})
}

func TestHTTPServer_CreateRule_BadRuleData(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodPost,
		Endpoint:     server.RuleEndpoint,
		EndpointArgs: []interface{}{testdata.Rule1ID},
		Body:         "not-json",
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body:       `{"status": "invalid character 'o' in literal null (expecting 'u')"}`,
	})
}

func TestHTTPServer_CreateRule_DBError(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	connection := mockStorage.(*storage.DBStorage).GetConnection()

	_, err := connection.Exec(`DROP TABLE rule;`)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodPost,
		Endpoint:     server.RuleEndpoint,
		EndpointArgs: []interface{}{testdata.Rule1ID},
		Body: fmt.Sprintf(`{
			"module": "%v",
			"name": "%v",
			"summary": "%v",
			"reason": "%v",
			"resolution": "%v",
			"more_info": "%v"
		}`,
			testdata.Rule1ID,
			testdata.Rule1Name,
			testdata.Rule1Summary,
			testdata.Rule1Reason,
			testdata.Rule1Resolution,
			testdata.Rule1MoreInfo,
		),
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       `{"status": "Internal Server Error"}`,
	})
}

func TestHTTPServer_CreateRuleErrorKey(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodPost,
		Endpoint:     server.RuleEndpoint,
		EndpointArgs: []interface{}{testdata.Rule1ID},
		Body: fmt.Sprintf(`{
			"module": "%v",
			"name": "%v",
			"summary": "%v",
			"reason": "%v",
			"resolution": "%v",
			"more_info": "%v"
		}`,
			testdata.Rule1ID,
			testdata.Rule1Name,
			testdata.Rule1Summary,
			testdata.Rule1Reason,
			testdata.Rule1Resolution,
			testdata.Rule1MoreInfo,
		),
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body: `{
			"status": "ok",
			"rule": ` + fmt.Sprintf(`{
				"module": "%v",
				"name": "%v",
				"summary": "%v",
				"reason": "%v",
				"resolution": "%v",
				"more_info": "%v"
			}`,
			testdata.Rule1ID,
			testdata.Rule1Name,
			testdata.Rule1Summary,
			testdata.Rule1Reason,
			testdata.Rule1Resolution,
			testdata.Rule1MoreInfo,
		) + `
		}`,
	})

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodPost,
		Endpoint:     server.RuleErrorKeyEndpoint,
		EndpointArgs: []interface{}{testdata.Rule1ID, "ek"},
		Body: fmt.Sprintf(`{
			"error_key": "%v",
			"rule_module": "%v",
			"condition": "%v",
			"description": "%v",
			"impact": %v,
			"likelihood": %v,
			"publish_date": "%v",
			"active": %v,
			"generic": "%v"
		}`,
			"ek",
			"module",
			"",
			"",
			1,
			2,
			testdata.LastCheckedAt.Format(time.RFC3339),
			true,
			"",
		),
	}, &helpers.APIResponse{
		StatusCode: http.StatusOK,
		Body: `{
			"status": "ok",
			"rule_error_key": ` + fmt.Sprintf(`{
				"error_key": "%v",
				"rule_module": "%v",
				"condition": "%v",
				"description": "%v",
				"impact": %v,
				"likelihood": %v,
				"publish_date": "%v",
				"active": %v,
				"generic": "%v"
			}`,
			"ek",
			testdata.Rule1ID,
			"",
			"",
			1,
			2,
			testdata.LastCheckedAt.Format(time.RFC3339),
			true,
			"",
		) + `
		}`,
	})
}

func TestHTTPServer_CreateRuleErrorKey_BadRuleKey(t *testing.T) {
	const errMessage = "Error during parsing param 'rule_id' with value 'rule id with spaces'." +
		" Error: 'invalid rule ID, it must contain only from latin characters, number, underscores or dots'"

	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodPost,
		Endpoint:     server.RuleErrorKeyEndpoint,
		EndpointArgs: []interface{}{testdata.BadRuleID, "ek"},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body:       `{"status": "` + errMessage + `"}`,
	})
}

func TestHTTPServer_CreateRuleErrorKey_BadRuleData(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodPost,
		Endpoint:     server.RuleErrorKeyEndpoint,
		EndpointArgs: []interface{}{testdata.Rule1ID, "ek"},
		Body:         "not-json",
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body:       `{"status": "invalid character 'o' in literal null (expecting 'u')"}`,
	})
}
