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
	"encoding/json"
	"strconv"
	"testing"
	"time"

	"database/sql"
	"database/sql/driver"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

var (
	now             = time.Now().UTC()
	nowAfterOneHour = now.Add(1 * time.Hour).UTC()
	dummyTime       = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

	namespaceAWorkload = types.DVOWorkload{
		Namespace:    "namespace-name-A",
		NamespaceUID: "NAMESPACE-UID-A",
		Kind:         "DaemonSet",
		Name:         "test-name-0099",
		UID:          "UID-0099",
	}
	namespaceBWorkload = types.DVOWorkload{
		Namespace:    "namespace-name-B",
		NamespaceUID: "NAMESPACE-UID-B",
		Kind:         "NotDaemonSet",
		Name:         "test-name-1199",
		UID:          "UID-1199",
	}
	validDVORecommendation = []types.WorkloadRecommendation{
		{
			ResponseID: "an_issue|DVO_AN_ISSUE",
			Component:  "ccx_rules_ocp.external.dvo.an_issue_pod.recommendation",
			Key:        "DVO_AN_ISSUE",
			Links: types.DVOLinks{
				Jira:                 []string{"https://issues.redhat.com/browse/AN_ISSUE"},
				ProductDocumentation: []string{},
			},
			Details: map[string]interface{}{
				"check_name": "",
				"check_url":  "",
				"samples": []map[string]interface{}{
					{"namespace_uid": "193a2099-1234-5678-916a-d570c9aac158", "kind": "Deployment", "uid": "0501e150-1234-5678-907f-ee732c25044a"},
					{"namespace_uid": "337477af-1234-5678-b258-16f19d8a6289", "kind": "Deployment", "uid": "8c534861-1234-5678-9af5-913de71a545b"},
				},
			},
			Tags: []string{},
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
	validReport = `{"system":{"metadata":{},"hostname":null},"fingerprints":[],"version":1,"analysis_metadata":{},"workload_recommendations":[{"response_id":"an_issue|DVO_AN_ISSUE","component":"ccx_rules_ocp.external.dvo.an_issue_pod.recommendation","key":"DVO_AN_ISSUE","details":{},"tags":[],"links":{"jira":["https://issues.redhat.com/browse/AN_ISSUE"],"product_documentation":[]},"workloads":[{"namespace":"namespace-name-A","namespace_uid":"NAMESPACE-UID-A","kind":"DaemonSet","name":"test-name-0099","uid":"UID-0099"}]}]}`

	twoNamespacesRecommendation = []types.WorkloadRecommendation{
		{
			ResponseID: "an_issue|DVO_AN_ISSUE",
			Component:  "ccx_rules_ocp.external.dvo.an_issue_pod.recommendation",
			Key:        "DVO_AN_ISSUE",
			Links: types.DVOLinks{
				Jira:                 []string{"https://issues.redhat.com/browse/AN_ISSUE"},
				ProductDocumentation: []string{},
			},
			Details:   types.DVODetails{CheckName: "", CheckURL: ""},
			Tags:      []string{},
			Workloads: []types.DVOWorkload{namespaceAWorkload, namespaceBWorkload},
		},
	}
)

func init() {
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
}

func TestDVOStorage_Init(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)
	defer closer()

	dbStorage := mockStorage.(*storage.DVORecommendationsDBStorage)

	err := dbStorage.MigrateToLatest()
	helpers.FailOnError(t, err)
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
		now,
		dummyTime,
		dummyTime,
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
		now,
		dummyTime,
		dummyTime,
		testdata.RequestID1,
	)
	assert.EqualError(t, err, "writing workloads with DB -1 is not supported")
}

// TestDVOStorageWriteReportForClusterMoreRecentInDB checks that older report
// will not replace a more recent one when writing a report to storage.
func TestDVOStorageWriteReportForClusterMoreRecentInDB(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)
	defer closer()

	newerTime := now.UTC()
	olderTime := newerTime.Add(-time.Hour)

	// Insert newer report.
	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.ClusterReportEmpty,
		validDVORecommendation,
		newerTime,
		dummyTime,
		dummyTime,
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
		now,
		now,
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
		validDVORecommendation, now, now, now,
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

	expects.ExpectExec("DELETE FROM dvo.dvo_report").
		WillReturnResult(driver.ResultNoRows)

	expects.ExpectExec("INSERT INTO dvo.dvo_report").
		WillReturnResult(driver.ResultNoRows)

	expects.ExpectCommit()
	expects.ExpectClose()
	expects.ExpectClose()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, `{"test": "report"}`,
		validDVORecommendation, testdata.LastCheckedAt, now, now,
		testdata.RequestID1)
	helpers.FailOnError(t, mockStorage.Close())
	helpers.FailOnError(t, err)
}

func TestDVOStorageWriteReportForClusterCheckItIsStored(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)
	defer closer()

	err := mockStorage.DeleteReportsForOrg(testdata.OrgID)
	helpers.FailOnError(t, err)

	err = mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		types.ClusterReport(validReport),
		validDVORecommendation,
		now,
		dummyTime,
		dummyTime,
		testdata.RequestID1,
	)
	helpers.FailOnError(t, err)

	row := mockStorage.GetConnection().QueryRow(
		"SELECT namespace_id, namespace_name, report, recommendations, objects, last_checked_at, reported_at FROM dvo.dvo_report WHERE org_id = $1 AND cluster_id = $2;",
		testdata.OrgID, testdata.ClusterName,
	)
	checkStoredReport(t, row, namespaceAWorkload, 1, now, now)
}

func TestDVOStorageWriteReportForClusterCheckPreviousIsDeleted(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)
	defer closer()

	err := mockStorage.DeleteReportsForOrg(testdata.OrgID)
	helpers.FailOnError(t, err)

	err = mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		types.ClusterReport(validReport),
		twoNamespacesRecommendation,
		now,
		dummyTime,
		dummyTime,
		testdata.RequestID1,
	)
	helpers.FailOnError(t, err)

	// Check both namespaces are stored in the DB
	row := mockStorage.GetConnection().QueryRow(`
		SELECT namespace_id, namespace_name, report, recommendations, objects, last_checked_at, reported_at
		FROM dvo.dvo_report WHERE org_id = $1 AND cluster_id = $2 AND namespace_id = $3;`,
		testdata.OrgID, testdata.ClusterName, namespaceAWorkload.NamespaceUID,
	)
	checkStoredReport(t, row, namespaceAWorkload, 1, now, now)
	row = mockStorage.GetConnection().QueryRow(`
		SELECT namespace_id, namespace_name, report, recommendations, objects, last_checked_at, reported_at
		FROM dvo.dvo_report WHERE org_id = $1 AND cluster_id = $2 AND namespace_id = $3;`,
		testdata.OrgID, testdata.ClusterName, namespaceBWorkload.NamespaceUID,
	)
	checkStoredReport(t, row, namespaceBWorkload, 1, now, now)

	// Now receive a report with just one namespace for the same cluster
	err = mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		types.ClusterReport(validReport),
		validDVORecommendation,
		nowAfterOneHour,
		dummyTime,
		dummyTime,
		testdata.RequestID1,
	)
	helpers.FailOnError(t, err)

	// Make sure just one namespace is in the DB now
	row = mockStorage.GetConnection().QueryRow(`
		SELECT namespace_id, namespace_name, report, recommendations, objects, last_checked_at, reported_at
		FROM dvo.dvo_report WHERE org_id = $1 AND cluster_id = $2 AND namespace_id = $3;`,
		testdata.OrgID, testdata.ClusterName, namespaceAWorkload.NamespaceUID,
	)
	checkStoredReport(t, row, namespaceAWorkload, 1, nowAfterOneHour, now)
	row = mockStorage.GetConnection().QueryRow(`
		SELECT namespace_id, namespace_name, report, recommendations, objects, last_checked_at, reported_at
		FROM dvo.dvo_report WHERE org_id = $1 AND cluster_id = $2 AND namespace_id = $3;`,
		testdata.OrgID, testdata.ClusterName, namespaceBWorkload.NamespaceUID,
	)
	checkRowDoesntExist(t, row)
}

func checkStoredReport(t *testing.T, row *sql.Row, want types.DVOWorkload, wantObjects int, wantLastChecked, wantReportedAt time.Time) {
	var (
		namespaceID     string
		namespaceName   string
		report          types.ClusterReport
		recommendations int
		objects         int
		lastChecked     time.Time
		reportedAt      time.Time
	)

	err := row.Scan(&namespaceID, &namespaceName, &report, &recommendations, &objects, &lastChecked, &reportedAt)
	helpers.FailOnError(t, err)

	unquotedReport, err := strconv.Unquote(string(report))
	helpers.FailOnError(t, err)
	var gotWorkloads types.DVOMetrics
	err = json.Unmarshal([]byte(unquotedReport), &gotWorkloads)
	helpers.FailOnError(t, err)

	assert.Equal(t, want.NamespaceUID, namespaceID, "the column namespace_id is different than expected")
	assert.Equal(t, want.Namespace, namespaceName, "the column namespace_name is different than expected")
	assert.Equal(t, validDVORecommendation, gotWorkloads.WorkloadRecommendations, "the column report is different than expected")
	assert.Equal(t, 1, recommendations, "the column recommendations is different than expected")
	assert.Equal(t, wantObjects, objects, "the column objects is different than expected")
	assert.Equal(t, wantLastChecked.Truncate(time.Second), lastChecked.UTC().Truncate(time.Second), "the column reported_at is different than expected")
	assert.Equal(t, wantReportedAt.Truncate(time.Second), reportedAt.UTC().Truncate(time.Second), "the column last_checked_at is different than expected")
}

func checkRowDoesntExist(t *testing.T, row *sql.Row) {
	var (
		namespaceID     string
		namespaceName   string
		report          types.ClusterReport
		recommendations int
		objects         int
		lastChecked     time.Time
		reportedAt      time.Time
	)

	err := row.Scan(&namespaceID, &namespaceName, &report, &recommendations, &objects, &lastChecked, &reportedAt)
	assert.ErrorIs(t, err, sql.ErrNoRows, "a row was found for this queryß")
}
