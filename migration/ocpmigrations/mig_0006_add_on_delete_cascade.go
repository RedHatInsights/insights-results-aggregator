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

package ocpmigrations

import (
	"github.com/RedHatInsights/insights-results-aggregator/migration"
)

var mig0006AddOnDeleteCascade = migration.NewUpdateTableMigration(
	ruleErrorKeyTable,
	`
		CREATE TABLE rule_error_key (
			"error_key"     VARCHAR NOT NULL,
			"rule_module"   VARCHAR NOT NULL,
			"condition"     VARCHAR NOT NULL,
			"description"   VARCHAR NOT NULL,
			"impact"        INTEGER NOT NULL,
			"likelihood"    INTEGER NOT NULL,
			"publish_date"  TIMESTAMP NOT NULL,
			"active"        BOOLEAN NOT NULL,
			"generic"       VARCHAR NOT NULL,
			PRIMARY KEY("error_key", "rule_module"),
			CONSTRAINT fk_rule_error_key
        		FOREIGN KEY ("rule_module") REFERENCES rule("module")
		)
		`,
	nil,
	`
		CREATE TABLE rule_error_key (
			"error_key"     VARCHAR NOT NULL,
			"rule_module"   VARCHAR NOT NULL REFERENCES rule(module) ON DELETE CASCADE,
			"condition"     VARCHAR NOT NULL,
			"description"   VARCHAR NOT NULL,
			"impact"        INTEGER NOT NULL,
			"likelihood"    INTEGER NOT NULL,
			"publish_date"  TIMESTAMP NOT NULL,
			"active"        BOOLEAN NOT NULL,
			"generic"       VARCHAR NOT NULL,
			PRIMARY KEY("error_key", "rule_module"),
			CONSTRAINT fk_rule_error_key
        		FOREIGN KEY ("rule_module") REFERENCES rule("module") ON DELETE CASCADE
		)
	`,
)
