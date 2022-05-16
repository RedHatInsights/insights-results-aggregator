// Copyright 2022 Red Hat, Inc
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

package storage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-operator-utils/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	ira_helpers "github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
	"github.com/RedHatInsights/insights-results-aggregator/types"
	ctypes "github.com/RedHatInsights/insights-results-types"
)

func TestWriteReportInfoForCluster(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	expectations := []struct {
		input   []types.InfoItem
		version types.Version
		err     error
	}{
		{
			input:   nil,
			version: "",
			err:     nil,
		},
		{
			input:   []types.InfoItem{},
			version: "",
			err:     nil,
		},
		{
			input: []types.InfoItem{
				{
					InfoID: "An info ID",
					Details: map[string]string{
						"version": "1.0",
					},
				},
			},
			version: "",
			err:     nil,
		},
		{
			input: []types.InfoItem{
				{
					InfoID: "version_info|CLUSTER_VERSION_INFO",
					Details: map[string]string{
						"version": "1.0",
					},
				},
			},
			version: "1.0",
			err:     nil,
		},
	}

	for _, test := range expectations {
		err := mockStorage.WriteReportInfoForCluster(
			testdata.OrgID,
			testdata.ClusterName,
			test.input,
			testdata.LastCheckedAt,
		)
		helpers.FailOnError(t, err)

		version, err := mockStorage.ReadReportInfoForCluster(
			testdata.OrgID, testdata.ClusterName,
		)
		assert.Equal(t, test.version, version)
		assert.Equal(t, test.err, err)
	}
}

// TestDBStorageReadClusterListRecommendationsNoRecommendations checks that when no recommendations
// are stored, it is an OK state
func TestDBStorageReadClusterListRecommendationsNoRecommendationsWithVersion(t *testing.T) {
	mockStorage, closer := ira_helpers.MustGetMockStorage(t, true)
	defer closer()

	err := mockStorage.WriteReportForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.Report0Rules, []ctypes.ReportItem{},
		testdata.LastCheckedAt, testdata.LastCheckedAt, testdata.LastCheckedAt, 0,
	)
	helpers.FailOnError(t, err)

	err = mockStorage.WriteRecommendationsForCluster(
		testdata.OrgID, testdata.ClusterName, testdata.ClusterReportEmpty, RecommendationCreatedAtTimestamp,
	)
	helpers.FailOnError(t, err)

	err = mockStorage.WriteReportInfoForCluster(
		testdata.OrgID, testdata.ClusterName, []types.InfoItem{}, testdata.LastCheckedAt,
	)

	res, err := mockStorage.ReadClusterListRecommendations([]string{string(testdata.ClusterName)}, testdata.OrgID)
	helpers.FailOnError(t, err)

	assert.True(t, res[testdata.ClusterName].CreatedAt.Equal(testdata.LastCheckedAt))
}
