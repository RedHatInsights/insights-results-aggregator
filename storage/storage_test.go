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
	"testing"

	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/stretchr/testify/assert"
)

func getMockStorage() (storage.Storage, error) {
	mockStorage, err := storage.New(storage.Configuration{
		Driver:     "sqlite3",
		DataSource: ":memory:",
	})
	if err != nil {
		return nil, err
	}
	err = mockStorage.Init()
	if err != nil {
		return nil, err
	}

	return mockStorage, nil
}

func TestMockDBStorageReadReport(t *testing.T) {
	mockStorage, err := getMockStorage()
	if err != nil {
		t.Fatal(err)
	}
	defer mockStorage.Close()

	const testOrgID = storage.OrgID(1)
	const testClusterName = storage.ClusterName("84f7eedc-0dd8-49cd-9d4d-f6646df3a5bc")
	const testClusterReport = storage.ClusterReport("{}")

	err = mockStorage.WriteReportForCluster(testOrgID, testClusterName, testClusterReport)
	if err != nil {
		t.Fatal(err)
	}

	result, err := mockStorage.ReadReportForCluster(testOrgID, testClusterName)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, testClusterReport, result)

}

func TestMockDBStorageListOfOrgs(t *testing.T) {
	mockStorage, err := getMockStorage()
	if err != nil {
		t.Fatal(err)
	}
	defer mockStorage.Close()

	err = mockStorage.WriteReportForCluster(1, "1deb586c-fb85-4db4-ae5b-139cdbdf77ae", "{}")
	if err != nil {
		t.Fatal(err)
	}

	err = mockStorage.WriteReportForCluster(3, "a1bf5b15-5229-4042-9825-c69dc36b57f5", "{}")
	if err != nil {
		t.Fatal(err)
	}

	result, err := mockStorage.ListOfOrgs()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []storage.OrgID{1, 3}, result)
}

func TestMockDBStorageListOfClustersForOrg(t *testing.T) {
	mockStorage, err := getMockStorage()
	if err != nil {
		t.Fatal(err)
	}
	defer mockStorage.Close()

	err = mockStorage.WriteReportForCluster(1, "eabb4fbf-edfa-45d0-9352-fb05332fdb82", "{}")
	if err != nil {
		t.Fatal(err)
	}
	err = mockStorage.WriteReportForCluster(1, "edf5f242-0c12-4307-8c9f-29dcd289d045", "{}")
	if err != nil {
		t.Fatal(err)
	}
	// also pushing cluster for different org
	err = mockStorage.WriteReportForCluster(5, "4016d01b-62a1-4b49-a36e-c1c5a3d02750", "{}")
	if err != nil {
		t.Fatal(err)
	}

	result, err := mockStorage.ListOfClustersForOrg(1)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []storage.ClusterName{
		"eabb4fbf-edfa-45d0-9352-fb05332fdb82",
		"edf5f242-0c12-4307-8c9f-29dcd289d045",
	}, result)

	result, err = mockStorage.ListOfClustersForOrg(5)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []storage.ClusterName{"4016d01b-62a1-4b49-a36e-c1c5a3d02750"}, result)
}
