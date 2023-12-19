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

package ocpmigrations

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

var mig0013AddRuleHitTable = migration.Migration{
	StepUp: func(tx *sql.Tx, driver types.DBDriver) error {
		_, err := tx.Exec(`
		CREATE TABLE rule_hit (
			org_id          INTEGER NOT NULL,
			cluster_id      VARCHAR NOT NULL,
			rule_fqdn       VARCHAR NOT NULL,
			error_key       VARCHAR NOT NULL,
			template_data   VARCHAR NOT NULL,
			PRIMARY KEY(cluster_id, org_id, rule_fqdn, error_key)
		)`)
		if err != nil {
			return err
		}

		if driver != types.DBDriverPostgres {
			// if sqlite, just ignore the actual migration cuz sqlite is too stupid for that
			return nil
		}

		_, err = tx.Exec(`
			DECLARE report_cursor CURSOR FOR
			SELECT
				org_id, cluster, report, reported_at, last_checked_at
			FROM
				report
		`)
		if err != nil {
			return err
		}

		for {
			var (
				orgID         types.OrgID
				clusterID     types.ClusterName
				report        types.ClusterReport
				reportedAt    time.Time
				lastCheckedAt time.Time
			)

			err := tx.QueryRow("FETCH NEXT FROM report_cursor").
				Scan(&orgID, &clusterID, &report, &reportedAt, &lastCheckedAt)
			if err == sql.ErrNoRows {
				break
			} else if err != nil {
				return err
			}

			err = writeRulesFromReportToRuleHit(tx, orgID, clusterID, report)
			if err != nil {
				return err
			}
		}

		_, err = tx.Exec("CLOSE report_cursor")
		return err
	},
	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {
		_, err := tx.Exec(`
			DROP TABLE rule_hit
		`)
		return err
	},
}

func writeRulesFromReportToRuleHit(
	tx *sql.Tx,
	orgID types.OrgID,
	clusterID types.ClusterName,
	stringReport types.ClusterReport,
) error {
	var report types.ReportRules

	err := json.Unmarshal([]byte(stringReport), &report)
	if err != nil {
		return err
	}

	for _, rule := range report.HitRules {
		var templateData []byte

		if templateDataStr, ok := rule.TemplateData.(string); ok {
			templateData = []byte(templateDataStr)
		} else {
			templateData, err = json.Marshal(rule.TemplateData)
			if err != nil {
				return err
			}
		}

		_, err = tx.Exec(`
			INSERT INTO rule_hit (
				org_id, cluster_id, rule_fqdn, error_key, template_data
			) VALUES ($1, $2, $3, $4, $5)
		`, orgID, clusterID, rule.Module, rule.ErrorKey, string(templateData))
		if err != nil {
			return err
		}
	}

	return nil
}
