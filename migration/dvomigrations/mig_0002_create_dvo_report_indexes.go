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

var mig0002CreateDVOReportIndexes = migration.Migration{
	StepUp: func(tx *sql.Tx, _ types.DBDriver) error {
		_, err := tx.Exec(`
			CREATE INDEX report_org_id_idx ON dvo.dvo_report USING HASH (org_id);
			CREATE INDEX report_org_id_cluster_id_idx ON dvo.dvo_report (org_id, cluster_id);
		`)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`
			COMMENT ON INDEX dvo.report_org_id_idx IS 'for Workload page we need to be able to retrieve all records for given organization';
			COMMENT ON INDEX dvo.report_org_id_cluster_id_idx IS 'for Namespace view';
		`)
		return err
	},
	StepDown: func(tx *sql.Tx, _ types.DBDriver) error {
		_, err := tx.Exec(`
			DROP INDEX dvo.report_org_id_idx;
			DROP INDEX dvo.report_org_id_cluster_id_idx;
		`)
		return err
	},
}
