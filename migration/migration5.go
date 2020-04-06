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

var mig5 = Migration{
	StepUp: func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			CREATE TABLE consumer_error (
				topic           VARCHAR NOT NULL,
				partition       INTEGER NOT NULL,
				topic_offset    INTEGER NOT NULL,
				key             VARCHAR,
				produced_at     TIMESTAMP NOT NULL,
				consumed_at     TIMESTAMP NOT NULL,
				message         VARCHAR,
				error           VARCHAR NOT NULL,

				PRIMARY KEY(topic, partition, topic_offset)
			)
		`)
		return err
	},
	StepDown: func(tx *sql.Tx) error {
		_, err := tx.Exec(`DROP TABLE consumer_error`)
		return err
	},
}
