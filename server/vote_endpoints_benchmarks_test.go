// Copyright 2020, 2021, 2022, 2023 Red Hat, Inc
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
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"

	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

func BenchmarkHTTPServer_VoteEndpoints_WithSQLiteMemoryStorage(b *testing.B) {
	mockStorage, closer := helpers.MustGetMockStorage(b, true)
	defer closer()

	benchmarkHTTPServerVoteEndpointsWithStorage(b, mockStorage)
}

func BenchmarkHTTPServer_VoteEndpoints_WithSQLiteFileStorage(b *testing.B) {
	mockStorage, cleaner := helpers.MustGetSQLiteFileStorage(b, true)
	defer cleaner()

	benchmarkHTTPServerVoteEndpointsWithStorage(b, mockStorage)
}

func BenchmarkHTTPServer_VoteEndpoints_WithPostgresStorage(b *testing.B) {
	mockStorage, cleaner := helpers.MustGetPostgresStorage(b, true)
	defer cleaner()

	benchmarkHTTPServerVoteEndpointsWithStorage(b, mockStorage)
}

func benchmarkHTTPServerVoteEndpointsWithStorage(b *testing.B, mockStorage storage.OCPRecommendationsStorage) {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

	// each test case will choose random endpoint args from this pool
	const numberOfEndpointArgs = 512

	endpointArgs := prepareVoteEndpointArgs(b, numberOfEndpointArgs, mockStorage)
	defer cleanupEndpointArgs(b, endpointArgs, mockStorage)

	testServer := server.New(helpers.DefaultServerConfig, mockStorage)

	type TestCase struct {
		TestName string
		Endpoint string
		N        uint
	}

	numberOfTestCases := []uint{1, 10, 100}
	if testing.Short() {
		numberOfTestCases = []uint{1, 5}
	}

	testCases := make([]TestCase, 3*len(numberOfTestCases))

	for i, n := range numberOfTestCases {
		testCases[i*3+0] = TestCase{"like", server.LikeRuleEndpoint, n}
		testCases[i*3+1] = TestCase{"dislike", server.DislikeRuleEndpoint, n}
		testCases[i*3+2] = TestCase{"reset_vote", server.ResetVoteOnRuleEndpoint, n}
	}

	for _, testCase := range testCases {
		testCase.TestName += fmt.Sprintf("/N=%v", testCase.N)

		b.Run(testCase.TestName, func(b *testing.B) {
			for benchIndex := 0; benchIndex < b.N; benchIndex++ {
				for i := uint(0); i < testCase.N; i++ {
					endpointArg := endpointArgs[rand.Intn(len(endpointArgs))]
					clusterID := endpointArg.ClusterID
					ruleID := endpointArg.RuleID
					userID := endpointArg.UserID
					orgID := endpointArg.OrgID

					url := httputils.MakeURLToEndpoint(
						helpers.DefaultServerConfig.APIPrefix,
						testCase.Endpoint,
						clusterID, ruleID,
					)

					req, err := http.NewRequest(http.MethodPut, url, http.NoBody)
					helpers.FailOnError(b, err)

					// authorize user
					identity := types.Identity{
						AccountNumber: userID,
						OrgID:         orgID,
						User: types.User{
							UserID: userID,
						},
					}
					req = req.WithContext(context.WithValue(req.Context(), types.ContextKeyUser, identity))

					response := helpers.ExecuteRequest(testServer, req).Result()

					assert.Equal(b, http.StatusOK, response.StatusCode)
				}
			}
		})
	}
}

type voteEndpointArg struct {
	ClusterID types.ClusterName
	RuleID    types.RuleID
	OrgID     types.OrgID
	UserID    types.UserID
	ErrorKey  types.ErrorKey
}

func prepareVoteEndpointArgs(tb testing.TB, numberOfEndpointArgs uint, mockStorage storage.OCPRecommendationsStorage) []voteEndpointArg {
	var endpointArgs []voteEndpointArg

	for i := uint(0); i < numberOfEndpointArgs; i++ {
		clusterID := types.ClusterName(uuid.New().String())
		ruleID := testdata.GetRandomRuleID(32)
		errorKey := types.ErrorKey("ek")
		userID := testdata.GetRandomUserID()
		orgID := testdata.GetRandomOrgID()

		err := mockStorage.WriteReportForCluster(
			testdata.OrgID,
			clusterID,
			"{}",
			testdata.ReportEmptyRulesParsed,
			time.Now(),
			time.Now(),
			time.Now(),
			testdata.RequestID1,
		)
		helpers.FailOnError(tb, err)

		endpointArgs = append(endpointArgs, voteEndpointArg{
			ClusterID: clusterID,
			RuleID:    ruleID,
			OrgID:     orgID,
			UserID:    userID,
			ErrorKey:  errorKey,
		})
	}

	return endpointArgs
}

func cleanupEndpointArgs(tb testing.TB, args []voteEndpointArg, mockStorage storage.OCPRecommendationsStorage) {
	for _, arg := range args {
		err := mockStorage.DeleteReportsForCluster(arg.ClusterID)
		helpers.FailOnError(tb, err)
	}
}
