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
	_ "embed"
	"encoding/json"
	"fmt"
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
	ira_data "github.com/RedHatInsights/insights-results-aggregator/tests/data"
	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

var (
	now             = time.Now().UTC()
	nowAfterOneHour = now.Add(1 * time.Hour).UTC()
	dummyTime       = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
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
		ira_data.ValidDVORecommendation,
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
		ira_data.ValidDVORecommendation,
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
		ira_data.ValidDVORecommendation,
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
		ira_data.ValidDVORecommendation,
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
		ira_data.ValidDVORecommendation, now, now, now,
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
		ira_data.ValidDVORecommendation, testdata.LastCheckedAt, now, now,
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
		types.ClusterReport(ira_data.ValidReport),
		ira_data.ValidDVORecommendation,
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
	checkStoredReport(t, row, ira_data.NamespaceAWorkload, 1, now, now)
}

func TestDVOStorageWriteReportForClusterCheckPreviousIsDeleted(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)
	defer closer()

	err := mockStorage.DeleteReportsForOrg(testdata.OrgID)
	helpers.FailOnError(t, err)

	err = mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		types.ClusterReport(ira_data.ValidReport),
		ira_data.TwoNamespacesRecommendation,
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
		testdata.OrgID, testdata.ClusterName, ira_data.NamespaceAWorkload.NamespaceUID,
	)
	checkStoredReport(t, row, ira_data.NamespaceAWorkload, 1, now, now)
	row = mockStorage.GetConnection().QueryRow(`
		SELECT namespace_id, namespace_name, report, recommendations, objects, last_checked_at, reported_at
		FROM dvo.dvo_report WHERE org_id = $1 AND cluster_id = $2 AND namespace_id = $3;`,
		testdata.OrgID, testdata.ClusterName, ira_data.NamespaceBWorkload.NamespaceUID,
	)
	checkStoredReport(t, row, ira_data.NamespaceBWorkload, 1, now, now)

	// Now receive a report with just one namespace for the same cluster
	err = mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		types.ClusterReport(ira_data.ValidReport),
		ira_data.ValidDVORecommendation,
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
		testdata.OrgID, testdata.ClusterName, ira_data.NamespaceAWorkload.NamespaceUID,
	)
	checkStoredReport(t, row, ira_data.NamespaceAWorkload, 1, nowAfterOneHour, now)
	row = mockStorage.GetConnection().QueryRow(`
		SELECT namespace_id, namespace_name, report, recommendations, objects, last_checked_at, reported_at
		FROM dvo.dvo_report WHERE org_id = $1 AND cluster_id = $2 AND namespace_id = $3;`,
		testdata.OrgID, testdata.ClusterName, ira_data.NamespaceBWorkload.NamespaceUID,
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
	assert.Equal(t, ira_data.ValidDVORecommendation, gotWorkloads.WorkloadRecommendations, "the column report is different than expected")
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

// TestDVOStorageReadWorkloadsForOrganization tests timestamps being kept correctly
func TestDVOStorageReadWorkloadsForOrganization(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)
	defer closer()

	activeClusterMap := make(map[types.ClusterName]struct{})
	activeClusterMap[testdata.ClusterName] = struct{}{}

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		types.ClusterReport(ira_data.ValidReport),
		ira_data.TwoNamespacesRecommendation,
		now,
		now,
		now,
		testdata.RequestID1,
	)
	helpers.FailOnError(t, err)

	// write new archive with newer timestamp, old reported_at must be kept
	err = mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		types.ClusterReport(ira_data.ValidReport),
		ira_data.TwoNamespacesRecommendation,
		nowAfterOneHour,
		nowAfterOneHour,
		nowAfterOneHour,
		testdata.RequestID1,
	)
	helpers.FailOnError(t, err)

	workloads, err := mockStorage.ReadWorkloadsForOrganization(testdata.OrgID, activeClusterMap, true)
	helpers.FailOnError(t, err)

	assert.Equal(t, testdata.ClusterName, types.ClusterName(workloads[0].Cluster.UUID))
	assert.Equal(t, testdata.ClusterName, types.ClusterName(workloads[1].Cluster.UUID))
	assert.Equal(t, nowAfterOneHour.UTC().Format(time.RFC3339), workloads[0].Metadata.LastCheckedAt)
	assert.Equal(t, nowAfterOneHour.UTC().Format(time.RFC3339), workloads[1].Metadata.LastCheckedAt)
	assert.Equal(t, now.UTC().Format(time.RFC3339), workloads[0].Metadata.ReportedAt)
	assert.Equal(t, now.UTC().Format(time.RFC3339), workloads[1].Metadata.ReportedAt)
	assert.Equal(t, types.RuleHitsCount{"ccx_rules_ocp.external.dvo.an_issue_pod|DVO_AN_ISSUE": 1}, workloads[0].RecommendationsHitCount)
	assert.Equal(t, types.RuleHitsCount{"ccx_rules_ocp.external.dvo.an_issue_pod|DVO_AN_ISSUE": 1}, workloads[1].RecommendationsHitCount)
}

// TestDVOStorageReadWorkloadsForNamespace tests timestamps being kept correctly
func TestDVOStorageReadWorkloadsForNamespace_Timestamps(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		types.ClusterReport(ira_data.ValidReport),
		ira_data.TwoNamespacesRecommendation,
		now,
		now,
		now,
		testdata.RequestID1,
	)
	helpers.FailOnError(t, err)

	// write new archive with newer timestamp, old reported_at must be kept
	err = mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		types.ClusterReport(ira_data.ValidReport),
		ira_data.TwoNamespacesRecommendation,
		nowAfterOneHour,
		nowAfterOneHour,
		nowAfterOneHour,
		testdata.RequestID1,
	)
	helpers.FailOnError(t, err)

	report, err := mockStorage.ReadWorkloadsForClusterAndNamespace(testdata.OrgID, testdata.ClusterName, ira_data.NamespaceAUID)
	helpers.FailOnError(t, err)

	assert.Equal(t, testdata.ClusterName, types.ClusterName(report.ClusterID))
	assert.Equal(t, ira_data.NamespaceAUID, report.NamespaceID)
	assert.Equal(t, uint(1), report.Recommendations)
	assert.Equal(t, uint(1), report.Objects)
	assert.Equal(t, types.Timestamp(nowAfterOneHour.UTC().Format(time.RFC3339)), report.LastCheckedAt)
	assert.Equal(t, types.Timestamp(now.UTC().Format(time.RFC3339)), report.ReportedAt)
}

// TestDVOStorageReadWorkloadsForNamespace tests the behavior when we insert 1 recommendation and 1 object
// and then 1 recommendation with 2 objects
func TestDVOStorageReadWorkloadsForNamespace_TwoObjectsOneNamespace(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)
	defer closer()

	nowTstmp := types.Timestamp(now.UTC().Format(time.RFC3339))

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		types.ClusterReport(ira_data.ValidReport),
		ira_data.TwoNamespacesRecommendation,
		now,
		now,
		now,
		testdata.RequestID1,
	)
	helpers.FailOnError(t, err)

	report, err := mockStorage.ReadWorkloadsForClusterAndNamespace(testdata.OrgID, testdata.ClusterName, ira_data.NamespaceAUID)
	helpers.FailOnError(t, err)

	assert.Equal(t, testdata.ClusterName, types.ClusterName(report.ClusterID))
	assert.Equal(t, ira_data.NamespaceAUID, report.NamespaceID)
	assert.Equal(t, uint(1), report.Recommendations)
	assert.Equal(t, uint(1), report.Objects)
	assert.Equal(t, nowTstmp, report.ReportedAt)
	assert.Equal(t, nowTstmp, report.LastCheckedAt)

	newerReport2Objs := ira_data.TwoNamespacesRecommendation
	newerReport2Objs[0].Workloads = []types.DVOWorkload{ira_data.NamespaceAWorkload, ira_data.NamespaceAWorkload2, ira_data.NamespaceBWorkload}
	// write new archive with newer timestamp and 1 more object in the recommendation hit
	err = mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		types.ClusterReport(ira_data.ValidReport),
		newerReport2Objs,
		nowAfterOneHour,
		nowAfterOneHour,
		nowAfterOneHour,
		testdata.RequestID1,
	)
	helpers.FailOnError(t, err)

	report, err = mockStorage.ReadWorkloadsForClusterAndNamespace(testdata.OrgID, testdata.ClusterName, ira_data.NamespaceAUID)
	helpers.FailOnError(t, err)

	assert.Equal(t, testdata.ClusterName, types.ClusterName(report.ClusterID))
	assert.Equal(t, ira_data.NamespaceAUID, report.NamespaceID)
	assert.Equal(t, uint(1), report.Recommendations)
	assert.Equal(t, uint(2), report.Objects) // <-- two objs now

	// reported_at keeps timestamp, last_checked_at gets updated
	assert.Equal(t, types.Timestamp(nowAfterOneHour.UTC().Format(time.RFC3339)), report.LastCheckedAt)
	assert.Equal(t, nowTstmp, report.ReportedAt)
}

// TestDVOStorageReadWorkloadsForNamespace tests the behavior when we insert 1 recommendation and 1 object
// and then 1 recommendation with 2 objects
func TestDVOStorageWriteReport_TwoNamespacesTwoRecommendations(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)
	defer closer()

	activeClusterMap := make(map[types.ClusterName]struct{})
	activeClusterMap[testdata.ClusterName] = struct{}{}

	nowTstmp := types.Timestamp(now.UTC().Format(time.RFC3339))

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		types.ClusterReport(ira_data.ValidReport2Rules2Namespaces),
		[]types.WorkloadRecommendation{ira_data.Recommendation1TwoNamespaces, ira_data.Recommendation2OneNamespace},
		now,
		now,
		now,
		testdata.RequestID1,
	)
	helpers.FailOnError(t, err)

	expectedWorkloads := []types.WorkloadsForNamespace{
		{
			Cluster: types.Cluster{
				UUID: string(testdata.ClusterName),
			},
			Namespace: types.Namespace{
				UUID: ira_data.NamespaceAUID,
				Name: ira_data.NamespaceAWorkload.Namespace,
			},
			Metadata: types.DVOMetadata{
				Recommendations: 2,
				Objects:         2, // <-- must be 2, because one workload is hitting more recommendations, but counts as 1
				ReportedAt:      now.UTC().Format(time.RFC3339),
				LastCheckedAt:   now.UTC().Format(time.RFC3339),
			},
			RecommendationsHitCount: types.RuleHitsCount{
				"ccx_rules_ocp.external.dvo.an_issue_pod|DVO_AN_ISSUE":                 1,
				"ccx_rules_ocp.external.dvo.unset_requirements|DVO_UNSET_REQUIREMENTS": 2,
			},
		},
		{
			Cluster: types.Cluster{
				UUID: string(testdata.ClusterName),
			},
			Namespace: types.Namespace{
				UUID: ira_data.NamespaceBUID,
				Name: ira_data.NamespaceBWorkload.Namespace,
			},
			Metadata: types.DVOMetadata{
				Recommendations: 1, // <-- must contain only 1 rule, the other rule wasn't hitting this ns
				Objects:         1,
				ReportedAt:      now.UTC().Format(time.RFC3339),
				LastCheckedAt:   now.UTC().Format(time.RFC3339),
			},
			RecommendationsHitCount: types.RuleHitsCount{
				"ccx_rules_ocp.external.dvo.an_issue_pod|DVO_AN_ISSUE": 1,
			},
		},
	}

	workloads, err := mockStorage.ReadWorkloadsForOrganization(testdata.OrgID, activeClusterMap, true)
	helpers.FailOnError(t, err)

	assert.Equal(t, 2, len(workloads))
	assert.ElementsMatch(t, expectedWorkloads, workloads)

	report, err := mockStorage.ReadWorkloadsForClusterAndNamespace(testdata.OrgID, testdata.ClusterName, ira_data.NamespaceAUID)
	helpers.FailOnError(t, err)

	assert.Equal(t, testdata.ClusterName, types.ClusterName(report.ClusterID))
	assert.Equal(t, ira_data.NamespaceAUID, report.NamespaceID)
	assert.Equal(t, uint(2), report.Recommendations)
	assert.Equal(t, uint(2), report.Objects)
	assert.Equal(t, nowTstmp, report.ReportedAt)
	assert.Equal(t, nowTstmp, report.LastCheckedAt)
}

func TestDVOStorageWriteReport_FilterOutDuplicateObjects_CCXDEV_12608_Reproducer(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)
	defer closer()

	activeClusterMap := make(map[types.ClusterName]struct{})
	activeClusterMap[testdata.ClusterName] = struct{}{}

	nowTstmp := types.Timestamp(now.UTC().Format(time.RFC3339))

	// writing 3 recommendations, but objects/workloads are hitting multiple recommendations
	// and need to be filtered out
	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		types.ClusterReport(ira_data.ValidReport2Rules2Namespaces),
		[]types.WorkloadRecommendation{
			ira_data.Recommendation1TwoNamespaces,
			ira_data.Recommendation2OneNamespace,
			ira_data.Recommendation3OneNamespace,
		},
		now,
		now,
		now,
		testdata.RequestID1,
	)
	helpers.FailOnError(t, err)

	expectedWorkloads := []types.WorkloadsForNamespace{
		{
			Cluster: types.Cluster{
				UUID: string(testdata.ClusterName),
			},
			Namespace: types.Namespace{
				UUID: ira_data.NamespaceAUID,
				Name: ira_data.NamespaceAWorkload.Namespace,
			},
			Metadata: types.DVOMetadata{
				Recommendations: 3,
				Objects:         2, // <-- must be 2, because workloadA and workloadB are hitting more rules, but count as 1 within a namespace
				ReportedAt:      now.UTC().Format(time.RFC3339),
				LastCheckedAt:   now.UTC().Format(time.RFC3339),
			},
			RecommendationsHitCount: types.RuleHitsCount{
				"ccx_rules_ocp.external.dvo.an_issue_pod|DVO_AN_ISSUE":                 1,
				"ccx_rules_ocp.external.dvo.unset_requirements|DVO_UNSET_REQUIREMENTS": 2,
				"ccx_rules_ocp.external.dvo.bad_requirements|BAD_REQUIREMENTS":         2,
			},
		},
		{
			Cluster: types.Cluster{
				UUID: string(testdata.ClusterName),
			},
			Namespace: types.Namespace{
				UUID: ira_data.NamespaceBUID,
				Name: ira_data.NamespaceBWorkload.Namespace,
			},
			Metadata: types.DVOMetadata{
				Recommendations: 1, // <-- must contain only 1 rule, the other rules weren't affecting this namespace
				Objects:         1,
				ReportedAt:      now.UTC().Format(time.RFC3339),
				LastCheckedAt:   now.UTC().Format(time.RFC3339),
			},
			RecommendationsHitCount: types.RuleHitsCount{
				"ccx_rules_ocp.external.dvo.an_issue_pod|DVO_AN_ISSUE": 1,
			},
		},
	}

	workloads, err := mockStorage.ReadWorkloadsForOrganization(testdata.OrgID, activeClusterMap, true)
	helpers.FailOnError(t, err)

	assert.Equal(t, 2, len(workloads))
	assert.ElementsMatch(t, expectedWorkloads, workloads)

	report, err := mockStorage.ReadWorkloadsForClusterAndNamespace(testdata.OrgID, testdata.ClusterName, ira_data.NamespaceAUID)
	helpers.FailOnError(t, err)

	assert.Equal(t, testdata.ClusterName, types.ClusterName(report.ClusterID))
	assert.Equal(t, ira_data.NamespaceAUID, report.NamespaceID)
	assert.Equal(t, uint(3), report.Recommendations)
	assert.Equal(t, uint(2), report.Objects)
	assert.Equal(t, nowTstmp, report.ReportedAt)
	assert.Equal(t, nowTstmp, report.LastCheckedAt)
}

// TestDVOStorageWriteReport_ActiveClusterListFiltering tests the behavior when providing the active
// cluster list in POST body
func TestDVOStorageWriteReport_ActiveClusterListFiltering(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)
	defer closer()

	enabledCluster := testdata.GetRandomClusterID()

	// only contains one cluster
	activeClusterMap := make(map[types.ClusterName]struct{})
	activeClusterMap[enabledCluster] = struct{}{}

	nowTstmp := types.Timestamp(now.UTC().Format(time.RFC3339))

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		types.ClusterReport(ira_data.ValidReport2Rules2Namespaces),
		[]types.WorkloadRecommendation{ira_data.Recommendation1TwoNamespaces, ira_data.Recommendation2OneNamespace},
		now,
		now,
		now,
		testdata.RequestID1,
	)
	helpers.FailOnError(t, err)

	err = mockStorage.WriteReportForCluster(
		testdata.OrgID,
		enabledCluster,
		types.ClusterReport(ira_data.ValidReport2Rules2Namespaces),
		[]types.WorkloadRecommendation{ira_data.Recommendation1TwoNamespaces, ira_data.Recommendation2OneNamespace},
		now,
		now,
		now,
		testdata.RequestID1,
	)
	helpers.FailOnError(t, err)

	expectedWorkloads := []types.WorkloadsForNamespace{
		{
			Cluster: types.Cluster{
				UUID: string(enabledCluster), // <-- not in active cluster list
			},
			Namespace: types.Namespace{
				UUID: ira_data.NamespaceAUID,
				Name: ira_data.NamespaceAWorkload.Namespace,
			},
			Metadata: types.DVOMetadata{
				Recommendations: 2,
				Objects:         2,
				ReportedAt:      now.UTC().Format(time.RFC3339),
				LastCheckedAt:   now.UTC().Format(time.RFC3339),
			},
			RecommendationsHitCount: types.RuleHitsCount{
				"ccx_rules_ocp.external.dvo.an_issue_pod|DVO_AN_ISSUE":                 1,
				"ccx_rules_ocp.external.dvo.unset_requirements|DVO_UNSET_REQUIREMENTS": 2,
			},
		},
		{
			Cluster: types.Cluster{
				UUID: string(enabledCluster),
			},
			Namespace: types.Namespace{
				UUID: ira_data.NamespaceBUID,
				Name: ira_data.NamespaceBWorkload.Namespace,
			},
			Metadata: types.DVOMetadata{
				Recommendations: 1,
				Objects:         1,
				ReportedAt:      now.UTC().Format(time.RFC3339),
				LastCheckedAt:   now.UTC().Format(time.RFC3339),
			},
			RecommendationsHitCount: types.RuleHitsCount{
				"ccx_rules_ocp.external.dvo.an_issue_pod|DVO_AN_ISSUE": 1,
			},
		},
	}

	workloads, err := mockStorage.ReadWorkloadsForOrganization(testdata.OrgID, activeClusterMap, true)
	helpers.FailOnError(t, err)

	assert.Equal(t, 2, len(workloads))
	assert.ElementsMatch(t, expectedWorkloads, workloads)

	report, err := mockStorage.ReadWorkloadsForClusterAndNamespace(testdata.OrgID, testdata.ClusterName, ira_data.NamespaceAUID)
	helpers.FailOnError(t, err)

	assert.Equal(t, testdata.ClusterName, types.ClusterName(report.ClusterID))
	assert.Equal(t, ira_data.NamespaceAUID, report.NamespaceID)
	assert.Equal(t, uint(2), report.Recommendations)
	assert.Equal(t, uint(2), report.Objects)
	assert.Equal(t, nowTstmp, report.ReportedAt)
	assert.Equal(t, nowTstmp, report.LastCheckedAt)
}

// TestDVOStorageReadWorkloadsForNamespace_MissingData tests wht happens if the data is missing
func TestDVOStorageReadWorkloadsForNamespace_MissingData(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)
	defer closer()

	// write data for namespaceA and testdata.ClusterName
	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		types.ClusterReport(ira_data.ValidReport),
		ira_data.ValidDVORecommendation,
		now,
		now,
		now,
		testdata.RequestID1,
	)
	helpers.FailOnError(t, err)

	t.Run("cluster and namespace exist", func(t *testing.T) {
		report, err := mockStorage.ReadWorkloadsForClusterAndNamespace(testdata.OrgID, testdata.ClusterName, ira_data.NamespaceAUID)
		helpers.FailOnError(t, err)
		assert.Equal(t, testdata.ClusterName, types.ClusterName(report.ClusterID))
		assert.Equal(t, ira_data.NamespaceAUID, report.NamespaceID)
	})

	t.Run("cluster exists and namespace doesn't", func(t *testing.T) {
		_, err := mockStorage.ReadWorkloadsForClusterAndNamespace(testdata.OrgID, testdata.ClusterName, ira_data.NamespaceBUID)
		assert.Equal(t, &types.ItemNotFoundError{ItemID: fmt.Sprintf("%d:%s:%s", testdata.OrgID, testdata.ClusterName, ira_data.NamespaceBUID)}, err)
	})

	t.Run("namespace exists and cluster doesn't", func(t *testing.T) {
		nonExistingCluster := types.ClusterName("a6fe3cd2-2c6a-48b8-a58d-b05853d47f4f")
		_, err := mockStorage.ReadWorkloadsForClusterAndNamespace(testdata.OrgID, nonExistingCluster, ira_data.NamespaceAUID)
		assert.Equal(t, &types.ItemNotFoundError{ItemID: fmt.Sprintf("%d:%s:%s", testdata.OrgID, nonExistingCluster, ira_data.NamespaceAUID)}, err)
	})
}

//go:embed report_for_heartbeats.json
var heartbeatsFilteringReport string

func TestReadWorkloadsForClusterAndNamespace_HearbeatsFiltering(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorageDVO(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		types.ClusterReport(heartbeatsFilteringReport),
		ira_data.ValidDVORecommendation,
		now,
		now,
		now,
		testdata.RequestID1,
	)
	helpers.FailOnError(t, err)

	// write heartbeats
	mockStorage.WriteHeartbeat("UID-0099", time.Now().UTC())
	mockStorage.WriteHeartbeat("UID-0100", time.Now().UTC().Add(-1*time.Hour).UTC())

	report, err := mockStorage.ReadWorkloadsForClusterAndNamespace(testdata.OrgID, testdata.ClusterName, "NAMESPACE-UID-A")
	helpers.FailOnError(t, err)

	assert.Equal(t, uint(1), report.Objects)

}
