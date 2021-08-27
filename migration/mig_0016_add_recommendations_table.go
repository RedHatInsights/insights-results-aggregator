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

package migration

import (
	"database/sql"
	"encoding/json"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

var mig0016AddRecommendationsTable = Migration{
	StepUp: func(tx *sql.Tx, driver types.DBDriver) error {
		_, err := tx.Exec(`
		CREATE TABLE recommendations (
			cluster_id      VARCHAR NOT NULL,
			rule_fqdn       VARCHAR NOT NULL,
			error_key		VARCHAR NOT NULL,
			PRIMARY KEY(cluster_id, rule_fqdn, error_key)
		)`)
		if err != nil {
			return err
		}

		if driver != types.DBDriverPostgres {
			// if sqlite, just ignore the actual migration
			return nil
		}

		_, err = tx.Exec(`
			DECLARE report_cursor CURSOR FOR
			SELECT
				cluster, report
			FROM
				report
		`)
		if err != nil {
			return err
		}

		for {
			var (
				clusterID     types.ClusterName
				report        types.ClusterReport
			)

			err := tx.QueryRow("FETCH NEXT FROM report_cursor").
				Scan(&clusterID, &report)
			if err == sql.ErrNoRows {
				break
			} else if err != nil {
				return err
			}

			err = writeRulesFromReportToRecommendations(tx, clusterID, report)
			if err != nil {
				return err
			}
		}

		_, err = tx.Exec("CLOSE report_cursor")
		return err
	},
	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {
		_, err := tx.Exec(`
			DROP TABLE recommendations
		`)
		return err
	},
}

func writeRulesFromReportToRecommendations(
	tx *sql.Tx,
	clusterID types.ClusterName,
	stringReport types.ClusterReport,
) error {
	var report types.ReportRules

	err := json.Unmarshal([]byte(stringReport), &report)
	if err != nil {
		return err
	}

	for _, rule := range report.HitRules {
		_, err = tx.Exec(`
			INSERT OR IGNORE INTO recommendations (
				cluster_id, rule_fqdn, error_key
			) VALUES ($1, $2, $3)
		`, clusterID, rule.Module, rule.ErrorKey)
		if err != nil {
			return err
		}
	}

	return nil
}
