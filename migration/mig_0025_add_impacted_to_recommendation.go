package migration

import (
	"database/sql"
	"fmt"

	"github.com/RedHatInsights/insights-results-aggregator/types"
)

var mig0025AddImpactedToRecommendation = Migration{
	StepUp: func(tx *sql.Tx, _ types.DBDriver) error {
		_, err := tx.Exec(`ALTER TABLE recommendation ADD COLUMN impacted_since TIMESTAMP WITHOUT TIME ZONE`)
		return err
	},
	StepDown: func(tx *sql.Tx, driver types.DBDriver) error {
		// Remove the gathering_time column
		if driver == types.DBDriverPostgres {
			_, err := tx.Exec(`
				ALTER TABLE recommendation DROP COLUMN IF EXISTS impacted_since;
			`)
			return err
		}

		return fmt.Errorf(driverUnsupportedErr, driver)
	},
}
