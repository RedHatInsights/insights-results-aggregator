/*
Copyright © 2023 Red Hat, Inc.

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
	"fmt"

	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/types"
	"github.com/rs/zerolog/log"
)

// PostgreSQL database driver

// Storage represents an interface to almost any database or storage system
type Storage interface {
	Init() error
	Close() error
	GetConnection() *sql.DB
	GetMigrations() []migration.Migration
	GetDBDriverType() types.DBDriver
	GetDBSchema() migration.Schema
	GetMaxVersion() migration.Version
	MigrateToLatest() error
}

// TableInfo contains information about a table for orgID updates
type TableInfo struct {
	TableName     string
	ClusterColumn string
}

// checkOrgIDForCluster checks if there are existing records for the cluster
// and returns the current orgID if found.
func checkOrgIDForCluster(
	tx *sql.Tx,
	clusterName types.ClusterName,
	tablesToCheck []TableInfo,
) (currentOrgID types.OrgID, hasExistingRecord bool, err error) {
	for _, table := range tablesToCheck {
		// #nosec G201 - table/column names are from predefined TableInfo, not user input
		query := fmt.Sprintf("SELECT org_id FROM %s WHERE %s = $1 LIMIT 1", table.TableName, table.ClusterColumn)
		err = tx.QueryRow(query, clusterName).Scan(&currentOrgID)
		if err != nil && err != sql.ErrNoRows {
			return 0, false, fmt.Errorf("failed to check current org_id in %s table: %w", table.TableName, err)
		}
		if err != sql.ErrNoRows {
			hasExistingRecord = true
			return currentOrgID, hasExistingRecord, nil
		}
	}
	return 0, false, nil
}

// updateOrgIDInTables updates org_id in all specified tables for a given cluster.
func updateOrgIDInTables(
	tx *sql.Tx,
	newOrgID types.OrgID,
	clusterName types.ClusterName,
	tables []TableInfo,
) error {
	for _, tbl := range tables {
		// #nosec G201 - table/column names are from predefined TableInfo, not user input
		updateQuery := fmt.Sprintf("UPDATE %s SET org_id = $1 WHERE %s = $2", tbl.TableName, tbl.ClusterColumn)
		result, err := tx.Exec(updateQuery, newOrgID, clusterName)
		if err != nil {
			log.Warn().Err(err).Str(tableNameKey, tbl.TableName).Str(clusterKey, string(clusterName)).Msg("Failed to update org_id")
			// Continue with other tables even if one fails
			continue
		}

		rowsAffected, err := result.RowsAffected()
		if err == nil && rowsAffected > 0 {
			log.Debug().Int64(rowsAffectedKey, rowsAffected).Str(tableNameKey, tbl.TableName).Str(clusterKey, string(clusterName)).Msg("Updated rows in table")
		}
	}

	return nil
}
