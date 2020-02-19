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
	"github.com/stretchr/testify/assert"
	"testing"
	"time"

	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

const (
	testOrgID         = types.OrgID(1)
	testClusterName   = types.ClusterName("84f7eedc-0dd8-49cd-9d4d-f6646df3a5bc")
	testClusterReport = types.ClusterReport("")
)

// Create mocked storage based on in-memory Sqlite instance
func getMockStorage(init bool) (storage.Storage, error) {
	mockStorage, err := storage.New(storage.Configuration{
		Driver:     "sqlite3",
		DataSource: ":memory:",
	})
	if err != nil {
		return nil, err
	}

	// initialize the database by all required tables
	if init {
		err = mockStorage.Init()
		if err != nil {
			return nil, err
		}
	}

	return mockStorage, nil
}

func checkReportForCluster(t *testing.T, storage storage.Storage, orgID types.OrgID, clusterName types.ClusterName, expected types.ClusterReport) {
	// try to read report for cluster
	result, err := storage.ReadReportForCluster(orgID, clusterName)
	if err != nil {
		t.Fatal(err)
	}

	// and check the read report with expected one
	assert.Equal(t, expected, result)
}

func writeReportForCluster(t *testing.T, storage storage.Storage, orgID types.OrgID, clusterName types.ClusterName, report types.ClusterReport) {
	err := storage.WriteReportForCluster(orgID, clusterName, report, time.Now())
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

// TestNewStorage checks whether constructor for new storage returns error for improper storage configuration
func TestNewStorage(t *testing.T) {
	_, err := storage.New(storage.Configuration{
		Driver:     "",
		DataSource: "",
	})

	if err == nil {
		t.Fatal("Error needs to be reported for improper storage")
	}
}

// TestMockDBStorageReadReportForClusterEmptyTable check the behaviour of method ReadReportForCluster
func TestMockDBStorageReadReportForClusterEmptyTable(t *testing.T) {
	mockStorage, err := getMockStorage(true)
	if err != nil {
		t.Fatal(err)
	}
	defer mockStorage.Close()

	_, err = mockStorage.ReadReportForCluster(testOrgID, testClusterName)
	if _, ok := err.(*storage.ItemNotFoundError); err == nil || !ok {
		t.Fatalf("expected ItemNotFoundError, got %T, %+v", err, err)
	}
}

// TestMockDBStorageReadReportForClusterClosedStorage check the behaviour of method ReadReportForCluster
func TestMockDBStorageReadReportForClusterClosedStorage(t *testing.T) {
	mockStorage, err := getMockStorage(true)
	if err != nil {
		t.Fatal(err)
	}
	mockStorage.Close()

	_, err = mockStorage.ReadReportForCluster(testOrgID, testClusterName)
	expectErrorClosedStorage(t, err)
}

// TestMockDBStorageReadReportForCluster check the behaviour of method ReadReportForCluster
func TestMockDBStorageReadReportForCluster(t *testing.T) {
	mockStorage, err := getMockStorage(true)
	if err != nil {
		t.Fatal(err)
	}
	defer mockStorage.Close()

	writeReportForCluster(t, mockStorage, testOrgID, testClusterName, testClusterReport)
	checkReportForCluster(t, mockStorage, testOrgID, testClusterName, testClusterReport)
}

// TestMockDBStorageReadReportNoTable check the behaviour of method ReadReportForCluster when the table with results does not exist
func TestMockDBStorageReadReportNoTable(t *testing.T) {
	mockStorage, err := getMockStorage(false)
	if err != nil {
		t.Fatal(err)
	}
	defer mockStorage.Close()

	_, err = mockStorage.ReadReportForCluster(testOrgID, testClusterName)
	expectErrorEmptyTable(t, err)
}

// TestMockDBStorageWriteReportForClusterClosedStorage check the behaviour of method WriteReportForCluster
func TestMockDBStorageWriteReportForClusterClosedStorage(t *testing.T) {
	mockStorage, err := getMockStorage(true)
	if err != nil {
		t.Fatal(err)
	}
	mockStorage.Close()

	err = mockStorage.WriteReportForCluster(testOrgID, testClusterName, testClusterReport, time.Now())
	expectErrorClosedStorage(t, err)
}

// TestMockDBStorageListOfOrgs check the behaviour of method ListOfOrgs
func TestMockDBStorageListOfOrgs(t *testing.T) {
	mockStorage, err := getMockStorage(true)
	if err != nil {
		t.Fatal(err)
	}
	defer mockStorage.Close()

	writeReportForCluster(t, mockStorage, 1, "1deb586c-fb85-4db4-ae5b-139cdbdf77ae", "{}")
	writeReportForCluster(t, mockStorage, 3, "a1bf5b15-5229-4042-9825-c69dc36b57f5", "{}")

	result, err := mockStorage.ListOfOrgs()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []types.OrgID{1, 3}, result)
}

func TestMockDBStorageListOfOrgsNoTable(t *testing.T) {
	mockStorage, err := getMockStorage(false)
	if err != nil {
		t.Fatal(err)
	}
	defer mockStorage.Close()

	_, err = mockStorage.ListOfOrgs()
	expectErrorEmptyTable(t, err)
}

// TestMockDBStorageListOfOrgsClosedStorage check the behaviour of method ListOfOrgs
func TestMockDBStorageListOfOrgsClosedStorage(t *testing.T) {
	mockStorage, err := getMockStorage(true)
	if err != nil {
		t.Fatal(err)
	}
	mockStorage.Close()

	_, err = mockStorage.ListOfOrgs()
	expectErrorClosedStorage(t, err)
}

// TestMockDBStorageListOfClustersFor check the behaviour of method ListOfClustersForOrg
func TestMockDBStorageListOfClustersForOrg(t *testing.T) {
	mockStorage, err := getMockStorage(true)
	if err != nil {
		t.Fatal(err)
	}
	defer mockStorage.Close()

	writeReportForCluster(t, mockStorage, 1, "eabb4fbf-edfa-45d0-9352-fb05332fdb82", "{}")
	writeReportForCluster(t, mockStorage, 1, "edf5f242-0c12-4307-8c9f-29dcd289d045", "{}")

	// also pushing cluster for different org
	writeReportForCluster(t, mockStorage, 5, "4016d01b-62a1-4b49-a36e-c1c5a3d02750", "{}")

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
	mockStorage, err := getMockStorage(false)
	if err != nil {
		t.Fatal(err)
	}
	defer mockStorage.Close()

	_, err = mockStorage.ListOfClustersForOrg(5)
	expectErrorEmptyTable(t, err)
}

// TestMockDBStorageListOfClustersClosedStorage check the behaviour of method ListOfOrgs
func TestMockDBStorageListOfClustersClosedStorage(t *testing.T) {
	mockStorage, err := getMockStorage(true)
	if err != nil {
		t.Fatal(err)
	}
	mockStorage.Close()

	_, err = mockStorage.ListOfClustersForOrg(5)
	expectErrorClosedStorage(t, err)
}

// TestMockDBReportsCount check the behaviour of method ReportsCount
func TestMockDBReportsCount(t *testing.T) {
	mockStorage, err := getMockStorage(true)
	if err != nil {
		t.Fatal(err)
	}
	defer mockStorage.Close()

	cnt, err := mockStorage.ReportsCount()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, cnt, 0)

	writeReportForCluster(t, mockStorage, 5, "4016d01b-62a1-4b49-a36e-c1c5a3d02750", "{}")

	cnt, err = mockStorage.ReportsCount()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, cnt, 1)
}

func TestMockDBReportsCountNoTable(t *testing.T) {
	mockStorage, err := getMockStorage(false)
	if err != nil {
		t.Fatal(err)
	}
	defer mockStorage.Close()

	_, err = mockStorage.ReportsCount()
	expectErrorEmptyTable(t, err)
}

func TestMockDBReportsCountClosedStorage(t *testing.T) {
	mockStorage, err := getMockStorage(false)
	if err != nil {
		t.Fatal(err)
	}
	mockStorage.Close()

	_, err = mockStorage.ReportsCount()
	expectErrorClosedStorage(t, err)
}
