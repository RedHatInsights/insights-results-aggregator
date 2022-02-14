package migration

import (
	"database/sql"

	"github.com/RedHatInsights/insights-results-aggregator/types"
)

var mig0021AddGatheredAtToReport = Migration{
	StepUp: func(tx *sql.Tx, _ types.DBDriver) error {
		_, err := tx.Exec(`ALTER TABLE report ADD COLUMN gathered_at TIMESTAMP`)
		return err
	},
	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {
		// Remove the gathering_time column
		if driver == types.DBDriverPostgres {
			_, err := tx.Exec(`
				ALTER TABLE report DROP COLUMN IF EXISTS gathered_at;
			`)
			return err
		} else if driver == types.DBDriverSQLite3 {
			// return mig0021ReportUpdateSQLite.StepDown(tx, driver)
			// Disable de foreign key and recreate it
			_, err := tx.Exec(`
			    ALTER TABLE report DROP gathered_at;
			`)
			// ALTER TABLE report RENAME TO report_tmp;
			// CREATE TABLE report AS SELECT org_id, cluster, report, reported_at, last_checked_at, kafka_offset FROM report_tmp;
			// DROP TABLE report_tmp;
			//`)
			return err
		}
		return nil
	},
}

var mig0021ReportUpdateSQLite = NewUpdateTableMigration(
	"report",
	`
	CREATE TABLE report (
		org_id	INTEGER NOT NULL,
		cluster	VARCHAR NOT NULL UNIQUE,
		report	VARCHAR NOT NULL,
		reported_at	TIMESTAMP,
		last_checked_at	TIMESTAMP,
		kafka_offset	BIGINT NOT NULL DEFAULT 0,
		PRIMARY KEY(org_id,cluster)
	)
	`,
	[]string{"org_id", "cluster", "report", "reported_at", "last_checked_at", "kafka_offset"},
	`
	CREATE TABLE "report" (
		"org_id"	INTEGER NOT NULL,
		"cluster"	VARCHAR NOT NULL UNIQUE,
		"report"	VARCHAR NOT NULL,
		"reported_at"	TIMESTAMP,
		"last_checked_at"	TIMESTAMP,
		"kafka_offset"	BIGINT NOT NULL DEFAULT 0,
		"gathered_at"	TIMESTAMP,
		PRIMARY KEY("org_id","cluster")
	);
	`,
)
