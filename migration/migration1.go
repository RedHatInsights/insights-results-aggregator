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
)

var mig1 = Migration{
	StepUp: func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			CREATE TABLE report (
				org_id          INTEGER NOT NULL,
				cluster         VARCHAR NOT NULL UNIQUE,
				report          VARCHAR NOT NULL,
				reported_at     TIMESTAMP,
				last_checked_at TIMESTAMP,
				PRIMARY KEY(org_id, cluster)
			)`)
		return err
	},
	StepDown: func(tx *sql.Tx) error {
		_, err := tx.Exec(`DROP TABLE report`)
		return err
	},
}
