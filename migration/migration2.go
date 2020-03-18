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

var mig2 = Migration{
	StepUp: func(tx *sql.Tx) error {
		_, err := tx.Exec(`
			CREATE TABLE rule (
				"module"        VARCHAR PRIMARY KEY,
				"name"          VARCHAR NOT NULL,
				"summary"       VARCHAR NOT NULL,
				"reason"        VARCHAR NOT NULL,
				"resolution"    VARCHAR NOT NULL,
				"more_info"     VARCHAR NOT NULL
			)`)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`
			CREATE TABLE rule_error_key (
				"error_key"     VARCHAR NOT NULL,
				"rule_module"   VARCHAR NOT NULL REFERENCES rule(module),
				"condition"     VARCHAR NOT NULL,
				"description"   VARCHAR NOT NULL,
				"impact"        INTEGER NOT NULL,
				"likelihood"    INTEGER NOT NULL,
				"publish_date"  TIMESTAMP NOT NULL,
				"active"        BOOLEAN NOT NULL,
				"generic"       VARCHAR NOT NULL,
				PRIMARY KEY("error_key", "rule_module")
			)`)
		return err
	},
	StepDown: func(tx *sql.Tx) error {
		_, err := tx.Exec(`DROP TABLE rule_error_key`)
		if err != nil {
			return err
		}

		_, err = tx.Exec(`DROP TABLE rule`)
		return err
	},
}
