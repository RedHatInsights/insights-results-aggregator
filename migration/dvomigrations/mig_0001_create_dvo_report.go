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

package dvomigrations

import (
	"database/sql"

	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

var mig0001CreateDVOReport = migration.Migration{
	StepUp: func(tx *sql.Tx, _ types.DBDriver) error {
		_, err := tx.Exec(`
			CREATE TABLE dvo.dvo_report (
				org_id          INTEGER NOT NULL,
				cluster_id      VARCHAR NOT NULL,
				namespace_id    VARCHAR NOT NULL,
				namespace_name  VARCHAR,
				report          TEXT,
				recommendations INTEGER NOT NULL,
				objects         INTEGER NOT NULL,
				reported_at     TIMESTAMP,
				last_checked_at TIMESTAMP,
				PRIMARY KEY(org_id, cluster_id, namespace_id)
			);
		`)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`
			COMMENT ON TABLE dvo.dvo_report IS 'This table is used as a cache for DVO reports. Only the latest report for a given cluster is stored.';
			COMMENT ON COLUMN dvo.dvo_report.org_id IS 'organization ID';
			COMMENT ON COLUMN dvo.dvo_report.cluster_id IS 'cluster UUID';
			COMMENT ON COLUMN dvo.dvo_report.namespace_id IS 'namespace UUID (always set)';
			COMMENT ON COLUMN dvo.dvo_report.namespace_name IS 'namespace name (might be null - not set)';
			COMMENT ON COLUMN dvo.dvo_report.report IS 'report structure stored in JSON format';
			COMMENT ON COLUMN dvo.dvo_report.recommendations IS 'number of recommendations stored in report';
			COMMENT ON COLUMN dvo.dvo_report.objects IS 'number of objects stored in report';
			COMMENT ON COLUMN dvo.dvo_report.reported_at IS 'timestamp, same meaning as in report table';
			COMMENT ON COLUMN dvo.dvo_report.last_checked_at IS 'timestamp, same meaning as in report table';
		`)

		return err
	},
	StepDown: func(tx *sql.Tx, _ types.DBDriver) error {
		_, err := tx.Exec(`DROP TABLE dvo.dvo_report;`)
		return err
	},
}
