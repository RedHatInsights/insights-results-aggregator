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

package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/RedHatInsights/insights-results-aggregator/types"
	ctypes "github.com/RedHatInsights/insights-results-types"
)

const (
	versionInfoKey = "version_info|CLUSTER_VERSION_INFO"
)

// WriteReportInfoForCluster writes the relevant report info for selected cluster for hiven organization
func (storage DBStorage) WriteReportInfoForCluster(
	orgID types.OrgID,
	clusterName types.ClusterName,
	info []types.InfoItem,
	lastCheckedTime time.Time,
) error {
	// Skip writing the report if it isn't newer than a report
	// that is already in the database for the same cluster.
	if oldLastChecked, exists := storage.clustersLastChecked[clusterName]; exists && !lastCheckedTime.After(oldLastChecked) {
		return types.ErrOldReport
	}

	if storage.dbDriverType != types.DBDriverSQLite3 && storage.dbDriverType != types.DBDriverPostgres {
		return fmt.Errorf("writing report with DB %v is not supported", storage.dbDriverType)
	}

	// Begin a new transaction.
	tx, err := storage.connection.Begin()
	if err != nil {
		return err
	}

	err = storage.updateInfoReport(tx, orgID, clusterName, info)

	finishTransaction(tx, err)
	return err
}

func (storage DBStorage) updateInfoReport(
	tx *sql.Tx,
	orgID types.OrgID,
	clusterName types.ClusterName,
	infoRules []types.InfoItem,
) error {
	// Get the UPSERT query for writing an info report into the database.
	infoUpsertQuery := storage.getReportInfoUpsertQuery()

	for _, info := range infoRules {
		if info.InfoID != versionInfoKey {
			continue
		}

		_, err := tx.Exec(infoUpsertQuery, orgID, clusterName, info.Details["version"])
		if err != nil {
			return err
		}
	}
	return nil
}

// ReadReportInfoForCluster retrieve the Version for a given cluster and org id
func (storage *DBStorage) ReadReportInfoForCluster(
	orgID types.OrgID,
	clusterName types.ClusterName,
) (types.Version, error) {
	var version types.Version

	err := storage.connection.QueryRow(
		`
SELECT 
	COALESCE ( 
		( 
			SELECT version_info 
			FROM report_info 
			WHERE org_id = $1 AND cluster_id = $2 
		), '') 
		AS version_info;
		`,
		orgID, clusterName,
	).Scan(&version)

	err = types.ConvertDBError(err, []interface{}{orgID, clusterName})
	return version, err
}

func (storage DBStorage) fillInMetadata(orgID types.OrgID, clusterMap ctypes.ClusterRecommendationMap) {
	for cluster, recommendationList := range clusterMap {
		version, err := storage.ReadReportInfoForCluster(orgID, cluster)
		if err != nil {
			continue
		}

		recommendationList.Meta = ctypes.ClusterMetadata{Version: version}
		clusterMap[cluster] = recommendationList
	}
}
