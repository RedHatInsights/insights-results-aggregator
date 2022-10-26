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

/*
	migration4 adds foreign keys to cluster_rule_user_feedback
*/

var mig0004ModifyClusterRuleUserFeedback = NewUpdateTableMigration(
	clusterRuleUserFeedbackTable,
	`
		CREATE TABLE cluster_rule_user_feedback (
			cluster_id VARCHAR NOT NULL,
			rule_id VARCHAR NOT NULL,
			user_id VARCHAR NOT NULL,
			message VARCHAR NOT NULL,
			user_vote SMALLINT NOT NULL,
			added_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,

			PRIMARY KEY(cluster_id, rule_id, user_id)
		);
	`,
	nil,
	`
		CREATE TABLE cluster_rule_user_feedback (
			cluster_id VARCHAR NOT NULL,
			rule_id VARCHAR NOT NULL,
			user_id VARCHAR NOT NULL,
			message VARCHAR NOT NULL,
			user_vote SMALLINT NOT NULL,
			added_at TIMESTAMP NOT NULL,
			updated_at TIMESTAMP NOT NULL,

			PRIMARY KEY(cluster_id, rule_id, user_id),
			FOREIGN KEY (cluster_id)
				REFERENCES report(cluster)
				ON DELETE CASCADE,
			FOREIGN KEY (rule_id)
				REFERENCES rule(module)
				ON DELETE CASCADE
		);
	`,
)
