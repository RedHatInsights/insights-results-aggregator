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
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/RedHatInsights/insights-results-aggregator/tests/testdata"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const (
	testOrgID              = types.OrgID(1)
	testClusterName        = types.ClusterName("84f7eedc-0dd8-49cd-9d4d-f6646df3a5bc")
	testClusterEmptyReport = types.ClusterReport("{}")
	testRuleID             = types.RuleID("ccx_rules_ocp.external.rules.nodes_kubelet_version_check")
	testUserID             = types.UserID("1")
)

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
	err := storage.WriteReportForCluster(orgID, clusterName, clusterReport, time.Now())
	helpers.FailOnError(t, err)
}

func expectErrorEmptyTable(t *testing.T, err error) {
	if err == nil {
		t.Fatal("Error is expected to be reported because table does not exist")
	}
}

func expectErrorClosedStorage(t *testing.T, err error) {
	if err == nil {
		t.Fatal("Error is expected to be reported because storage has been closed")
	}
}

// TestNewStorage checks whether constructor for new storage returns error for improper storage configuration
func TestNewStorage(t *testing.T) {
	_, err := storage.New(storage.Configuration{
		Driver:           "",
		SQLiteDataSource: "",
	})

	if err == nil {
		t.Fatal("Error needs to be reported for improper storage")
	}
}

// TestNewStorage checks whether constructor for new storage returns error for improper storage configuration
func TestNewStorageError(t *testing.T) {
	_, err := storage.New(storage.Configuration{
		Driver: "non existing driver",
	})

	if err == nil {
		t.Fatal("Error expected")
	}
}

// TestNewStorageWithLogging tests creatign new storage with logs
func TestNewStorageWithLoggingError(t *testing.T) {
	s, _ := storage.New(storage.Configuration{
		Driver:        "postgres",
		PGPort:        1234,
		LogSQLQueries: true,
	})

	if err := s.Init(); err == nil {
		t.Fatal("Error needs to be reported for improper storage")
	}

	_, err := storage.New(storage.Configuration{
		Driver:        "non existing driver",
		LogSQLQueries: true,
	})
	if err == nil {
		t.Fatal(fmt.Errorf("error expected"))
	}
}

// TestMockDBStorageReadReportForClusterEmptyTable check the behaviour of method ReadReportForCluster
func TestMockDBStorageReadReportForClusterEmptyTable(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	_, _, err := mockStorage.ReadReportForCluster(testOrgID, testClusterName)
	if _, ok := err.(*storage.ItemNotFoundError); err == nil || !ok {
		t.Fatalf("expected ItemNotFoundError, got %T, %+v", err, err)
	}

	assert.Equal(
		t,
		fmt.Sprintf(
			"Item with ID %+v/%+v was not found in the storage",
			testOrgID, testClusterName,
		),
		err.Error(),
	)
}

// TestMockDBStorageReadReportForClusterClosedStorage check the behaviour of method ReadReportForCluster
func TestMockDBStorageReadReportForClusterClosedStorage(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	// we need to close storage right now
	helpers.MustCloseStorage(t, mockStorage)

	_, _, err := mockStorage.ReadReportForCluster(testOrgID, testClusterName)
	expectErrorClosedStorage(t, err)
}

// TestMockDBStorageReadReportForCluster check the behaviour of method ReadReportForCluster
func TestMockDBStorageReadReportForCluster(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	writeReportForCluster(t, mockStorage, testOrgID, testClusterName, `{"report":{}}`)
	checkReportForCluster(t, mockStorage, testOrgID, testClusterName, `{"report":{}}`)
}

// TestMockDBStorageReadReportNoTable check the behaviour of method ReadReportForCluster
// when the table with results does not exist
func TestMockDBStorageReadReportNoTable(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, false)
	defer helpers.MustCloseStorage(t, mockStorage)

	_, _, err := mockStorage.ReadReportForCluster(testOrgID, testClusterName)
	expectErrorEmptyTable(t, err)
}

// TestMockDBStorageWriteReportForClusterClosedStorage check the behaviour of method WriteReportForCluster
func TestMockDBStorageWriteReportForClusterClosedStorage(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	// we need to close storage right now
	helpers.MustCloseStorage(t, mockStorage)

	err := mockStorage.WriteReportForCluster(
		testOrgID,
		testClusterName,
		testClusterEmptyReport,
		time.Now(),
	)
	expectErrorClosedStorage(t, err)
}

// TestMockDBStorageListOfOrgs check the behaviour of method ListOfOrgs
func TestMockDBStorageListOfOrgs(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	writeReportForCluster(t, mockStorage, 1, "1deb586c-fb85-4db4-ae5b-139cdbdf77ae", testClusterEmptyReport)
	writeReportForCluster(t, mockStorage, 3, "a1bf5b15-5229-4042-9825-c69dc36b57f5", testClusterEmptyReport)

	result, err := mockStorage.ListOfOrgs()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []types.OrgID{1, 3}, result)
}

func TestMockDBStorageListOfOrgsNoTable(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, false)
	defer helpers.MustCloseStorage(t, mockStorage)

	_, err := mockStorage.ListOfOrgs()
	expectErrorEmptyTable(t, err)
}

// TestMockDBStorageListOfOrgsClosedStorage check the behaviour of method ListOfOrgs
func TestMockDBStorageListOfOrgsClosedStorage(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	// we need to close storage right now
	helpers.MustCloseStorage(t, mockStorage)

	_, err := mockStorage.ListOfOrgs()
	expectErrorClosedStorage(t, err)
}

// TestMockDBStorageListOfClustersFor check the behaviour of method ListOfClustersForOrg
func TestMockDBStorageListOfClustersForOrg(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	writeReportForCluster(t, mockStorage, 1, "eabb4fbf-edfa-45d0-9352-fb05332fdb82", testClusterEmptyReport)
	writeReportForCluster(t, mockStorage, 1, "edf5f242-0c12-4307-8c9f-29dcd289d045", testClusterEmptyReport)

	// also pushing cluster for different org
	writeReportForCluster(t, mockStorage, 5, "4016d01b-62a1-4b49-a36e-c1c5a3d02750", testClusterEmptyReport)

	result, err := mockStorage.ListOfClustersForOrg(1)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []types.ClusterName{
		"eabb4fbf-edfa-45d0-9352-fb05332fdb82",
		"edf5f242-0c12-4307-8c9f-29dcd289d045",
	}, result)

	result, err = mockStorage.ListOfClustersForOrg(5)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []types.ClusterName{"4016d01b-62a1-4b49-a36e-c1c5a3d02750"}, result)
}

func TestMockDBStorageListOfClustersNoTable(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, false)
	defer helpers.MustCloseStorage(t, mockStorage)

	_, err := mockStorage.ListOfClustersForOrg(5)
	expectErrorEmptyTable(t, err)
}

// TestMockDBStorageListOfClustersClosedStorage check the behaviour of method ListOfOrgs
func TestMockDBStorageListOfClustersClosedStorage(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	// we need to close storage right now
	helpers.MustCloseStorage(t, mockStorage)

	_, err := mockStorage.ListOfClustersForOrg(5)
	expectErrorClosedStorage(t, err)
}

// TestMockDBReportsCount check the behaviour of method ReportsCount
func TestMockDBReportsCount(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	cnt, err := mockStorage.ReportsCount()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, cnt, 0)

	writeReportForCluster(t, mockStorage, 5, "4016d01b-62a1-4b49-a36e-c1c5a3d02750", testClusterEmptyReport)

	cnt, err = mockStorage.ReportsCount()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, cnt, 1)
}

func TestMockDBReportsCountNoTable(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, false)
	defer helpers.MustCloseStorage(t, mockStorage)

	_, err := mockStorage.ReportsCount()
	expectErrorEmptyTable(t, err)
}

func TestMockDBReportsCountClosedStorage(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, false)
	// we need to close storage right now
	helpers.MustCloseStorage(t, mockStorage)

	_, err := mockStorage.ReportsCount()
	expectErrorClosedStorage(t, err)
}

func TestDBStorageNewPostgresqlError(t *testing.T) {
	s, _ := storage.New(storage.Configuration{
		Driver: "postgres",
		PGHost: "non-existing-host",
		PGPort: 12345,
	})

	err := s.Init()
	if err == nil {
		t.Fatal(fmt.Errorf("error expected, got %v", err))
	}
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
	if err != nil {
		t.Fatal(err)
	}

	_, err = statement.Exec(
		orgID,
		clusterName,
		clusterReport,
		time.Now(),
		time.Now(),
	)
	if err != nil {
		t.Fatal(err)
	}

	err = statement.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func TestDBStorageListOfOrgsLogError(t *testing.T) {
	buf := new(bytes.Buffer)
	log.Logger = zerolog.New(buf)

	s := helpers.MustGetMockStorage(t, true)

	connection := storage.GetConnection(s.(*storage.DBStorage))
	// write illegal negative org_id
	mustWriteReport(t, connection, -1, testClusterName, testClusterEmptyReport)

	_, err := s.ListOfOrgs()
	if err != nil {
		t.Fatal(err)
	}

	assert.Contains(t, buf.String(), "sql: Scan error")
}

func TestGetDataSourceForDriverFromConfigDriverIsNotSupportedError(t *testing.T) {
	_, err := storage.GetDataSourceForDriverFromConfig(
		-1,
		storage.Configuration{},
	)
	if err == nil {
		t.Fatalf("Expected error, got %v", err)
	}

	assert.Equal(t, "driver -1 is not supported", err.Error())
}

func TestMockDBStorageCloseError(t *testing.T) {
	const errString = "unable to close the database"
	mockStorage, expects := helpers.MustGetMockStorageWithExpects(t)
	expects.ExpectClose().WillReturnError(fmt.Errorf(errString))
	err := mockStorage.Close()
	assert.EqualError(t, err, errString)
}

func TestMockDBStorageListOfClustersForOrgScanError(t *testing.T) {
	// just for the coverage, because this error can't happen ever because we use
	// not null in table creation
	buf := new(bytes.Buffer)
	log.Logger = zerolog.New(buf)

	mockStorage, expects := helpers.MustGetMockStorageWithExpects(t)

	expects.ExpectQuery("SELECT cluster FROM report").WillReturnRows(
		sqlmock.NewRows([]string{"cluster"}).AddRow(nil),
	)

	_, _ = mockStorage.ListOfClustersForOrg(testdata.OrgID)

	assert.Contains(t, buf.String(), "converting NULL to string is unsupported")
}
