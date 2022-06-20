package migration

import (
	"database/sql"
	"fmt"

	"github.com/RedHatInsights/insights-results-aggregator/types"
)

var mig0024AddTimestampToRuleHit = Migration{
	StepUp: func(tx *sql.Tx, _ types.DBDriver) error {
		_, err := tx.Exec(`ALTER TABLE rule_hit ADD COLUMN created_at TIMESTAMP`)
		return err
	},
	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {
		// Remove the gathering_time column
		if driver == types.DBDriverPostgres {
			_, err := tx.Exec(`
				ALTER TABLE rule_hit DROP COLUMN IF EXISTS created_at;
			`)
			return err
		}

		return fmt.Errorf("%v driver is not supported", driver)
	},
}
