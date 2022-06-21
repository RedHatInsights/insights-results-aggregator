// Copyright 2022 Red Hat, Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
