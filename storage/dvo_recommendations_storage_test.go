/*
Copyright © 2023 Red Hat, Inc.

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

package storage_test

import (
	"testing"
	"time"

	"database/sql/driver"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

var validDVORecommendation = []types.WorkloadRecommendation{
	{
		ResponseID: "an_issue|DVO_AN_ISSUE",
		Component:  "ccx_rules_ocp.external.dvo.an_issue_pod.recommendation",
		Key:        "DVO_AN_ISSUE",
		Links: types.DVOLinks{
			Jira:                 []string{"https://issues.redhat.com/browse/AN_ISSUE"},
			ProductDocumentation: []string{},
		},
		Details: types.DVODetails{CheckName: "", CheckURL: ""},
		Tags:    []string{},
		Workloads: []types.DVOWorkload{
			{
				Namespace:    "namespace-name-A",
				NamespaceUID: "NAMESPACE-UID-A",
				Kind:         "DaemonSet",
				Name:         "test-name-0099",
				UID:          "UID-0099",
			},
		},
	},
}

func init() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

// TestNewDVOStorageError checks whether constructor for new DVO storage returns error for improper storage configuration
func TestNewDVOStorageError(t *testing.T) {
	_, err := storage.NewDVORecommendationsStorage(storage.Configuration{
		Driver: "non existing driver",
		Type:   "sql",
	})
	assert.EqualError(t, err, "driver non existing driver is not supported")
}

// TestNewDVOStorageNoType checks whether constructor for new DVO storage returns error for improper storage configuration
func TestNewDVOStorageNoType(t *testing.T) {
	_, err := storage.NewDVORecommendationsStorage(storage.Configuration{
		Driver: "non existing driver",
	})
	assert.EqualError(t, err, "Unknown storage type ''")
}

// TestNewDVOStorageWrongType checks whether constructor for new DVO storage returns error for improper storage configuration
func TestNewDVOStorageWrongType(t *testing.T) {
	_, err := storage.NewDVORecommendationsStorage(storage.Configuration{
		Driver: "non existing driver",
		Type:   "foobar",
	})
	assert.EqualError(t, err, "Unknown storage type 'foobar'")
}

// TestNewDVOStorageReturnedImplementation check what implementation of storage is returnd
func TestNewDVOStorageReturnedImplementation(t *testing.T) {
	s, _ := storage.NewDVORecommendationsStorage(storage.Configuration{
		Driver:        "postgres",
		PGPort:        1234,
		PGUsername:    "user",
		LogSQLQueries: true,
		Type:          "sql",
	})
	assert.IsType(t, &storage.DVORecommendationsDBStorage{}, s)

	s, _ = storage.NewDVORecommendationsStorage(storage.Configuration{
		Driver:        "postgres",
		PGPort:        1234,
		PGUsername:    "user",
		LogSQLQueries: true,
		Type:          "noop",
	})
	assert.IsType(t, &storage.NoopDVOStorage{}, s)

	s, _ = storage.NewDVORecommendationsStorage(storage.Configuration{
		Driver:        "postgres",
		PGPort:        1234,
		PGUsername:    "user",
		LogSQLQueries: true,
		Type:          "redis",
	})
	assert.Nil(t, s, "redis type is not supported for DVO storage")
}

// TestDVOStorageWriteReportForClusterClosedStorage check the behaviour of method WriteReportForCluster
func TestDVOStorageWriteReportForClusterClosedStorage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)
	// we need to close storage right now
	closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.ClusterReportEmpty,
		validDVORecommendation,
		time.Now(),
		time.Now(),
		time.Now(),
		testdata.RequestID1,
	)
	assert.EqualError(t, err, "sql: database is closed")
}

// TestDVOStorageWriteReportForClusterUnsupportedDriverError check the behaviour of method WriteReportForCluster
func TestDVOStorageWriteReportForClusterUnsupportedDriverError(t *testing.T) {
	fakeStorage := storage.NewDVORecommendationsFromConnection(nil, -1)
	// no need to close it

	err := fakeStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.ClusterReportEmpty,
		validDVORecommendation,
		time.Now(),
		time.Now(),
		time.Now(),
		testdata.RequestID1,
	)
	assert.EqualError(t, err, "writing workloads with DB -1 is not supported")
}

// TestDVOStorageWriteReportForClusterMoreRecentInDB checks that older report
// will not replace a more recent one when writing a report to storage.
func TestDVOStorageWriteReportForClusterMoreRecentInDB(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)
	defer closer()

	newerTime := time.Now().UTC()
	olderTime := newerTime.Add(-time.Hour)

	// Insert newer report.
	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.ClusterReportEmpty,
		validDVORecommendation,
		newerTime,
		time.Now(),
		time.Now(),
		testdata.RequestID1,
	)
	helpers.FailOnError(t, err)

	// Try to insert older report.
	err = mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.ClusterReportEmpty,
		validDVORecommendation,
		olderTime,
		time.Now(),
		time.Now(),
		testdata.RequestID1,
	)
	assert.Equal(t, types.ErrOldReport, err)
}

// TestDVOStorageWriteReportForClusterDroppedReportTable checks the error
// returned when trying to SELECT from a dropped/missing dvo.dvo_report table.
func TestDVOStorageWriteReportForClusterDroppedReportTable(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)
	defer closer()

	connection := storage.GetConnectionDVO(mockStorage.(*storage.DVORecommendationsDBStorage))

	query := "DROP TABLE dvo.dvo_report CASCADE;"

	_, err := connection.Exec(query)
	helpers.FailOnError(t, err)

	err = mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.ClusterReportEmpty,
		validDVORecommendation, time.Now(), time.Now(), time.Now(),
		testdata.RequestID1,
	)
	assert.EqualError(t, err, "no such table: dvo.dvo_report")
}

func TestDVOStorageWriteReportForClusterFakePostgresOK(t *testing.T) {
	mockStorage, expects := ira_helpers.MustGetMockStorageWithExpectsForDriverDVO(t, types.DBDriverPostgres)
	defer ira_helpers.MustCloseMockStorageWithExpectsDVO(t, mockStorage, expects)

	expects.ExpectBegin()

	expects.ExpectQuery(`SELECT last_checked_at FROM dvo.dvo_report`).
		WillReturnRows(expects.NewRows([]string{"last_checked_at"})).
		RowsWillBeClosed()

	expects.ExpectExec("INSERT INTO dvo.dvo_report").
		WillReturnResult(driver.ResultNoRows)

	expects.ExpectCommit()
	expects.ExpectClose()
	expects.ExpectClose()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, `{"test": "report"}`,
		validDVORecommendation, testdata.LastCheckedAt, time.Now(), time.Now(),
		testdata.RequestID1)
	helpers.FailOnError(t, mockStorage.Close())
	helpers.FailOnError(t, err)
}
