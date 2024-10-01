/*
Copyright © 2020, 2021, 2022 Red Hat, Inc.

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
	"bytes"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	ctypes "github.com/RedHatInsights/insights-results-types"
	"github.com/Shopify/sarama"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"

	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

var (
	RecommendationCreatedAtTimestamp = types.Timestamp(testdata.LastCheckedAt.Format(time.RFC3339))
	RecommendationImpactedSinceMap   = make(map[string]types.Timestamp)
)

func init() {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
}

func assertNumberOfReports(t *testing.T, mockStorage storage.OCPRecommendationsStorage, expectedNumberOfReports int) {
	numberOfReports, err := mockStorage.ReportsCount()
	helpers.FailOnError(t, err)
	assert.Equal(t, expectedNumberOfReports, numberOfReports)
}

func checkReportForCluster(
	t *testing.T,
	s storage.OCPRecommendationsStorage,
	orgID types.OrgID,
	clusterName types.ClusterName,
	expected []types.RuleOnReport,
) {
	// try to read report for cluster
	result, _, _, _, err := s.ReadReportForCluster(orgID, clusterName)
	helpers.FailOnError(t, err)

	// and check the read report with expected one
	assert.ElementsMatch(t, expected, result)
}

func writeReportForCluster(
	t *testing.T,
	storageImpl storage.OCPRecommendationsStorage,
	orgID types.OrgID,
	clusterName types.ClusterName,
	clusterReport types.ClusterReport,
	rules []types.ReportItem,
) {
	err := storageImpl.WriteReportForCluster(orgID, clusterName, clusterReport,
		rules, time.Now(), time.Now(), time.Now(), testdata.RequestID1)
	helpers.FailOnError(t, err)
}

// TestNewStorageError checks whether constructor for new storage returns error for improper storage configuration
func TestNewStorageError(t *testing.T) {
	_, err := storage.NewOCPRecommendationsStorage(storage.Configuration{
		Driver: "non existing driver",
		Type:   "sql",
	})
	assert.EqualError(t, err, "driver non existing driver is not supported")
}

// TestNewStorageNoType checks whether constructor for new storage returns error for improper storage configuration
func TestNewStorageNoType(t *testing.T) {
	_, err := storage.NewOCPRecommendationsStorage(storage.Configuration{
		Driver: "non existing driver",
	})
	assert.EqualError(t, err, "Unknown storage type ''")
}

// TestNewStorageWrongType checks whether constructor for new storage returns error for improper storage configuration
func TestNewStorageWrongType(t *testing.T) {
	_, err := storage.NewOCPRecommendationsStorage(storage.Configuration{
		Driver: "non existing driver",
		Type:   "foobar",
	})
	assert.EqualError(t, err, "Unknown storage type 'foobar'")
}

// TestNewStorageWithLogging tests creating new storage with logs
func TestNewStorageWithLoggingError(t *testing.T) {
	s, _ := storage.NewOCPRecommendationsStorage(storage.Configuration{
		Driver:        "postgres",
		PGPort:        1234,
		PGUsername:    "user",
		LogSQLQueries: true,
		Type:          "sql",
	})

	err := s.Init()
	assert.Contains(t, err.Error(), "connect: connection refused")
}

// TestNewStorageReturnedImplementation check what implementation of storage is returnd
func TestNewStorageReturnedImplementation(t *testing.T) {
	s, _ := storage.NewOCPRecommendationsStorage(storage.Configuration{
		Driver:        "postgres",
		PGPort:        1234,
		PGUsername:    "user",
		LogSQLQueries: true,
		Type:          "sql",
	})
	assert.IsType(t, &storage.OCPRecommendationsDBStorage{}, s)

	s, _ = storage.NewOCPRecommendationsStorage(storage.Configuration{
		Driver:        "postgres",
		PGPort:        1234,
		PGUsername:    "user",
		LogSQLQueries: true,
		Type:          "redis",
	})
	assert.IsType(t, &storage.RedisStorage{}, s)

	s, _ = storage.NewOCPRecommendationsStorage(storage.Configuration{
		Driver:        "postgres",
		PGPort:        1234,
		PGUsername:    "user",
		LogSQLQueries: true,
		Type:          "noop",
	})
	assert.IsType(t, &storage.NoopOCPStorage{}, s)
}

// TestDBStorageReadReportForClusterEmptyTable check the behaviour of method ReadReportForCluster
func TestDBStorageReadReportForClusterEmptyTable(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	_, _, _, _, err := mockStorage.ReadReportForCluster(testdata.OrgID, testdata.ClusterName)
	if _, ok := err.(*types.ItemNotFoundError); err == nil || !ok {
		t.Fatalf("expected ItemNotFoundError, got %T, %+v", err, err)
	}

	assert.EqualError(
		t,
		err,
		fmt.Sprintf(
			"Item with ID %v/%v was not found in the storage",
			testdata.OrgID, testdata.ClusterName,
		),
	)
}

// TestDBStorageReadReportForClusterClosedStorage check the behaviour of method ReadReportForCluster
func TestDBStorageReadReportForClusterClosedStorage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	// we need to close storage right now
	closer()

	_, _, _, _, err := mockStorage.ReadReportForCluster(testdata.OrgID, testdata.ClusterName)
	assert.EqualError(t, err, "sql: database is closed")
}

// TestDBStorageReadReportForCluster check the behaviour of method ReadReportForCluster
func TestDBStorageReadReportForCluster(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	writeReportForCluster(t, mockStorage, testdata.OrgID, testdata.ClusterName, `{"report":{}}`, testdata.ReportEmptyRulesParsed)
	checkReportForCluster(t, mockStorage, testdata.OrgID, testdata.ClusterName, nil)
}

// TestDBStorageGetOrgIDByClusterID check the behaviour of method GetOrgIDByClusterID
func TestDBStorageGetOrgIDByClusterID(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	writeReportForCluster(t, mockStorage, testdata.OrgID, testdata.ClusterName, `{"report":{}}`, testdata.ReportEmptyRulesParsed)
	orgID, err := mockStorage.GetOrgIDByClusterID(testdata.ClusterName)
	helpers.FailOnError(t, err)

	assert.Equal(t, orgID, testdata.OrgID)
}

func TestDBStorageGetOrgIDByClusterID_Error(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, false)
	defer closer()

	dbStorage := mockStorage.(*storage.OCPRecommendationsDBStorage)
	connection := dbStorage.GetConnection()

	query := `
		CREATE TABLE report (
			org_id          VARCHAR  NOT NULL,
			cluster         VARCHAR  NOT NULL,
			report          VARCHAR NOT NULL,
			reported_at     TIMESTAMP,
			last_checked_at TIMESTAMP,
			kafka_offset BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY(org_id, cluster)
		);
	`

	// create a table with a bad type for org_id
	_, err := connection.Exec(query)
	helpers.FailOnError(t, err)

	// insert some data
	_, err = connection.Exec(`
		INSERT INTO report (org_id, cluster, report, reported_at, last_checked_at)
		VALUES ($1, $2, $3, $4, $5);
	`, "not-int", testdata.ClusterName, testdata.ClusterReportEmpty, time.Now(), time.Now())
	helpers.FailOnError(t, err)

	_, err = mockStorage.GetOrgIDByClusterID(testdata.ClusterName)
	assert.EqualError(
		t,
		err,
		`sql: Scan error on column index 0, name "org_id": `+
			`converting driver.Value type string ("not-int") to a uint32: invalid syntax`,
	)
}

// TestDBStorageGetOrgIDByClusterIDFailing check the behaviour of method GetOrgIDByClusterID for not existed ClusterID
func TestDBStorageGetOrgIDByClusterIDFailing(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	orgID, err := mockStorage.GetOrgIDByClusterID(testdata.ClusterName)
	assert.EqualError(t, err, "sql: no rows in result set")
	assert.Equal(t, orgID, types.OrgID(0))
}

// TestDBStorageReadReportNoTable check the behaviour of method ReadReportForCluster
// when the table with results does not exist
func TestDBStorageReadReportNoTable(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, false)
	defer closer()

	_, _, _, _, err := mockStorage.ReadReportForCluster(testdata.OrgID, testdata.ClusterName)
	assert.EqualError(t, err, "no such table: report")
}

// TestDBStorageWriteReportForClusterClosedStorage check the behaviour of method WriteReportForCluster
func TestDBStorageWriteReportForClusterClosedStorage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	// we need to close storage right now
	closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.ClusterReportEmpty,
		testdata.ReportEmptyRulesParsed,
		time.Now(),
		time.Now(),
		time.Now(),
		testdata.RequestID1,
	)
	assert.EqualError(t, err, "sql: database is closed")
}

// TestDBStorageWriteReportForClusterClosedStorage check the behaviour of method WriteReportForCluster
func TestDBStorageWriteReportForClusterUnsupportedDriverError(t *testing.T) {
	fakeStorage := storage.NewOCPRecommendationsFromConnection(nil, -1)
	// no need to close it

	err := fakeStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.ClusterReportEmpty,
		testdata.ReportEmptyRulesParsed,
		time.Now(),
		time.Now(),
		time.Now(),
		testdata.RequestID1,
	)
	assert.EqualError(t, err, "writing report with DB -1 is not supported")
}

// TestDBStorageWriteReportForClusterMoreRecentInDB checks that older report
// will not replace a more recent one when writing a report to storage.
func TestDBStorageWriteReportForClusterMoreRecentInDB(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	newerTime := time.Now().UTC()
	olderTime := newerTime.Add(-time.Hour)

	// Insert newer report.
	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.ClusterReportEmpty,
		testdata.ReportEmptyRulesParsed,
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
		testdata.ReportEmptyRulesParsed,
		olderTime,
		time.Now(),
		time.Now(),
		testdata.RequestID1,
	)
	assert.Equal(t, types.ErrOldReport, err)
}

// TestDBStorageClusterOrgTransfer checks the behaviour of report storage in case of cluster org transfer
func TestDBStorageClusterOrgTransfer(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	cluster1ID, cluster2ID := types.ClusterName("aaaaaaaa-1234-cccc-dddd-eeeeeeeeeeee"),
		types.ClusterName("aaaaaaaa-1234-5678-dddd-eeeeeeeeeeee")

	writeReportForCluster(t, mockStorage, testdata.OrgID, cluster1ID, testdata.ClusterReportEmpty, testdata.ReportEmptyRulesParsed)
	writeReportForCluster(t, mockStorage, testdata.Org2ID, cluster2ID, testdata.ClusterReportEmpty, testdata.ReportEmptyRulesParsed)

	result, err := mockStorage.ListOfOrgs()
	helpers.FailOnError(t, err)
	assert.ElementsMatch(t, []types.OrgID{testdata.OrgID, testdata.Org2ID}, result)

	// cluster1 owned by org1
	orgID, err := mockStorage.GetOrgIDByClusterID(cluster1ID)
	helpers.FailOnError(t, err)
	assert.Equal(t, orgID, testdata.OrgID)

	// cluster2 owned by org2
	orgID, err = mockStorage.GetOrgIDByClusterID(cluster2ID)
	helpers.FailOnError(t, err)
	assert.Equal(t, orgID, testdata.Org2ID)

	// org2 can read cluster2
	_, _, _, _, err = mockStorage.ReadReportForCluster(testdata.Org2ID, cluster2ID)
	helpers.FailOnError(t, err)

	// "org transfer"
	writeReportForCluster(t, mockStorage, testdata.OrgID, cluster2ID, testdata.ClusterReportEmpty, testdata.ReportEmptyRulesParsed)

	// only 1 org now
	result, err = mockStorage.ListOfOrgs()
	helpers.FailOnError(t, err)
	assert.ElementsMatch(t, []types.OrgID{testdata.OrgID}, result)

	// cluster2 owned by org1
	orgID, err = mockStorage.GetOrgIDByClusterID(cluster2ID)
	helpers.FailOnError(t, err)
	assert.Equal(t, orgID, testdata.OrgID)

	// org2 can no longer read cluster2
	_, _, _, _, err = mockStorage.ReadReportForCluster(testdata.Org2ID, cluster2ID)
	assert.NotNil(t, err)

	// org1 can now  read cluster2
	_, _, _, _, err = mockStorage.ReadReportForCluster(testdata.OrgID, cluster2ID)
	helpers.FailOnError(t, err)
}

// TestDBStorageWriteReportForClusterDroppedReportTable checks the error
// returned when trying to SELECT from a dropped/missing report table.
func TestDBStorageWriteReportForClusterDroppedReportTable(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	connection := storage.GetConnection(mockStorage.(*storage.OCPRecommendationsDBStorage))

	query := "DROP TABLE report CASCADE;"

	_, err := connection.Exec(query)
	helpers.FailOnError(t, err)

	err = mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.ClusterReportEmpty,
		testdata.ReportEmptyRulesParsed, time.Now(), time.Now(), time.Now(),
		testdata.RequestID1,
	)
	assert.EqualError(t, err, "no such table: report")
}

func TestDBStorageWriteReportForClusterExecError(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, false)
	defer closer()

	createReportTableWithBadClusterField(t, mockStorage)

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules,
		testdata.Report3RulesParsed, testdata.LastCheckedAt, time.Now(), time.Now(),
		testdata.RequestID1,
	)
	assert.Error(t, err)

	const postgresErrMessage = "pq: invalid input syntax"
	if !strings.HasPrefix(err.Error(), postgresErrMessage) {
		t.Fatalf("expected: \n%v\ngot:\n%v", postgresErrMessage, err.Error())
	}
}

func TestDBStorageWriteReportForClusterFakePostgresOK(t *testing.T) {
	mockStorage, expects := ira_helpers.MustGetMockStorageWithExpectsForDriver(t, types.DBDriverPostgres)
	defer ira_helpers.MustCloseMockStorageWithExpects(t, mockStorage, expects)

	expects.ExpectBegin()

	expects.ExpectQuery(`SELECT last_checked_at FROM report`).
		WillReturnRows(expects.NewRows([]string{"last_checked_at"})).
		RowsWillBeClosed()

	expects.ExpectQuery("SELECT rule_fqdn, error_key, created_at FROM rule_hit").
		WillReturnRows(expects.NewRows([]string{"last_checked_at"})).
		RowsWillBeClosed()
	expects.ExpectExec("DELETE FROM rule_hit").
		WillReturnResult(driver.ResultNoRows)

	// one INSERT with multiple rows is expected
	expects.ExpectExec("INSERT INTO rule_hit").
		WillReturnResult(driver.ResultNoRows)

	expects.ExpectExec("INSERT INTO report").
		WillReturnResult(driver.ResultNoRows)

	expects.ExpectCommit()
	expects.ExpectClose()
	expects.ExpectClose()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules,
		testdata.Report3RulesParsed, testdata.LastCheckedAt, time.Now(), time.Now(),
		testdata.RequestID1)
	helpers.FailOnError(t, mockStorage.Close())
	helpers.FailOnError(t, err)
}

// TestDBStorageListOfOrgs check the behaviour of method ListOfOrgs
func TestDBStorageListOfOrgs(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	writeReportForCluster(t, mockStorage, 1, "1deb586c-fb85-4db4-ae5b-139cdbdf77ae", testdata.ClusterReportEmpty, testdata.ReportEmptyRulesParsed)
	writeReportForCluster(t, mockStorage, 3, "a1bf5b15-5229-4042-9825-c69dc36b57f5", testdata.ClusterReportEmpty, testdata.ReportEmptyRulesParsed)

	result, err := mockStorage.ListOfOrgs()
	helpers.FailOnError(t, err)

	assert.ElementsMatch(t, []types.OrgID{1, 3}, result)
}

func TestDBStorageListOfOrgsNoTable(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, false)
	defer closer()

	_, err := mockStorage.ListOfOrgs()
	assert.EqualError(t, err, "no such table: report")
}

// TestDBStorageListOfOrgsClosedStorage check the behaviour of method ListOfOrgs
func TestDBStorageListOfOrgsClosedStorage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	// we need to close storage right now
	closer()

	_, err := mockStorage.ListOfOrgs()
	assert.EqualError(t, err, "sql: database is closed")
}

// TestDBStorageListOfClustersFor check the behaviour of method ListOfClustersForOrg
func TestDBStorageListOfClustersForOrg(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	cluster1ID, cluster2ID, cluster3ID := testdata.GetRandomClusterID(), testdata.GetRandomClusterID(), testdata.GetRandomClusterID()
	// writeReportForCluster writes the report at time.Now()
	writeReportForCluster(t, mockStorage, testdata.OrgID, cluster1ID, testdata.ClusterReportEmpty, testdata.ReportEmptyRulesParsed)
	writeReportForCluster(t, mockStorage, testdata.OrgID, cluster2ID, testdata.ClusterReportEmpty, testdata.ReportEmptyRulesParsed)

	// also pushing cluster for different org
	writeReportForCluster(t, mockStorage, testdata.Org2ID, cluster3ID, testdata.ClusterReportEmpty, testdata.ReportEmptyRulesParsed)

	result, err := mockStorage.ListOfClustersForOrg(testdata.OrgID, time.Now().Add(-time.Hour))
	helpers.FailOnError(t, err)

	assert.ElementsMatch(t, []types.ClusterName{
		cluster1ID,
		cluster2ID,
	}, result)

	result, err = mockStorage.ListOfClustersForOrg(testdata.Org2ID, time.Now().Add(-time.Hour))
	helpers.FailOnError(t, err)

	assert.Equal(t, []types.ClusterName{cluster3ID}, result)
}

func TestDBStorageListOfClustersTimeLimit(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	// writeReportForCluster writes the report at time.Now()
	cluster1ID, cluster2ID := testdata.GetRandomClusterID(), testdata.GetRandomClusterID()
	// writeReportForCluster writes the report at time.Now()
	writeReportForCluster(t, mockStorage, testdata.OrgID, cluster1ID, testdata.ClusterReportEmpty, testdata.ReportEmptyRulesParsed)
	writeReportForCluster(t, mockStorage, testdata.OrgID, cluster2ID, testdata.ClusterReportEmpty, testdata.ReportEmptyRulesParsed)

	// since we can't easily change reported_at without changing the core source code, let's make a request from the "future"
	// fetch org overview with T+2h
	result, err := mockStorage.ListOfClustersForOrg(testdata.OrgID, time.Now().Add(time.Hour*2))
	helpers.FailOnError(t, err)

	// must fetch nothing
	// assert.ElementsMatch(t, []types.ClusterName{}, result)
	assert.Empty(t, result)

	// request with T-2h
	result, err = mockStorage.ListOfClustersForOrg(testdata.OrgID, time.Now().Add(-time.Hour*2))
	helpers.FailOnError(t, err)

	// must fetch all reports
	assert.ElementsMatch(t, []types.ClusterName{
		cluster1ID,
		cluster2ID,
	}, result)
}

func TestDBStorageListOfClustersNoTable(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, false)
	defer closer()

	_, err := mockStorage.ListOfClustersForOrg(5, time.Now().Add(-time.Hour))
	assert.EqualError(t, err, "no such table: report")
}

// TestDBStorageListOfClustersClosedStorage check the behaviour of method ListOfOrgs
func TestDBStorageListOfClustersClosedStorage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	// we need to close storage right now
	closer()

	_, err := mockStorage.ListOfClustersForOrg(5, time.Now().Add(-time.Hour))
	assert.EqualError(t, err, "sql: database is closed")
}

// TestMockDBReportsCount check the behaviour of method ReportsCount
func TestMockDBReportsCount(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	assertNumberOfReports(t, mockStorage, 0)

	writeReportForCluster(t, mockStorage, 5, "4016d01b-62a1-4b49-a36e-c1c5a3d02750", testdata.ClusterReportEmpty, testdata.ReportEmptyRulesParsed)

	assertNumberOfReports(t, mockStorage, 1)
}

func TestMockDBReportsCountNoTable(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, false)
	defer closer()

	_, err := mockStorage.ReportsCount()
	assert.EqualError(t, err, "no such table: report")
}

func TestMockDBReportsCountClosedStorage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, false)
	// we need to close storage right now
	closer()

	_, err := mockStorage.ReportsCount()
	assert.EqualError(t, err, "sql: database is closed")
}

func TestDBStorageNewPostgresqlError(t *testing.T) {
	s, _ := storage.NewOCPRecommendationsStorage(storage.Configuration{
		Driver:     "postgres",
		PGHost:     "non-existing-host",
		PGPort:     12345,
		PGUsername: "user",
		Type:       "sql",
	})

	err := s.Init()
	assert.Contains(t, err.Error(), "non-existing-host")
}

func mustWriteReport(
	t *testing.T,
	connection *sql.DB,
	orgID interface{},
	clusterName interface{},
	clusterReport interface{},
) {
	query := `
		INSERT INTO report(org_id, cluster, report, reported_at, last_checked_at)
		VALUES ($1, $2, $3, $4, $5);
	`

	statement, err := connection.Prepare(query)
	helpers.FailOnError(t, err)

	_, err = statement.Exec(
		orgID,
		clusterName,
		clusterReport,
		time.Now(),
		time.Now(),
	)
	helpers.FailOnError(t, err)

	err = statement.Close()
	helpers.FailOnError(t, err)
}

func TestDBStorageListOfOrgsLogError(t *testing.T) {
	buf := new(bytes.Buffer)
	log.Logger = zerolog.New(buf)

	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	connection := storage.GetConnection(mockStorage.(*storage.OCPRecommendationsDBStorage))
	// write illegal negative org_id
	mustWriteReport(t, connection, -1, testdata.ClusterName, testdata.ClusterReportEmpty)

	_, err := mockStorage.ListOfOrgs()
	helpers.FailOnError(t, err)

	assert.Contains(t, buf.String(), "sql: Scan error")
}

func TestDBStorageCloseError(t *testing.T) {
	const errString = "unable to close the database"

	mockStorage, expects := ira_helpers.MustGetMockStorageWithExpects(t)

	expects.ExpectClose().WillReturnError(fmt.Errorf(errString))
	err := mockStorage.Close()

	assert.EqualError(t, err, errString)
}

func TestDBStorageListOfClustersForOrgScanError(t *testing.T) {
	// just for the coverage, because this error can't happen ever because we use
	// not null in table creation
	buf := new(bytes.Buffer)
	log.Logger = zerolog.New(buf)

	mockStorage, expects := ira_helpers.MustGetMockStorageWithExpects(t)
	defer ira_helpers.MustCloseMockStorageWithExpects(t, mockStorage, expects)

	expects.ExpectQuery("SELECT cluster FROM report").WillReturnRows(
		sqlmock.NewRows([]string{"cluster"}).AddRow(nil),
	)

	_, err := mockStorage.ListOfClustersForOrg(testdata.OrgID, time.Now().Add(-time.Hour))
	helpers.FailOnError(t, err)

	assert.Contains(t, buf.String(), "converting NULL to string is unsupported")
}

func TestDBStorageDeleteReports(t *testing.T) {
	for _, functionName := range []string{
		"DeleteReportsForOrg", "DeleteReportsForCluster",
	} {
		func() {
			mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
			defer closer()
			assertNumberOfReports(t, mockStorage, 0)

			err := mockStorage.WriteReportForCluster(
				testdata.OrgID,
				testdata.ClusterName,
				testdata.Report3Rules,
				testdata.Report3RulesParsed,
				testdata.LastCheckedAt,
				time.Now(),
				time.Now(),
				testdata.RequestID1,
			)
			helpers.FailOnError(t, err)

			assertNumberOfReports(t, mockStorage, 1)

			switch functionName {
			case "DeleteReportsForOrg":
				err = mockStorage.DeleteReportsForOrg(testdata.OrgID)
			case "DeleteReportsForCluster":
				err = mockStorage.DeleteReportsForCluster(testdata.ClusterName)
			default:
				t.Fatal(fmt.Errorf("unexpected function name"))
			}
			helpers.FailOnError(t, err)

			assertNumberOfReports(t, mockStorage, 0)
		}()
	}
}

func TestDBStorage_ReadReportForClusterByClusterName_OK(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	mustWriteReport3Rules(t, mockStorage)

	report, lastCheckedAt, err := mockStorage.ReadReportForClusterByClusterName(testdata.ClusterName)
	helpers.FailOnError(t, err)

	for i, v := range testdata.RuleOnReportResponses {
		helpers.CompareReportResponses(t, v, report[i], time.Now())
	}
	assert.Equal(t, types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)), lastCheckedAt)
}

func TestDBStorage_CheckIfClusterExists_ClusterDoesNotExist(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	_, _, err := mockStorage.ReadReportForClusterByClusterName(testdata.ClusterName)
	assert.EqualError(
		t,
		err,
		fmt.Sprintf("Item with ID %v was not found in the storage", testdata.ClusterName),
	)
}

func TestDBStorage_CheckIfClusterExists_DBError(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	closer()

	_, _, err := mockStorage.ReadReportForClusterByClusterName(testdata.ClusterName)
	assert.EqualError(t, err, "sql: database is closed")
}

func TestDBStorageWriteConsumerError(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	testTopic := "topic"
	var testPartition int32 = 2
	var testOffset int64 = 10
	testKey := []byte("key")
	testMessage := []byte("value")
	testProducedAt := time.Now().Add(-time.Hour).UTC()
	testError := fmt.Errorf("Consumer error")

	err := mockStorage.WriteConsumerError(&sarama.ConsumerMessage{
		Topic:     testTopic,
		Partition: testPartition,
		Offset:    testOffset,
		Key:       testKey,
		Value:     testMessage,
		Timestamp: testProducedAt,
	}, testError)

	assert.NoError(t, err)

	conn := storage.GetConnection(mockStorage.(*storage.OCPRecommendationsDBStorage))
	row := conn.QueryRow(`
		SELECT key, message, produced_at, consumed_at, error
		  FROM consumer_error
		 WHERE topic = $1 AND partition = $2 AND topic_offset = $3
	`, testTopic, testPartition, testOffset)

	var storageKey []byte
	var storageMessage []byte
	var storageProducedAt time.Time
	var storageConsumedAt time.Time
	var storageError string
	err = row.Scan(&storageKey, &storageMessage, &storageProducedAt, &storageConsumedAt, &storageError)
	assert.NoError(t, err)

	assert.Equal(t, testKey, storageKey)
	assert.Equal(t, testMessage, storageMessage)
	assert.Equal(t, testProducedAt.Unix(), storageProducedAt.Unix())
	assert.True(t, time.Now().UTC().After(storageConsumedAt))
	assert.Equal(t, testError.Error(), storageError)
}

func TestDBStorage_Init(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	dbStorage := mockStorage.(*storage.OCPRecommendationsDBStorage)

	err := dbStorage.MigrateToLatest()
	helpers.FailOnError(t, err)

	mustWriteReport3Rules(t, mockStorage)

	err = mockStorage.Init()
	helpers.FailOnError(t, err)

	clustersLastChecked := storage.GetClustersLastChecked(dbStorage)

	assert.Len(t, clustersLastChecked, 1)
	assert.Equal(t, testdata.LastCheckedAt.Unix(), clustersLastChecked[testdata.ClusterName].Unix())
}

func TestDBStorage_Init_Error(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, false)
	defer closer()

	createReportTableWithBadClusterField(t, mockStorage)

	connection := storage.GetConnection(mockStorage.(*storage.OCPRecommendationsDBStorage))

	// create a table with a bad type
	_, err := connection.Exec(`
		INSERT INTO report (org_id, cluster, report, reported_at, last_checked_at)
		VALUES($1, $2, $3, $4, $5)
	`, testdata.OrgID, 1, testdata.ClusterReportEmpty, time.Now(), time.Now())
	helpers.FailOnError(t, err)

	err = mockStorage.Init()
	assert.EqualError(
		t,
		err,
		`sql: Scan error on column index 0, name "cluster": `+
			`unsupported Scan, storing driver.Value type int64 into type *types.ClusterName`,
	)
}

func createReportTableWithBadClusterField(t *testing.T, mockStorage storage.OCPRecommendationsStorage) {
	connection := storage.GetConnection(mockStorage.(*storage.OCPRecommendationsDBStorage))
	query := `
		CREATE TABLE report (
			org_id          INTEGER NOT NULL,
			cluster         INTEGER NOT NULL UNIQUE,
			report          VARCHAR NOT NULL,
			reported_at     TIMESTAMP,
			last_checked_at TIMESTAMP,
			kafka_offset    BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY(org_id, cluster)
		)
	`

	// create a table with a bad type
	_, err := connection.Exec(query)
	helpers.FailOnError(t, err)

	query = `
		CREATE TABLE rule_hit (
			org_id          INTEGER NOT NULL,
			cluster_id      VARCHAR NOT NULL,
			rule_fqdn       VARCHAR NOT NULL,
			error_key       VARCHAR NOT NULL,
			template_data   VARCHAR NOT NULL,
			PRIMARY KEY(cluster_id, org_id, rule_fqdn, error_key)
		)
	`

	_, err = connection.Exec(query)
	helpers.FailOnError(t, err)
}

// TestConstructInClausule checks the helper function constructInClausule
func TestConstructInClausule(t *testing.T) {
	_, err := storage.ConstructInClausule(0)
	assert.NotNil(t, err)
	assert.EqualErrorf(t, err, err.Error(), "at least one value needed")

	c1, err := storage.ConstructInClausule(1)
	assert.Equal(t, c1, "$1")
	helpers.FailOnError(t, err)

	c2, err := storage.ConstructInClausule(2)
	assert.Equal(t, c2, "$1,$2")
	helpers.FailOnError(t, err)

	c3, err := storage.ConstructInClausule(3)
	assert.Equal(t, c3, "$1,$2,$3")
	helpers.FailOnError(t, err)
}

// TestArgsWithClusterNames checks the helper function argsWithClusterNames
func TestArgsWithClusterNames(t *testing.T) {
	cn0 := []types.ClusterName{}
	args0 := storage.ArgsWithClusterNames(cn0)
	assert.Equal(t, len(args0), 0)

	cn1 := []types.ClusterName{"aaa"}
	args1 := storage.ArgsWithClusterNames(cn1)
	assert.Equal(t, len(args1), 1)
	assert.Equal(t, args1[0], types.ClusterName("aaa"))

	cn2 := []types.ClusterName{"aaa", "bbb"}
	args2 := storage.ArgsWithClusterNames(cn2)
	assert.Equal(t, len(args2), 2)
	assert.Equal(t, args2[0], types.ClusterName("aaa"))
	assert.Equal(t, args2[1], types.ClusterName("bbb"))
}

// TestDBStorageReadReportsForClusters1 check the behaviour of method
// ReadReportForClusters
func TestDBStorageReadReportsForClusters1(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	writeReportForCluster(t, mockStorage, testdata.OrgID, testdata.ClusterName, `{"report":{}}`, testdata.ReportEmptyRulesParsed)

	// try to read reports for clusters
	cn1 := []types.ClusterName{"not-a-cluster"}
	results, err := mockStorage.ReadReportsForClusters(cn1)
	helpers.FailOnError(t, err)

	// and check the read report with expected one
	assert.Equal(t, len(results), 0)
}

// TestDBStorageReadReportsForClusters2 check the behaviour of method
// ReadReportForClusters
func TestDBStorageReadReportsForClusters2(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	writeReportForCluster(t, mockStorage, testdata.OrgID, testdata.ClusterName, `{"report":{}}`, testdata.ReportEmptyRulesParsed)

	// try to read reports for clusters
	cn1 := []types.ClusterName{testdata.ClusterName}
	results, err := mockStorage.ReadReportsForClusters(cn1)
	helpers.FailOnError(t, err)

	// and check the read report with expected one
	assert.Equal(t, len(results), 1)
}

// TestDBStorageReadReportsForClusters3 check the behaviour of method
// ReadReportForClusters
func TestDBStorageReadReportsForClusters3(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	writeReportForCluster(t, mockStorage, testdata.OrgID, testdata.ClusterName, `{"report":{}}`, testdata.ReportEmptyRulesParsed)

	// try to read reports for clusters
	noClusters := []types.ClusterName{}
	results, err := mockStorage.ReadReportsForClusters(noClusters)

	// previously would fail on the DB due to improper SQL syntax, now returns empty list
	helpers.FailOnError(t, err)

	// and check the read report with expected one
	assert.Equal(t, len(results), 0)
}

// TestDBStorageReadOrgIDsForClusters0_Reproducer reproduces a bug caused by improper in clause handling
func TestDBStorageReadOrgIDsForClusters0_Reproducer(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	writeReportForCluster(t, mockStorage, testdata.OrgID, testdata.ClusterName, `{"report":{}}`, testdata.ReportEmptyRulesParsed)

	// try to read org IDs for clusters
	noClusters := []types.ClusterName{}
	results, err := mockStorage.ReadOrgIDsForClusters(noClusters)

	// previously would fail on the DB due to improper SQL syntax, now returns empty list
	helpers.FailOnError(t, err)

	// and check the read report with expected one
	assert.Equal(t, len(results), 0)
}

// TestDBStorageReadOrgIDsForClusters1 check the behaviour of method
// ReadOrgIDsForClusters
func TestDBStorageReadOrgIDsForClusters1(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	writeReportForCluster(t, mockStorage, testdata.OrgID, testdata.ClusterName, `{"report":{}}`, testdata.ReportEmptyRulesParsed)

	// try to read org IDs for clusters
	cn1 := []types.ClusterName{"not-a-cluster"}
	results, err := mockStorage.ReadOrgIDsForClusters(cn1)
	helpers.FailOnError(t, err)

	// and check the read report with expected one
	assert.Equal(t, len(results), 0)
}

// TestDBStorageReadOrgIDsForClusters2 check the behaviour of method
// ReadOrgIDsForClusters
func TestDBStorageReadOrgIDsForClusters2(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	writeReportForCluster(t, mockStorage, testdata.OrgID, testdata.ClusterName, `{"report":{}}`, testdata.ReportEmptyRulesParsed)

	// try to read org IDs for clusters
	cn1 := []types.ClusterName{testdata.ClusterName}
	results, err := mockStorage.ReadOrgIDsForClusters(cn1)
	helpers.FailOnError(t, err)

	// and check the read report with expected one
	assert.Equal(t, len(results), 1)
	assert.Equal(t, results[0], testdata.OrgID)
}

// TestDBStorageWriteRecommendationsForClusterClosedStorage check the behaviour of method WriteRecommendationsForCluster
func TestDBStorageWriteRecommendationsForClusterClosedStorage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, false)
	// we need to close storage right now
	closer()
	err := mockStorage.WriteRecommendationsForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.ClusterReportEmpty,
		"",
	)
	assert.EqualError(t, err, "sql: database is closed")
}

// TestDBStorageWriteRecommendationForClusterNoConflict checks that
// recommendation is inserted correctly if there is not a more recent
// one stored.
func TestDBStorageWriteRecommendationForClusterNoConflict(t *testing.T) {
	mockStorage, expects := ira_helpers.MustGetMockStorageWithExpectsForDriver(t, types.DBDriverPostgres)
	defer ira_helpers.MustCloseMockStorageWithExpects(t, mockStorage, expects)

	expects.ExpectBegin()

	expects.ExpectExec("INSERT INTO recommendation").
		WillReturnResult(driver.ResultNoRows)

	expects.ExpectCommit()

	err := mockStorage.WriteRecommendationsForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, RecommendationCreatedAtTimestamp,
	)

	helpers.FailOnError(t, err)
}

// TestDBStorageInsertRecommendations checks that only hitting rules in the report
// are inserted in the recommendation table
func TestDBStorageInsertRecommendations(t *testing.T) {
	mockStorage, expects := ira_helpers.MustGetMockStorageWithExpectsForDriver(t, types.DBDriverPostgres)
	defer ira_helpers.MustCloseMockStorageWithExpects(t, mockStorage, expects)

	expects.ExpectBegin()

	expects.ExpectExec("INSERT INTO recommendation \\(org_id, cluster_id, rule_fqdn, error_key, rule_id, created_at, impacted_since\\) " +
		"VALUES \\(\\$1, \\$2, \\$3, \\$4, \\$5, \\$6, \\$7\\)," +
		"\\(\\$8, \\$9, \\$10, \\$11, \\$12, \\$13, \\$14\\)," +
		"\\(\\$15, \\$16, \\$17, \\$18, \\$19, \\$20, \\$21\\)").
		WillReturnResult(driver.ResultNoRows)

	expects.ExpectCommit()

	report := types.ReportRules{
		HitRules:     testdata.RuleOnReportResponses,
		SkippedRules: testdata.RuleOnReportResponses,
		PassedRules:  testdata.RuleOnReportResponses,
		TotalCount:   3 * len(testdata.RuleOnReportResponses),
	}
	// impactedSince first time a recommendation is inserted impacted
	// and created_at match
	impactedSince := RecommendationImpactedSinceMap
	inserted, err := storage.InsertRecommendations(
		mockStorage.(*storage.OCPRecommendationsDBStorage),
		testdata.OrgID, testdata.ClusterName, report,
		RecommendationCreatedAtTimestamp, impactedSince)
	assert.Equal(t, 3, inserted)
	helpers.FailOnError(t, err)
}

// TestDBStorageWriteRecommendationForClusterAlreadyStored checks that
// recommendation is not inserted if there is already one
// recommendation stored with the same values
func TestDBStorageWriteRecommendationForClusterAlreadyStored(t *testing.T) {
	mockStorage, expects := ira_helpers.MustGetMockStorageWithExpectsForDriver(t, types.DBDriverPostgres)
	defer ira_helpers.MustCloseMockStorageWithExpects(t, mockStorage, expects)

	expects.ExpectBegin()
	expects.ExpectExec("INSERT INTO recommendation").
		WillReturnResult(driver.ResultNoRows)
	expects.ExpectCommit()

	err := mockStorage.WriteRecommendationsForCluster(
		testdata.OrgID, testdata.ClusterName,
		testdata.Report3Rules,
		RecommendationCreatedAtTimestamp,
	)
	helpers.FailOnError(t, err)

	expects.ExpectBegin()
	expects.ExpectExec("INSERT INTO recommendation").
		WillReturnError(fmt.Errorf("Unable to insert the recommendation for cluster: " + string(testdata.ClusterName)))
	expects.ExpectRollback()

	err = mockStorage.WriteRecommendationsForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, RecommendationCreatedAtTimestamp,
	)

	assert.Error(t, err, "")
}

// TestDBStorageWriteRecommendationForClusterAlreadyStoredAndDeleted checks that
// recommendation is inserted correctly if there is already one
// recommendation stored with the same values, and existing recommendation
// is not deleted
func TestDBStorageWriteRecommendationForClusterAlreadyStoredAndDeleted(t *testing.T) {
	mockStorage, expects := ira_helpers.MustGetMockStorageWithExpectsForDriver(t, types.DBDriverPostgres)
	defer ira_helpers.MustCloseMockStorageWithExpects(t, mockStorage, expects)

	expects.ExpectBegin()
	expects.ExpectExec("INSERT INTO recommendation").
		WillReturnResult(driver.ResultNoRows)
	expects.ExpectCommit()

	err := mockStorage.WriteRecommendationsForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, RecommendationCreatedAtTimestamp,
	)

	helpers.FailOnError(t, err)

	// Need to update clustersLastChecked as would be done on init and in writeReportForClusters
	dbStorage := mockStorage.(*storage.OCPRecommendationsDBStorage)
	storage.SetClustersLastChecked(dbStorage, testdata.ClusterName, time.Now())

	expects.ExpectBegin()
	expects.ExpectQuery(`SELECT rule_fqdn, error_key, impacted_since FROM recommendation`).
		WillReturnRows(expects.NewRows([]string{"created_at"}).AddRow(time.Time{})).
		RowsWillBeClosed()
	expects.ExpectExec("DELETE FROM recommendation").
		WillReturnResult(driver.RowsAffected(3))
	expects.ExpectExec("INSERT INTO recommendation").
		WillReturnResult(driver.ResultNoRows)
	expects.ExpectCommit()
	expects.ExpectClose()
	expects.ExpectClose()

	err = mockStorage.WriteRecommendationsForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, RecommendationCreatedAtTimestamp,
	)
	helpers.FailOnError(t, err)
	helpers.FailOnError(t, mockStorage.Close())
}

// TestDBStorageInsertRecommendationsNoRuleHit checks that no
// recommendations are inserted if there is no rule hits in the report
func TestDBStorageInsertRecommendationsNoRuleHit(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	report := types.ReportRules{
		SkippedRules: testdata.RuleOnReportResponses,
		PassedRules:  testdata.RuleOnReportResponses,
		TotalCount:   2 * len(testdata.RuleOnReportResponses),
	}
	// impactedSincefirst time a recommendation is inserted impacted and created_at match
	impactedSince := RecommendationImpactedSinceMap
	inserted, err := storage.InsertRecommendations(
		mockStorage.(*storage.OCPRecommendationsDBStorage), testdata.OrgID, testdata.ClusterName,
		report, RecommendationCreatedAtTimestamp, impactedSince)

	assert.Equal(t, 0, inserted)
	helpers.FailOnError(t, err)
}

func Test_CCXDEV_10244_DBStorageGetImpactedSinceMap(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	err := mockStorage.WriteRecommendationsForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, RecommendationCreatedAtTimestamp,
	)
	helpers.FailOnError(t, err)

	// let's retrieve the stored data and ensure it is what we want to work with
	res, err := mockStorage.ReadRecommendationsForClusters([]string{string(testdata.ClusterName)}, testdata.OrgID)
	helpers.FailOnError(t, err)

	expect := []types.ClusterName{testdata.ClusterName}
	expectResp := ctypes.RecommendationImpactedClusters{
		testdata.Rule1CompositeID: expect,
		testdata.Rule2CompositeID: expect,
		testdata.Rule3CompositeID: expect,
	}
	assert.Equal(t, expectResp, res)

	// We have 3 recommendations with different rule FQDN for this Org ID + Clustername
	// Therefore, we expect 3 entries in the RuleKeyCreatedAtMap

	// query used before fix.
	query := "SELECT rule_fqdn, error_key, created_at FROM recommendation WHERE org_id = $1 AND cluster_id = $2 LIMIT 1;"
	rkcMap, err := storage.GetRuleKeyCreatedAtMap(mockStorage.(*storage.OCPRecommendationsDBStorage), query, testdata.OrgID, testdata.ClusterName)
	helpers.FailOnError(t, err)
	assert.Equal(t, len(rkcMap), 1)

	//query used after fix, without limiting number of rows retrieved from the DB
	rkcMap, err = storage.GetRuleKeyCreatedAtMapForTable(mockStorage.(*storage.OCPRecommendationsDBStorage), "recommendation", testdata.OrgID, testdata.ClusterName)
	helpers.FailOnError(t, err)
	assert.Equal(t, len(rkcMap), 3)
}

// TestDBStorageReadRecommendationsForClusters checks that stored recommendations
// are retrieved correctly
func TestDBStorageReadRecommendationsForClusters(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	err := mockStorage.WriteRecommendationsForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, RecommendationCreatedAtTimestamp,
	)
	helpers.FailOnError(t, err)

	expected := []types.ClusterName{testdata.ClusterName}
	expect := ctypes.RecommendationImpactedClusters{
		testdata.Rule1CompositeID: expected,
		testdata.Rule2CompositeID: expected,
		testdata.Rule3CompositeID: expected,
	}

	res, err := mockStorage.ReadRecommendationsForClusters([]string{string(testdata.ClusterName)}, testdata.OrgID)
	helpers.FailOnError(t, err)

	assert.Equal(t, expect, res)
}

// TestDBStorageReadRecommendationsForClustersMoreClusters checks that stored recommendations
// for multiplpe clusters is calculated correctly
func TestDBStorageReadRecommendationsForClustersMoreClusters(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	clusterList := make([]string, 3)
	for i := range clusterList {
		clusterList[i] = string(testdata.GetRandomClusterID())
	}

	// cluster 0 has 0 rules == should not affect the results
	err := mockStorage.WriteRecommendationsForCluster(
		testdata.OrgID, types.ClusterName(clusterList[0]), testdata.Report0Rules, RecommendationCreatedAtTimestamp,
	)
	helpers.FailOnError(t, err)

	// cluster 1 has rule1 and rule2
	err = mockStorage.WriteRecommendationsForCluster(
		testdata.OrgID, types.ClusterName(clusterList[1]), testdata.Report2Rules, RecommendationCreatedAtTimestamp,
	)
	helpers.FailOnError(t, err)

	// cluster 2 has rule1 and rule2 and rule3
	err = mockStorage.WriteRecommendationsForCluster(
		testdata.OrgID, types.ClusterName(clusterList[2]), testdata.Report3Rules, RecommendationCreatedAtTimestamp,
	)
	helpers.FailOnError(t, err)

	expect2 := []types.ClusterName{types.ClusterName(clusterList[1]), types.ClusterName(clusterList[2])}
	expect := ctypes.RecommendationImpactedClusters{
		testdata.Rule1CompositeID: expect2,
		testdata.Rule2CompositeID: expect2,
		testdata.Rule3CompositeID: []types.ClusterName{types.ClusterName(clusterList[2])},
	}

	res, err := mockStorage.ReadRecommendationsForClusters(clusterList, testdata.OrgID)
	helpers.FailOnError(t, err)

	// Compare the results
	assert.Equal(t, len(expect), len(res), "Number of rules should match")
	for ruleID, expectedClusters := range expect {
		actualClusters, exists := res[ruleID]
		assert.True(t, exists, "Rule %s should exist in the result", ruleID)
		assert.ElementsMatch(t, expectedClusters, actualClusters, "Clusters for rule %s should match", ruleID)
	}
}

// TestDBStorageReadRecommendationsForClustersNoRecommendations checks that when no recommendations
// are stored, it is an OK state
func TestDBStorageReadRecommendationsForClustersNoRecommendations(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	err := mockStorage.WriteRecommendationsForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.ClusterReportEmpty, RecommendationCreatedAtTimestamp,
	)
	helpers.FailOnError(t, err)

	expect := ctypes.RecommendationImpactedClusters{}

	res, err := mockStorage.ReadRecommendationsForClusters([]string{string(testdata.ClusterName)}, testdata.OrgID)
	helpers.FailOnError(t, err)

	assert.Equal(t, expect, res)
}

// TestDBStorageReadRecommendationsForClustersEmptyList_Reproducer reproduces a bug caused by improper in clause handling
func TestDBStorageReadRecommendationsForClustersEmptyList_Reproducer(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	err := mockStorage.WriteRecommendationsForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.ClusterReportEmpty, RecommendationCreatedAtTimestamp,
	)
	helpers.FailOnError(t, err)

	expect := ctypes.RecommendationImpactedClusters{}

	res, err := mockStorage.ReadRecommendationsForClusters([]string{}, testdata.OrgID)

	helpers.FailOnError(t, err)
	assert.Equal(t, expect, res)
}

// TestDBStorageReadRecommendationsGetSelectedClusters loads several recommendations for the same org
// but "simulates" a situation where we only get a subset of them from the AMS API
func TestDBStorageReadRecommendationsGetSelectedClusters(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	clusterList := make([]string, 3)
	for i := range clusterList {
		randomClusterID := testdata.GetRandomClusterID()

		clusterList[i] = string(randomClusterID)

		err := mockStorage.WriteRecommendationsForCluster(
			testdata.OrgID, randomClusterID, testdata.Report3Rules, RecommendationCreatedAtTimestamp,
		)
		helpers.FailOnError(t, err)
	}

	// we only retrieve one cluster
	res, err := mockStorage.ReadRecommendationsForClusters([]string{clusterList[0]}, testdata.OrgID)
	helpers.FailOnError(t, err)

	expect := []types.ClusterName{types.ClusterName(clusterList[0])}
	expectResp := ctypes.RecommendationImpactedClusters{
		testdata.Rule1CompositeID: expect,
		testdata.Rule2CompositeID: expect,
		testdata.Rule3CompositeID: expect,
	}

	assert.Equal(t, expectResp, res)
}

// TestDBStorageReadRecommendationsForNonexistingClusters simulates getting a list of clusters where
// we have none in the DB
func TestDBStorageReadRecommendationsForNonexistingClusters(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	err := mockStorage.WriteRecommendationsForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, RecommendationCreatedAtTimestamp,
	)
	helpers.FailOnError(t, err)

	clusterList := make([]string, 3)
	for i := range clusterList {
		clusterList[i] = string(testdata.GetRandomClusterID())
	}
	res, err := mockStorage.ReadRecommendationsForClusters(clusterList, testdata.OrgID)
	helpers.FailOnError(t, err)

	assert.Equal(t, ctypes.RecommendationImpactedClusters{}, res)
}

// TestDBStorageReadClusterListRecommendationsNoRecommendations checks that when no recommendations
// are stored, it is an OK state
func TestDBStorageReadClusterListRecommendationsNoRecommendations(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report0Rules, []ctypes.ReportItem{},
		testdata.LastCheckedAt, testdata.LastCheckedAt, testdata.LastCheckedAt,
		testdata.RequestID1,
	)
	helpers.FailOnError(t, err)

	err = mockStorage.WriteRecommendationsForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.ClusterReportEmpty, RecommendationCreatedAtTimestamp,
	)
	helpers.FailOnError(t, err)

	res, err := mockStorage.ReadClusterListRecommendations([]string{string(testdata.ClusterName)}, testdata.OrgID)
	helpers.FailOnError(t, err)

	assert.True(t, res[testdata.ClusterName].CreatedAt.Equal(testdata.LastCheckedAt))
}

// TestDBStorageReadClusterListRecommendationsDifferentCluster checks that when no recommendations
// are stored, it is an OK state
func TestDBStorageReadClusterListRecommendationsDifferentCluster(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	err := mockStorage.WriteRecommendationsForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, RecommendationCreatedAtTimestamp,
	)
	helpers.FailOnError(t, err)

	clusterList := make([]string, 3)
	for i := range clusterList {
		clusterList[i] = string(testdata.GetRandomClusterID())
	}
	expect := make(ctypes.ClusterRecommendationMap)

	res, err := mockStorage.ReadClusterListRecommendations(clusterList, testdata.OrgID)
	helpers.FailOnError(t, err)

	assert.Equal(t, expect, res)
}

// TestDBStorageReadClusterListRecommendationsGet1Cluster loads several recommendations for the same org
// but "simulates" a situation where we only get one of them from the AMS API
func TestDBStorageReadClusterListRecommendationsGet1Cluster(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	clusterList := make([]string, 3)
	for i := range clusterList {
		randomClusterID := testdata.GetRandomClusterID()

		clusterList[i] = string(randomClusterID)

		err := mockStorage.WriteReportForCluster(
			testdata.OrgID, randomClusterID, testdata.Report3Rules, testdata.Report3RulesParsed,
			testdata.LastCheckedAt, testdata.LastCheckedAt, testdata.LastCheckedAt,
			testdata.RequestID1,
		)
		helpers.FailOnError(t, err)

		err = mockStorage.WriteRecommendationsForCluster(
			testdata.OrgID, randomClusterID, testdata.Report3Rules, RecommendationCreatedAtTimestamp,
		)
		helpers.FailOnError(t, err)
	}

	// we only retrieve one cluster
	res, err := mockStorage.ReadClusterListRecommendations([]string{clusterList[0]}, testdata.OrgID)
	helpers.FailOnError(t, err)

	expectList := []ctypes.RuleID{
		testdata.Rule1CompositeID,
		testdata.Rule2CompositeID,
		testdata.Rule3CompositeID,
	}

	expectedClusterID := types.ClusterName(clusterList[0])
	assert.Contains(t, res, expectedClusterID)
	assert.ElementsMatch(t, expectList, res[expectedClusterID].Recommendations)
	// trivial timestamp check
	assert.True(t, res[expectedClusterID].CreatedAt.Equal(testdata.LastCheckedAt))
}

// TestDBStorageReadClusterListRecommendationsGet1Cluster loads several recommendations for the same org
// but "simulates" a situation where we only get one of them from the AMS API
func TestDBStorageReadClusterListRecommendationsGetMoreClusters(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	clusterList := make([]string, 3)
	for i := range clusterList {
		randomClusterID := testdata.GetRandomClusterID()

		clusterList[i] = string(randomClusterID)

		err := mockStorage.WriteReportForCluster(
			testdata.OrgID, randomClusterID, testdata.Report3Rules, testdata.Report3RulesParsed,
			testdata.LastCheckedAt, testdata.LastCheckedAt, testdata.LastCheckedAt,
			testdata.RequestID1,
		)
		helpers.FailOnError(t, err)

		err = mockStorage.WriteRecommendationsForCluster(
			testdata.OrgID, randomClusterID, testdata.Report3Rules, RecommendationCreatedAtTimestamp,
		)
		helpers.FailOnError(t, err)
	}

	// we only retrieve one cluster
	res, err := mockStorage.ReadClusterListRecommendations([]string{clusterList[0], clusterList[1]}, testdata.OrgID)
	helpers.FailOnError(t, err)

	expectRuleList := []ctypes.RuleID{
		testdata.Rule1CompositeID,
		testdata.Rule2CompositeID,
		testdata.Rule3CompositeID,
	}

	expectedCluster1ID := types.ClusterName(clusterList[0])
	expectedCluster2ID := types.ClusterName(clusterList[1])
	assert.Contains(t, res, expectedCluster1ID)
	assert.Contains(t, res, expectedCluster2ID)
	assert.ElementsMatch(t, expectRuleList, res[expectedCluster1ID].Recommendations)
	assert.ElementsMatch(t, expectRuleList, res[expectedCluster2ID].Recommendations)
	assert.True(t, res[expectedCluster1ID].CreatedAt.Equal(testdata.LastCheckedAt))
	assert.True(t, res[expectedCluster2ID].CreatedAt.Equal(testdata.LastCheckedAt))
}

// TestDBStorageWriteReportForClusterWithZeroGatheredTime tries to write a report in the DB
// using the "zero" time (if the report doesn't include a gathering time, its value will be
// this one)
func TestDBStorageWriteReportForClusterWithZeroGatheredTime(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	zeroTime := time.Time{}

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.ClusterReportEmpty,
		testdata.ReportEmptyRulesParsed,
		time.Now(),
		zeroTime,
		time.Now(),
		testdata.RequestID1,
	)
	helpers.FailOnError(t, err)
}

func TestDoesClusterExist(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetPostgresStorage(t, true)
	defer closer()

	exist, err := mockStorage.DoesClusterExist(testdata.GetRandomClusterID())
	helpers.FailOnError(t, err)
	assert.False(t, exist, "cluster should not exist")

	err = mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.ClusterReportEmpty,
		testdata.ReportEmptyRulesParsed,
		time.Now(),
		time.Time{},
		time.Now(),
		testdata.RequestID1,
	)
	helpers.FailOnError(t, err)
	exist, err = mockStorage.DoesClusterExist(testdata.ClusterName)
	helpers.FailOnError(t, err)
	assert.True(t, exist, "cluster should exist")
}

func TestReadSingleRuleTemplateData(t *testing.T) {
	mockStorage, expects := ira_helpers.MustGetMockStorageWithExpectsForDriver(t, types.DBDriverPostgres)
	defer ira_helpers.MustCloseMockStorageWithExpects(t, mockStorage, expects)

	expects.ExpectQuery(`SELECT template_data FROM rule_hit`).
		WillReturnRows(expects.NewRows([]string{"template_data"}).AddRow("{json}")).
		RowsWillBeClosed()

	value, err := mockStorage.ReadSingleRuleTemplateData(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.Rule1ID,
		testdata.RuleErrorKey1.ErrorKey,
	)
	assert.NoError(t, err)
	assert.NotNil(t, value)
}
