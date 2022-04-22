package migration

import (
	"database/sql"

	"github.com/RedHatInsights/insights-results-aggregator/types"
)

var mig0023AddReportInfoTable = Migration{
	StepUp: func(tx *sql.Tx, _ types.DBDriver) error {
		_, err := tx.Exec(`
		CREATE TABLE report_info (
			org_id VARCHAR NOT NULL,
			cluster_id VARCHAR NOT NULL,
			version_info VARCHAR NOT NULL,
			PRIMARY KEY(org_id, cluster_id)
		)`)
		return err
	},
	StepDown: func(tx *sql.Tx, _ types.DBDriver) error {
		_, err := tx.Exec(`
			DROP TABLE report_info;
		`)
		return err
	},
}
