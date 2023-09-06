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
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	httputils "github.com/RedHatInsights/insights-operator-utils/http"

	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

func BenchmarkHTTPServer_ReadReportForCluster(b *testing.B) {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)

	sameReportProvider := func() (types.ClusterReport, []types.ReportItem) {
		return testdata.Report3Rules, testdata.Report3RulesParsed
	}

	type testCase struct {
		storageName     string
		storageProvider func(testing.TB, bool) (storage.Storage, func())
		N               uint
	}

	var testCases []testCase

	for _, n := range []uint{1, 10, 100, 1000} {
		for storageName, storageProvider := range map[string]func(testing.TB, bool) (storage.Storage, func()){
			"SQLiteMemory": helpers.MustGetSQLiteMemoryStorage,
			"SQLiteFile":   helpers.MustGetSQLiteFileStorage,
			"Postgres":     helpers.MustGetPostgresStorage,
		} {
			testCases = append(testCases, testCase{
				storageName,
				storageProvider,
				n,
			})
		}
	}

	for _, testCase := range testCases {
		func() {
			mockStorage, cleaner := testCase.storageProvider(b, true)
			defer cleaner()

			testReportDataItems := initTestReports(b, 1, mockStorage, sameReportProvider)

			b.Run(fmt.Sprintf("%v/%v/N=%v", "SameReport", testCase.storageName, testCase.N), func(b *testing.B) {
				benchmarkHTTPServerReadReportForCluster(b, mockStorage, testReportDataItems, testCase.N)
			})
		}()
	}
}

func benchmarkHTTPServerReadReportForCluster(
	b *testing.B,
	mockStorage storage.Storage,
	testReportDataItems []testReportData,
	n uint,
) {
	testServer := server.New(helpers.DefaultServerConfig, mockStorage)

	b.ResetTimer()
	for benchIndex := 0; benchIndex < b.N; benchIndex++ {
		for i := uint(0); i < n; i++ {
			testReportDataItem := testReportDataItems[rand.Intn(len(testReportDataItems))]

			orgID := testReportDataItem.orgID
			clusterID := testReportDataItem.clusterID

			url := httputils.MakeURLToEndpoint(
				helpers.DefaultServerConfig.APIPrefix,
				server.ReportEndpoint,
				orgID, clusterID,
			)

			req, err := http.NewRequest(http.MethodGet, url, http.NoBody)
			helpers.FailOnError(b, err)

			response := helpers.ExecuteRequest(testServer, req).Result()
			respBody, err := io.ReadAll(response.Body)
			helpers.FailOnError(b, err)

			var resp struct {
				Report struct {
					Data []interface{} `json:"data"`
				} `json:"report"`
			}

			err = json.Unmarshal(respBody, &resp)
			helpers.FailOnError(b, err)

			assert.NotEmpty(b, resp.Report.Data, "Server should return some reports")

			assert.Equal(b, http.StatusOK, response.StatusCode)
		}
	}
}

type testReportData struct {
	orgID     types.OrgID
	clusterID types.ClusterName
}

func initTestReports(b *testing.B, n uint, mockStorage storage.Storage, reportProvider func() (types.ClusterReport, []types.ReportItem)) []testReportData {
	var testReportDataItems []testReportData

	for i := uint(0); i < n; i++ {
		orgID := testdata.GetRandomOrgID()
		clusterID := testdata.GetRandomClusterID()
		report, rules := reportProvider()

		err := mockStorage.WriteReportForCluster(
			orgID,
			clusterID,
			report,
			rules,
			time.Now(),
			time.Now(),
			time.Now(),
			testdata.RequestID1,
		)
		helpers.FailOnError(b, err)

		testReportDataItems = append(testReportDataItems, testReportData{
			orgID:     orgID,
			clusterID: clusterID,
		})
	}

	return testReportDataItems
}
