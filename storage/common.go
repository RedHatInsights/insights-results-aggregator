/*
Copyright © 2024 Red Hat, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
*/

package storage

import (
	"database/sql"
	"time"

	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/rs/zerolog/log"
)

func reportExists(
	tx *sql.Tx,
	tableName string,
	orgID types.OrgID,
	clusterName types.ClusterName,
	report types.ClusterReport,
	lastCheckedTime time.Time) (bool, error) {
	// Check if there is a more recent report for the cluster already in the database.
	rows, err := tx.Query(
		"SELECT last_checked_at FROM $1 WHERE org_id = $2 AND cluster = $3 AND last_checked_at > $4;",
		tableName, orgID, clusterName, lastCheckedTime)
	err = types.ConvertDBError(err, []interface{}{orgID, clusterName})
	if err != nil {
		log.Error().Err(err).Msg("Unable to look up the most recent report in the database")
		return false, err
	}

	defer closeRows(rows)

	// If there is one, print a warning and discard the report (don't update it).
	if rows.Next() {
		log.Warn().Msgf("Database already contains report for organization %d and cluster name %s more recent than %v",
			orgID, clusterName, lastCheckedTime)
		return true, nil
	}

	return false, nil
}
