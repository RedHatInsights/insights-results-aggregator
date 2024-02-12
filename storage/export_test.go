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

package storage

import (
	"database/sql"
	"time"

	"github.com/RedHatInsights/insights-results-aggregator/types"
)

// Export for testing
//
// This source file contains name aliases of all package-private functions
// that need to be called from unit tests. Aliases should start with uppercase
// letter because unit tests belong to different package.
//
// Please look into the following blogpost:
// https://medium.com/@robiplus/golang-trick-export-for-test-aa16cbd7b8cd
// to see why this trick is needed.
type SQLHooks = sqlHooks

const (
	LogFormatterString        = logFormatterString
	SQLHooksKeyQueryBeginTime = sqlHooksKeyQueryBeginTime
)

var (
	ConstructInClausule     = constructInClausule
	ArgsWithClusterNames    = argsWithClusterNames
	ValuesForRuleHitsInsert = valuesForRuleHitsInsert
	NewRedisStorage         = newRedisStorage
	GetRuleHitsCSV          = getRuleHitsCSV
)

func GetConnection(storage *OCPRecommendationsDBStorage) *sql.DB {
	return storage.connection
}

func GetConnectionDVO(storage *DVORecommendationsDBStorage) *sql.DB {
	return storage.connection
}

func GetClustersLastChecked(storage *OCPRecommendationsDBStorage) map[types.ClusterName]time.Time {
	return storage.clustersLastChecked
}

func SetClustersLastChecked(storage *OCPRecommendationsDBStorage, cluster types.ClusterName, lastChecked time.Time) {
	storage.clustersLastChecked[cluster] = lastChecked
}

func InsertRecommendations(
	storage *OCPRecommendationsDBStorage, orgID types.OrgID,
	clusterName types.ClusterName, report types.ReportRules,
	createdAt types.Timestamp,
	impactedSince map[string]types.Timestamp,
) (inserted int, err error) {
	tx, err := storage.connection.Begin()
	if err != nil {
		return 0, err
	}

	inserted, err = storage.insertRecommendations(
		tx, orgID, clusterName,
		report, createdAt, impactedSince)

	if err != nil {
		_ = tx.Rollback()
		return 0, err
	}
	_ = tx.Commit()
	return inserted, nil
}
