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
	"database/sql"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"

	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

var config = server.Configuration{
	Address:     ":8080",
	APIPrefix:   "/api/test/",
	APISpecFile: "openapi.json",
	Debug:       true,
	Auth:        false,
}

func init() {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
}

func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}

func TestMakeURLToEndpoint(t *testing.T) {
	assert.Equal(
		t,
		"api/prefix/organizations/2/clusters/cluster_id/users/1/report",
		httputils.MakeURLToEndpoint("api/prefix/", server.ReportEndpoint, 2, "cluster_id", 1),
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
	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.LastCheckedAt, testdata.KafkaOffset,
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
	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	closer()

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
	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		1, "8083c377-8a05-4922-af8d-e7d0970c1f49", "{}", time.Now(), testdata.KafkaOffset,
	)
	helpers.FailOnError(t, err)

	err = mockStorage.WriteReportForCluster(
		5, "52ab955f-b769-444d-8170-4b676c5d3c85", "{}", time.Now(), testdata.KafkaOffset,
	)
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
	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	closer()

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

			response := helpers.ExecuteRequest(s, req).Result()
			checkResponseCode(t, http.StatusForbidden, response.StatusCode)

			// stopping the server
			err = s.Stop(context.Background())
			helpers.FailOnError(t, err)
		}()

		err := s.Start(nil)
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

	err := testServer.Start(nil)
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
		var expectedVote types.UserVote

		switch endpoint {
		case server.LikeRuleEndpoint:
			expectedVote = types.UserVoteLike
		case server.DislikeRuleEndpoint:
			expectedVote = types.UserVoteDislike
		case server.ResetVoteOnRuleEndpoint:
			expectedVote = types.UserVoteNone
		default:
			t.Fatal("not expected action")
		}

		func(endpoint string) {
			mockStorage, closer := helpers.MustGetMockStorage(t, true)
			defer closer()

			err := mockStorage.WriteReportForCluster(
				testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.LastCheckedAt, testdata.KafkaOffset,
			)
			helpers.FailOnError(t, err)

			helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
				Method:       http.MethodPut,
				Endpoint:     endpoint,
				EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.UserID},
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

func TestRuleFeedbackVote_DBError(t *testing.T) {
	const errStr = "Internal Server Error"

	mockStorage, expects := helpers.MustGetMockStorageWithExpects(t)
	defer helpers.MustCloseMockStorageWithExpects(t, mockStorage, expects)

	expects.ExpectQuery("SELECT .* FROM report").
		WillReturnRows(
			sqlmock.NewRows([]string{"report", "last_checked_at"}).AddRow("1", time.Now()),
		)

	expects.ExpectPrepare("INSERT INTO").
		WillReturnError(fmt.Errorf(errStr))

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.LikeRuleEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.UserID},
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
			EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.UserID},
		}, &helpers.APIResponse{
			StatusCode: http.StatusNotFound,
			Body: fmt.Sprintf(
				`{"status": "Item with ID %v was not found in the storage"}`,
				testdata.ClusterName,
			),
		})
	}
}

// TODO: make working with the new arch
//func TestHTTPServer_UserFeedback_RuleDoesNotExistError(t *testing.T) {
//	mockStorage, closer := helpers.MustGetMockStorage(t, true)
//	defer closer()
//
//	err := mockStorage.WriteReportForCluster(
//		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.LastCheckedAt, testdata.KafkaOffset,
//	)
//	helpers.FailOnError(t, err)
//
//	for _, endpoint := range []string{
//		server.LikeRuleEndpoint, server.DislikeRuleEndpoint, server.ResetVoteOnRuleEndpoint,
//	} {
//		helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
//			Method:       http.MethodPut,
//			Endpoint:     endpoint,
//			EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.UserID},
//		}, &helpers.APIResponse{
//			StatusCode: http.StatusNotFound,
//			Body: fmt.Sprintf(
//				`{"status": "Item with ID %v was not found in the storage"}`,
//				testdata.Rule1ID,
//			),
//		})
//	}
//}

func TestRuleFeedbackErrorBadClusterName(t *testing.T) {
	buf := new(bytes.Buffer)
	log.Logger = zerolog.New(buf)

	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.LikeRuleEndpoint,
		EndpointArgs: []interface{}{testdata.BadClusterName, testdata.Rule1ID, testdata.UserID},
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
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.BadRuleID, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body: `{
			"status": "Error during parsing param 'rule_id' with value 'rule id with spaces'. Error: 'invalid rule ID, it must contain only from latin characters, number, underscores or dots'"
		}`,
	})
}

func TestRuleFeedbackErrorLongMessage(t *testing.T) {
	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.LastCheckedAt, testdata.KafkaOffset,
	)
	helpers.FailOnError(t, err)
	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.LikeRuleEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.UserID},
		Body: `{
			"message": "Veryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryvery long message"
		}`,
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body: `{
			"status": "Error during validating param 'message' with value 'Veryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryveryvery long message'. Error: 'String is longer then 250'"
		}`,
	})
}

func TestHTTPServer_GetVoteOnRule_BadRuleID(t *testing.T) {
	helpers.AssertAPIRequest(t, nil, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.GetVoteOnRuleEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.BadRuleID, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusBadRequest,
		Body: `{
			"status": "Error during parsing param 'rule_id' with value 'rule id with spaces'. Error: 'invalid rule ID, it must contain only from latin characters, number, underscores or dots'"
		}`,
	})
}

func TestHTTPServer_GetVoteOnRule_DBError(t *testing.T) {
	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.LastCheckedAt, testdata.KafkaOffset,
	)
	helpers.FailOnError(t, err)

	connection := mockStorage.(*storage.DBStorage).GetConnection()

	_, err = connection.Exec(`DROP TABLE cluster_rule_user_feedback;`)
	helpers.FailOnError(t, err)

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodGet,
		Endpoint:     server.GetVoteOnRuleEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       `{"status": "Internal Server Error"}`,
	})
}

func TestRuleFeedbackErrorClosedStorage(t *testing.T) {
	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	closer()

	helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
		Method:       http.MethodPut,
		Endpoint:     server.LikeRuleEndpoint,
		EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.UserID},
	}, &helpers.APIResponse{
		StatusCode: http.StatusInternalServerError,
		Body:       `{"status": "Internal Server Error"}`,
	})
}

func TestHTTPServer_GetVoteOnRule(t *testing.T) {
	for _, endpoint := range []string{
		server.LikeRuleEndpoint, server.DislikeRuleEndpoint, server.ResetVoteOnRuleEndpoint,
	} {
		var expectedVote types.UserVote

		switch endpoint {
		case server.LikeRuleEndpoint:
			expectedVote = types.UserVoteLike
		case server.DislikeRuleEndpoint:
			expectedVote = types.UserVoteDislike
		case server.ResetVoteOnRuleEndpoint:
			expectedVote = types.UserVoteNone
		default:
			t.Fatal("not expected action")
		}

		func(endpoint string) {
			mockStorage, closer := helpers.MustGetMockStorage(t, true)
			defer closer()

			err := mockStorage.WriteReportForCluster(
				testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.LastCheckedAt, testdata.KafkaOffset,
			)
			helpers.FailOnError(t, err)

			helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
				Method:       http.MethodPut,
				Endpoint:     endpoint,
				EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.UserID},
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       `{"status": "ok"}`,
			})

			helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
				Method:       http.MethodGet,
				Endpoint:     server.GetVoteOnRuleEndpoint,
				EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.UserID},
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       fmt.Sprintf(`{"status": "ok", "vote":%v}`, expectedVote),
			})
		}(endpoint)
	}
}

func TestRuleToggle(t *testing.T) {
	for _, endpoint := range []string{
		server.DisableRuleForClusterEndpoint, server.EnableRuleForClusterEndpoint,
	} {
		var expectedState storage.RuleToggle

		switch endpoint {
		case server.DisableRuleForClusterEndpoint:
			expectedState = storage.RuleToggleDisable
		case server.EnableRuleForClusterEndpoint:
			expectedState = storage.RuleToggleEnable
		default:
			t.Fatal("no such endpoint")
		}

		func(endpoint string) {
			mockStorage, closer := helpers.MustGetMockStorage(t, true)
			defer closer()

			err := mockStorage.WriteReportForCluster(
				testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.LastCheckedAt, testdata.KafkaOffset,
			)
			helpers.FailOnError(t, err)

			helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
				Method:       http.MethodPut,
				Endpoint:     endpoint,
				EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, testdata.UserID},
			}, &helpers.APIResponse{
				StatusCode: http.StatusOK,
				Body:       `{"status": "ok"}`,
			})

			toggledRule, err := mockStorage.GetFromClusterRuleToggle(testdata.ClusterName, testdata.Rule1ID, testdata.UserID)
			helpers.FailOnError(t, err)

			assert.Equal(t, testdata.ClusterName, toggledRule.ClusterID)
			assert.Equal(t, testdata.Rule1ID, toggledRule.RuleID)
			assert.Equal(t, testdata.UserID, toggledRule.UserID)
			assert.Equal(t, expectedState, toggledRule.Disabled)
			if toggledRule.Disabled == storage.RuleToggleDisable {
				assert.Equal(t, sql.NullTime{}, toggledRule.EnabledAt)
			} else {
				assert.Equal(t, sql.NullTime{}, toggledRule.DisabledAt)
			}
		}(endpoint)
	}
}

func TestRuleToggle_EmptyUserID(t *testing.T) {
	for _, endpoint := range []string{
		server.DisableRuleForClusterEndpoint, server.EnableRuleForClusterEndpoint,
	} {
		func(endpoint string) {
			mockStorage, closer := helpers.MustGetMockStorage(t, true)
			defer closer()

			err := mockStorage.WriteReportForCluster(
				testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.LastCheckedAt, testdata.KafkaOffset,
			)
			helpers.FailOnError(t, err)

			helpers.AssertAPIRequest(t, mockStorage, &config, &helpers.APIRequest{
				Method:       http.MethodPut,
				Endpoint:     endpoint,
				EndpointArgs: []interface{}{testdata.ClusterName, testdata.Rule1ID, " "}, // space is crucial here
			}, &helpers.APIResponse{
				StatusCode: http.StatusBadRequest,
				Body:       `{"status": "Missing required param from request: user_id"}`,
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
	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	closer()

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
	mockStorage, closer := helpers.MustGetMockStorage(t, true)
	closer()

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
