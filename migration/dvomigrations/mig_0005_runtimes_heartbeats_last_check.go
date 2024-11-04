package dvomigrations

import (
	"database/sql"

	"github.com/RedHatInsights/insights-results-aggregator/migration"
	"github.com/RedHatInsights/insights-results-aggregator/types"
)

var mig0005CreateRuntimesHeartbeats = migration.Migration{
	StepUp: func(tx *sql.Tx, _ types.DBDriver) error {
		_, err := tx.Exec(`
			CREATE TABLE dvo.runtimes_heartbeats (
				instance_id     VARCHAR NOT NULL,
				last_checked_at TIMESTAMP,
				PRIMARY KEY(instance_id)
			);
		`)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`
			COMMENT ON TABLE dvo.runtimes_heartbeats IS 'This table is used to store information of when the hearbeats was last received.';
			COMMENT ON COLUMN dvo.runtimes_heartbeats.instance_id IS 'instance ID';
			COMMENT ON COLUMN dvo.runtimes_heartbeats.last_checked_at IS 'timestamp of the received heartbeat';
		`)

		return err
	},
	StepDown: func(tx *sql.Tx, _ types.DBDriver) error {
		_, err := tx.Exec(`DROP TABLE dvo.runtimes_heartbeats;`)
		return err
	},
}
