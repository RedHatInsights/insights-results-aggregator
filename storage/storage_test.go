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
	"strings"
	"testing"
	"time"

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
	if err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatal(err)
	}
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

func closeStorage(t *testing.T, mockStorage storage.Storage) {
	err := mockStorage.Close()
	if err != nil {
		t.Fatal(err)
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
	defer closeStorage(t, mockStorage)

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
	closeStorage(t, mockStorage)

	_, _, err := mockStorage.ReadReportForCluster(testOrgID, testClusterName)
	expectErrorClosedStorage(t, err)
}

// TestMockDBStorageReadReportForCluster check the behaviour of method ReadReportForCluster
func TestMockDBStorageReadReportForCluster(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer closeStorage(t, mockStorage)

	writeReportForCluster(t, mockStorage, testOrgID, testClusterName, `{"report":{}}`)
	checkReportForCluster(t, mockStorage, testOrgID, testClusterName, `{"report":{}}`)
}

// TestMockDBStorageReadReportNoTable check the behaviour of method ReadReportForCluster
// when the table with results does not exist
func TestMockDBStorageReadReportNoTable(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, false)
	defer closeStorage(t, mockStorage)

	_, _, err := mockStorage.ReadReportForCluster(testOrgID, testClusterName)
	expectErrorEmptyTable(t, err)
}

// TestMockDBStorageWriteReportForClusterClosedStorage check the behaviour of method WriteReportForCluster
func TestMockDBStorageWriteReportForClusterClosedStorage(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	// we need to close storage right now
	closeStorage(t, mockStorage)

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
	defer closeStorage(t, mockStorage)

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
	defer closeStorage(t, mockStorage)

	_, err := mockStorage.ListOfOrgs()
	expectErrorEmptyTable(t, err)
}

// TestMockDBStorageListOfOrgsClosedStorage check the behaviour of method ListOfOrgs
func TestMockDBStorageListOfOrgsClosedStorage(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	// we need to close storage right now
	closeStorage(t, mockStorage)

	_, err := mockStorage.ListOfOrgs()
	expectErrorClosedStorage(t, err)
}

// TestMockDBStorageListOfClustersFor check the behaviour of method ListOfClustersForOrg
func TestMockDBStorageListOfClustersForOrg(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer closeStorage(t, mockStorage)

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
	defer closeStorage(t, mockStorage)

	_, err := mockStorage.ListOfClustersForOrg(5)
	expectErrorEmptyTable(t, err)
}

// TestMockDBStorageListOfClustersClosedStorage check the behaviour of method ListOfOrgs
func TestMockDBStorageListOfClustersClosedStorage(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	// we need to close storage right now
	closeStorage(t, mockStorage)

	_, err := mockStorage.ListOfClustersForOrg(5)
	expectErrorClosedStorage(t, err)
}

// TestMockDBReportsCount check the behaviour of method ReportsCount
func TestMockDBReportsCount(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer closeStorage(t, mockStorage)

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
	defer closeStorage(t, mockStorage)

	_, err := mockStorage.ReportsCount()
	expectErrorEmptyTable(t, err)
}

func TestMockDBReportsCountClosedStorage(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, false)
	// we need to close storage right now
	closeStorage(t, mockStorage)

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
	log.Logger = zerolog.New(buf).With().Str("type", "SQL").Logger()

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

func TestDBStorageVoteOnRule(t *testing.T) {
	for _, vote := range []storage.UserVote{
		storage.UserVoteDislike, storage.UserVoteLike, storage.UserVoteNone,
	} {
		mockStorage := helpers.MustGetMockStorage(t, true)

		helpers.FailOnError(t, mockStorage.VoteOnRule(
			testClusterName, testRuleID, testUserID, vote,
		))

		feedback, err := mockStorage.GetUserFeedbackOnRule(testClusterName, testRuleID, testUserID)
		helpers.FailOnError(t, err)

		assert.Equal(t, testClusterName, feedback.ClusterID)
		assert.Equal(t, testRuleID, feedback.RuleID)
		assert.Equal(t, testUserID, feedback.UserID)
		assert.Equal(t, "", feedback.Message)
		assert.Equal(t, vote, feedback.UserVote)

		helpers.FailOnError(t, mockStorage.Close())
	}
}

func TestDBStorageChangeVote(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	helpers.FailOnError(t, mockStorage.VoteOnRule(
		testClusterName, testRuleID, testUserID, storage.UserVoteLike,
	))
	// just to be sure that addedAt != to updatedAt
	time.Sleep(1 * time.Millisecond)
	helpers.FailOnError(t, mockStorage.VoteOnRule(
		testClusterName, testRuleID, testUserID, storage.UserVoteDislike,
	))

	feedback, err := mockStorage.GetUserFeedbackOnRule(testClusterName, testRuleID, testUserID)
	helpers.FailOnError(t, err)

	assert.Equal(t, testClusterName, feedback.ClusterID)
	assert.Equal(t, testRuleID, feedback.RuleID)
	assert.Equal(t, testUserID, feedback.UserID)
	assert.Equal(t, "", feedback.Message)
	assert.Equal(t, storage.UserVoteDislike, feedback.UserVote)
	assert.NotEqual(t, feedback.AddedAt, feedback.UpdatedAt)
}

func TestDBStorageTextFeedback(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	helpers.FailOnError(t, mockStorage.AddOrUpdateFeedbackOnRule(
		testClusterName, testRuleID, testUserID, "test feedback",
	))

	feedback, err := mockStorage.GetUserFeedbackOnRule(testClusterName, testRuleID, testUserID)
	helpers.FailOnError(t, err)

	assert.Equal(t, testClusterName, feedback.ClusterID)
	assert.Equal(t, testRuleID, feedback.RuleID)
	assert.Equal(t, testUserID, feedback.UserID)
	assert.Equal(t, "test feedback", feedback.Message)
	assert.Equal(t, storage.UserVoteNone, feedback.UserVote)
}

func TestDBStorageFeedbackChangeMessage(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	helpers.FailOnError(t, mockStorage.AddOrUpdateFeedbackOnRule(
		testClusterName, testRuleID, testUserID, "message1",
	))
	// just to be sure that addedAt != to updatedAt
	time.Sleep(1 * time.Millisecond)
	helpers.FailOnError(t, mockStorage.AddOrUpdateFeedbackOnRule(
		testClusterName, testRuleID, testUserID, "message2",
	))

	feedback, err := mockStorage.GetUserFeedbackOnRule(testClusterName, testRuleID, testUserID)
	helpers.FailOnError(t, err)

	assert.Equal(t, testClusterName, feedback.ClusterID)
	assert.Equal(t, testRuleID, feedback.RuleID)
	assert.Equal(t, testUserID, feedback.UserID)
	assert.Equal(t, "message2", feedback.Message)
	assert.Equal(t, storage.UserVoteNone, feedback.UserVote)
	assert.NotEqual(t, feedback.AddedAt, feedback.UpdatedAt)
}

func TestDBStorageFeedbackErrorItemNotFound(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	defer helpers.MustCloseStorage(t, mockStorage)

	_, err := mockStorage.GetUserFeedbackOnRule(testClusterName, testRuleID, testUserID)
	if _, ok := err.(*storage.ItemNotFoundError); err == nil || !ok {
		t.Fatalf("expected ItemNotFoundError, got %T, %+v", err, err)
	}
}

func TestDBStorageFeedbackErrorDBError(t *testing.T) {
	mockStorage := helpers.MustGetMockStorage(t, true)
	helpers.MustCloseStorage(t, mockStorage)

	_, err := mockStorage.GetUserFeedbackOnRule(testClusterName, testRuleID, testUserID)
	if err == nil || !strings.Contains(err.Error(), "database is closed") {
		t.Fatalf("expected sql database is closed error, got %T, %+v", err, err)
	}
}
