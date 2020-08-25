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

package storage_test

import (
	"bytes"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/Shopify/sarama"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"

	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

func init() {
	zerolog.SetGlobalLevel(zerolog.WarnLevel)
}

func assertNumberOfReports(t *testing.T, mockStorage storage.Storage, expectedNumberOfReports int) {
	numberOfReports, err := mockStorage.ReportsCount()
	helpers.FailOnError(t, err)
	assert.Equal(t, expectedNumberOfReports, numberOfReports)
}

func checkReportForCluster(
	t *testing.T,
	s storage.Storage,
	orgID types.OrgID,
	clusterName types.ClusterName,
	expected types.ClusterReport,
) {
	// try to read report for cluster
	result, _, err := s.ReadReportForCluster(orgID, clusterName)
	helpers.FailOnError(t, err)

	// and check the read report with expected one
	assert.Equal(t, expected, result)
}

func writeReportForCluster(
	t *testing.T,
	storage storage.Storage,
	orgID types.OrgID,
	clusterName types.ClusterName,
	clusterReport types.ClusterReport,
) {
	err := storage.WriteReportForCluster(orgID, clusterName, clusterReport, time.Now(), testdata.KafkaOffset)
	helpers.FailOnError(t, err)
}

// TestNewStorage checks whether constructor for new storage returns error for improper storage configuration
func TestNewStorageError(t *testing.T) {
	_, err := storage.New(storage.Configuration{
		Driver: "non existing driver",
	})
	assert.EqualError(t, err, "driver non existing driver is not supported")
}

// TestNewStorageWithLogging tests creating new storage with logs
func TestNewStorageWithLoggingError(t *testing.T) {
	s, _ := storage.New(storage.Configuration{
		Driver:        "postgres",
		PGPort:        1234,
		PGUsername:    "user",
		LogSQLQueries: true,
	})

	err := s.Init()
	assert.Contains(t, err.Error(), "connect: connection refused")
}

// TestDBStorageReadReportForClusterEmptyTable check the behaviour of method ReadReportForCluster
func TestDBStorageReadReportForClusterEmptyTable(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	_, _, err := mockStorage.ReadReportForCluster(testdata.OrgID, testdata.ClusterName)
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
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	// we need to close storage right now
	closer()

	_, _, err := mockStorage.ReadReportForCluster(testdata.OrgID, testdata.ClusterName)
	assert.EqualError(t, err, "sql: database is closed")
}

// TestDBStorageReadReportForCluster check the behaviour of method ReadReportForCluster
func TestDBStorageReadReportForCluster(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	writeReportForCluster(t, mockStorage, testdata.OrgID, testdata.ClusterName, `{"report":{}}`)
	checkReportForCluster(t, mockStorage, testdata.OrgID, testdata.ClusterName, `{"report":{}}`)
}

// TestDBStorageGetOrgIDByClusterID check the behaviour of method GetOrgIDByClusterID
func TestDBStorageGetOrgIDByClusterID(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	writeReportForCluster(t, mockStorage, testdata.OrgID, testdata.ClusterName, `{"report":{}}`)

	orgID, err := mockStorage.GetOrgIDByClusterID(testdata.ClusterName)
	helpers.FailOnError(t, err)

	assert.Equal(t, orgID, testdata.OrgID)
}

func TestDBStorageGetOrgIDByClusterID_Error(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, false)
	defer closer()

	dbStorage := mockStorage.(*storage.DBStorage)
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
			`converting driver.Value type string ("not-int") to a uint64: invalid syntax`,
	)
}

// TestDBStorageReadReportNoTable check the behaviour of method ReadReportForCluster
// when the table with results does not exist
func TestDBStorageReadReportNoTable(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, false)
	defer closer()

	_, _, err := mockStorage.ReadReportForCluster(testdata.OrgID, testdata.ClusterName)
	assert.EqualError(t, err, "no such table: report")
}

// TestDBStorageWriteReportForClusterClosedStorage check the behaviour of method WriteReportForCluster
func TestDBStorageWriteReportForClusterClosedStorage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	// we need to close storage right now
	closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.ClusterReportEmpty,
		time.Now(),
		testdata.KafkaOffset,
	)
	assert.EqualError(t, err, "sql: database is closed")
}

// TestDBStorageWriteReportForClusterClosedStorage check the behaviour of method WriteReportForCluster
func TestDBStorageWriteReportForClusterUnsupportedDriverError(t *testing.T) {
	fakeStorage := storage.NewFromConnection(nil, -1)
	// no need to close it

	err := fakeStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.ClusterReportEmpty,
		time.Now(),
		testdata.KafkaOffset,
	)
	assert.EqualError(t, err, "writing report with DB -1 is not supported")
}

// TestDBStorageWriteReportForClusterMoreRecentInDB checks that older report
// will not replace a more recent one when writing a report to storage.
func TestDBStorageWriteReportForClusterMoreRecentInDB(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	newerTime := time.Now().UTC()
	olderTime := newerTime.Add(-time.Hour)

	// Insert newer report.
	err := mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.ClusterReportEmpty,
		newerTime,
		testdata.KafkaOffset,
	)
	helpers.FailOnError(t, err)

	// Try to insert older report.
	err = mockStorage.WriteReportForCluster(
		testdata.OrgID,
		testdata.ClusterName,
		testdata.ClusterReportEmpty,
		olderTime,
		testdata.KafkaOffset,
	)
	assert.Equal(t, types.ErrOldReport, err)
}

// TestDBStorageWriteReportForClusterDroppedReportTable checks the error
// returned when trying to SELECT from a dropped/missing report table.
func TestDBStorageWriteReportForClusterDroppedReportTable(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	connection := storage.GetConnection(mockStorage.(*storage.DBStorage))

	query := "DROP TABLE report"
	if os.Getenv("INSIGHTS_RESULTS_AGGREGATOR__TESTS_DB") == "postgres" {
		query += " CASCADE"
	}
	query += ";"

	_, err := connection.Exec(query)
	helpers.FailOnError(t, err)

	err = mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.ClusterReportEmpty, time.Now(), testdata.KafkaOffset,
	)
	assert.EqualError(t, err, "no such table: report")
}

func TestDBStorageWriteReportForClusterExecError(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, false)
	defer closer()

	createReportTableWithBadClusterField(t, mockStorage)

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.LastCheckedAt, testdata.KafkaOffset,
	)
	assert.Error(t, err)
	const sqliteErrMessage = "CHECK constraint failed: report"
	const postgresErrMessage = "pq: invalid input syntax for integer"
	if err.Error() != sqliteErrMessage && !strings.HasPrefix(err.Error(), postgresErrMessage) {
		t.Fatalf("expected on of: \n%v\n%v\ngot:\n%v", sqliteErrMessage, postgresErrMessage, err.Error())
	}
}

func TestDBStorageWriteReportForClusterFakePostgresOK(t *testing.T) {
	mockStorage, expects := ira_helpers.MustGetMockStorageWithExpectsForDriver(t, types.DBDriverPostgres)
	defer ira_helpers.MustCloseMockStorageWithExpects(t, mockStorage, expects)

	expects.ExpectBegin()

	expects.ExpectQuery(`SELECT last_checked_at FROM report`).
		WillReturnRows(expects.NewRows([]string{"last_checked_at"})).
		RowsWillBeClosed()

	expects.ExpectExec("INSERT INTO report").
		WillReturnResult(driver.ResultNoRows)

	expects.ExpectCommit()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.LastCheckedAt, testdata.KafkaOffset,
	)
	helpers.FailOnError(t, err)
}

// TestDBStorageListOfOrgs check the behaviour of method ListOfOrgs
func TestDBStorageListOfOrgs(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	writeReportForCluster(t, mockStorage, 1, "1deb586c-fb85-4db4-ae5b-139cdbdf77ae", testdata.ClusterReportEmpty)
	writeReportForCluster(t, mockStorage, 3, "a1bf5b15-5229-4042-9825-c69dc36b57f5", testdata.ClusterReportEmpty)

	result, err := mockStorage.ListOfOrgs()
	helpers.FailOnError(t, err)

	assert.ElementsMatch(t, []types.OrgID{1, 3}, result)
}

func TestDBStorageListOfOrgsNoTable(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, false)
	defer closer()

	_, err := mockStorage.ListOfOrgs()
	assert.EqualError(t, err, "no such table: report")
}

// TestDBStorageListOfOrgsClosedStorage check the behaviour of method ListOfOrgs
func TestDBStorageListOfOrgsClosedStorage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	// we need to close storage right now
	closer()

	_, err := mockStorage.ListOfOrgs()
	assert.EqualError(t, err, "sql: database is closed")
}

// TestDBStorageListOfClustersFor check the behaviour of method ListOfClustersForOrg
func TestDBStorageListOfClustersForOrg(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	writeReportForCluster(t, mockStorage, 1, "eabb4fbf-edfa-45d0-9352-fb05332fdb82", testdata.ClusterReportEmpty)
	writeReportForCluster(t, mockStorage, 1, "edf5f242-0c12-4307-8c9f-29dcd289d045", testdata.ClusterReportEmpty)

	// also pushing cluster for different org
	writeReportForCluster(t, mockStorage, 5, "4016d01b-62a1-4b49-a36e-c1c5a3d02750", testdata.ClusterReportEmpty)

	result, err := mockStorage.ListOfClustersForOrg(1)
	helpers.FailOnError(t, err)

	assert.ElementsMatch(t, []types.ClusterName{
		"eabb4fbf-edfa-45d0-9352-fb05332fdb82",
		"edf5f242-0c12-4307-8c9f-29dcd289d045",
	}, result)

	result, err = mockStorage.ListOfClustersForOrg(5)
	helpers.FailOnError(t, err)

	assert.Equal(t, []types.ClusterName{"4016d01b-62a1-4b49-a36e-c1c5a3d02750"}, result)
}

func TestDBStorageListOfClustersNoTable(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, false)
	defer closer()

	_, err := mockStorage.ListOfClustersForOrg(5)
	assert.EqualError(t, err, "no such table: report")
}

// TestDBStorageListOfClustersClosedStorage check the behaviour of method ListOfOrgs
func TestDBStorageListOfClustersClosedStorage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	// we need to close storage right now
	closer()

	_, err := mockStorage.ListOfClustersForOrg(5)
	assert.EqualError(t, err, "sql: database is closed")
}

// TestMockDBReportsCount check the behaviour of method ReportsCount
func TestMockDBReportsCount(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	assertNumberOfReports(t, mockStorage, 0)

	writeReportForCluster(t, mockStorage, 5, "4016d01b-62a1-4b49-a36e-c1c5a3d02750", testdata.ClusterReportEmpty)

	assertNumberOfReports(t, mockStorage, 1)
}

func TestMockDBReportsCountNoTable(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, false)
	defer closer()

	_, err := mockStorage.ReportsCount()
	assert.EqualError(t, err, "no such table: report")
}

func TestMockDBReportsCountClosedStorage(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, false)
	// we need to close storage right now
	closer()

	_, err := mockStorage.ReportsCount()
	assert.EqualError(t, err, "sql: database is closed")
}

func TestDBStorageNewPostgresqlError(t *testing.T) {
	s, _ := storage.New(storage.Configuration{
		Driver:     "postgres",
		PGHost:     "non-existing-host",
		PGPort:     12345,
		PGUsername: "user",
	})

	err := s.Init()
	assert.Contains(t, err.Error(), "no such host")
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

	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	connection := storage.GetConnection(mockStorage.(*storage.DBStorage))
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

	_, err := mockStorage.ListOfClustersForOrg(testdata.OrgID)
	helpers.FailOnError(t, err)

	assert.Contains(t, buf.String(), "converting NULL to string is unsupported")
}

func TestDBStorageDeleteReports(t *testing.T) {
	for _, functionName := range []string{
		"DeleteReportsForOrg", "DeleteReportsForCluster",
	} {
		func() {
			mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
			defer closer()
			assertNumberOfReports(t, mockStorage, 0)

			err := mockStorage.WriteReportForCluster(
				testdata.OrgID,
				testdata.ClusterName,
				testdata.Report3Rules,
				testdata.LastCheckedAt,
				testdata.KafkaOffset,
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
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	mustWriteReport3Rules(t, mockStorage)

	report, lastCheckedAt, err := mockStorage.ReadReportForClusterByClusterName(testdata.ClusterName)
	helpers.FailOnError(t, err)

	assert.Equal(t, testdata.Report3Rules, report)
	assert.Equal(t, types.Timestamp(testdata.LastCheckedAt.UTC().Format(time.RFC3339)), lastCheckedAt)
}

func TestDBStorage_CheckIfClusterExists_ClusterDoesNotExist(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	_, _, err := mockStorage.ReadReportForClusterByClusterName(testdata.ClusterName)
	assert.EqualError(
		t,
		err,
		fmt.Sprintf("Item with ID %v was not found in the storage", testdata.ClusterName),
	)
}

func TestDBStorage_CheckIfClusterExists_DBError(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	closer()

	_, _, err := mockStorage.ReadReportForClusterByClusterName(testdata.ClusterName)
	assert.EqualError(t, err, "sql: database is closed")
}

func TestDBStorage_NewSQLite(t *testing.T) {
	_, err := storage.New(storage.Configuration{
		Driver:           "sqlite3",
		SQLiteDataSource: ":memory:",
	})
	helpers.FailOnError(t, err)
}

func TestDBStorageWriteConsumerError(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
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

	conn := storage.GetConnection(mockStorage.(*storage.DBStorage))
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

func TestDBStorage_GetLatestKafkaOffset(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	offset, err := mockStorage.GetLatestKafkaOffset()
	helpers.FailOnError(t, err)

	assert.Equal(t, types.KafkaOffset(0), offset)

	mustWriteReport3Rules(t, mockStorage)

	offset, err = mockStorage.GetLatestKafkaOffset()
	helpers.FailOnError(t, err)

	assert.Equal(t, types.KafkaOffset(1), offset)
}

func TestDBStorage_GetLatestKafkaOffset_ZeroOffset(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	offset, err := mockStorage.GetLatestKafkaOffset()
	helpers.FailOnError(t, err)

	assert.Equal(t, types.KafkaOffset(0), offset)

	err = mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report3Rules, testdata.LastCheckedAt, types.KafkaOffset(0),
	)
	helpers.FailOnError(t, err)

	offset, err = mockStorage.GetLatestKafkaOffset()
	helpers.FailOnError(t, err)

	assert.Equal(t, types.KafkaOffset(0), offset)
}

func TestDBStorage_Init(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	dbStorage := mockStorage.(*storage.DBStorage)

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
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, false)
	defer closer()

	createReportTableWithBadClusterField(t, mockStorage)

	connection := storage.GetConnection(mockStorage.(*storage.DBStorage))

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

func createReportTableWithBadClusterField(t *testing.T, mockStorage storage.Storage) {
	connection := storage.GetConnection(mockStorage.(*storage.DBStorage))

	query := `
		CREATE TABLE report (
			org_id          INTEGER NOT NULL,
			cluster         INTEGER NOT NULL UNIQUE CHECK(typeof(cluster) = 'integer'),
			report          VARCHAR NOT NULL,
			reported_at     TIMESTAMP,
			last_checked_at TIMESTAMP,
			kafka_offset BIGINT NOT NULL DEFAULT 0,
			PRIMARY KEY(org_id, cluster)
		)
	`

	if os.Getenv("INSIGHTS_RESULTS_AGGREGATOR__TESTS_DB") == "postgres" {
		query = `
			CREATE TABLE report (
				org_id          INTEGER NOT NULL,
				cluster         INTEGER NOT NULL UNIQUE,
				report          VARCHAR NOT NULL,
				reported_at     TIMESTAMP,
				last_checked_at TIMESTAMP,
				kafka_offset BIGINT NOT NULL DEFAULT 0,
				PRIMARY KEY(org_id, cluster)
			)
		`
	}

	// create a table with a bad type
	_, err := connection.Exec(query)
	helpers.FailOnError(t, err)
}
