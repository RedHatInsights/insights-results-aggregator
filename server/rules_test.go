// Copyright 2026 Red Hat, Inc
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

package server_test

import (
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/RedHatInsights/insights-results-aggregator-data/testdata"
	"github.com/stretchr/testify/assert"

	"github.com/RedHatInsights/insights-results-aggregator/server"
	"github.com/RedHatInsights/insights-results-aggregator/storage"
	"github.com/RedHatInsights/insights-results-aggregator/tests/helpers"
)

func TestGetRuleToggleMapForCluster_SkipsMalformedDisabledRule(t *testing.T) {
	mockStorage, expects := helpers.MustGetMockStorageWithExpects(t)
	defer helpers.MustCloseMockStorageWithExpects(t, mockStorage, expects)

	now := time.Now()
	expects.ExpectQuery("SELECT").
		WithArgs(testdata.OrgID, storage.RuleToggleDisable).
		WillReturnRows(sqlmock.NewRows([]string{
			"cluster_id", "rule_id", "error_key", "disabled_at", "updated_at", "disabled",
		}).
			AddRow(testdata.ClusterName, testdata.Rule1ID, "", now, now, storage.RuleToggleDisable).
			AddRow(testdata.ClusterName, testdata.Rule2ID, testdata.ErrorKey2, now, now, storage.RuleToggleDisable).
			AddRow(testdata.GetRandomClusterID(), testdata.Rule1ID, testdata.ErrorKey1, now, now, storage.RuleToggleDisable))

	testServer := server.New(helpers.DefaultServerConfig, mockStorage, nil)

	toggleMap, err := server.GetRuleToggleMapForCluster(testServer, testdata.ClusterName, testdata.OrgID)
	helpers.FailOnError(t, err)

	assert.Len(t, toggleMap, 1)
	assert.True(t, toggleMap[testdata.Rule2CompositeID])
}

func TestGetRuleToggleMapForCluster_ListOfDisabledRulesDBError(t *testing.T) {
	const errStr = "database error"

	mockStorage, expects := helpers.MustGetMockStorageWithExpects(t)
	defer helpers.MustCloseMockStorageWithExpects(t, mockStorage, expects)

	expects.ExpectQuery("SELECT").
		WithArgs(testdata.OrgID, storage.RuleToggleDisable).
		WillReturnError(errors.New(errStr))

	testServer := server.New(helpers.DefaultServerConfig, mockStorage, nil)

	toggleMap, err := server.GetRuleToggleMapForCluster(testServer, testdata.ClusterName, testdata.OrgID)
	assert.EqualError(t, err, errStr)
	assert.Empty(t, toggleMap)
}
